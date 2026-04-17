package payment

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sixers/fakturownia-cli/internal/config"
	"github.com/sixers/fakturownia-cli/internal/output"
	"github.com/sixers/fakturownia-cli/internal/transport"
)

type Service struct {
	store config.TokenStore
}

type ListRequest struct {
	ConfigPath string
	Profile    string
	Env        config.Env
	Timeout    time.Duration
	MaxRetries int
	Page       int
	PerPage    int
	Include    []string
}

type ListResponse struct {
	Payments    []map[string]any
	RawBody     []byte
	Profile     string
	RequestID   string
	Pagination  output.Pagination
	IncludeUsed []string
}

func (r *ListResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

type GetRequest struct {
	ConfigPath string
	Profile    string
	Env        config.Env
	Timeout    time.Duration
	MaxRetries int
	ID         string
}

type GetResponse struct {
	Payment   map[string]any
	RawBody   []byte
	Profile   string
	RequestID string
}

func (r *GetResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

type CreateRequest struct {
	ConfigPath string
	Profile    string
	Env        config.Env
	Timeout    time.Duration
	MaxRetries int
	Input      map[string]any
	DryRun     bool
}

type CreateResponse struct {
	Payment   map[string]any
	RawBody   []byte
	Profile   string
	RequestID string
	DryRun    *transport.RequestPlan
}

func (r *CreateResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

type UpdateRequest struct {
	ConfigPath string
	Profile    string
	Env        config.Env
	Timeout    time.Duration
	MaxRetries int
	ID         string
	Input      map[string]any
	DryRun     bool
}

type UpdateResponse struct {
	Payment   map[string]any
	RawBody   []byte
	Profile   string
	RequestID string
	DryRun    *transport.RequestPlan
}

func (r *UpdateResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

type DeleteRequest struct {
	ConfigPath string
	Profile    string
	Env        config.Env
	Timeout    time.Duration
	MaxRetries int
	ID         string
	DryRun     bool
}

type DeleteResponse struct {
	ID        string                 `json:"id"`
	Deleted   bool                   `json:"deleted"`
	Profile   string                 `json:"profile"`
	RequestID string                 `json:"request_id,omitempty"`
	DryRun    *transport.RequestPlan `json:"dry_run,omitempty"`
	RawBody   []byte                 `json:"-"`
}

func (r *DeleteResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

func NewService(store config.TokenStore) *Service {
	return &Service{store: store}
}

func (s *Service) List(ctx context.Context, req ListRequest) (*ListResponse, error) {
	if req.Page < 1 {
		return nil, output.Usage("invalid_page", "--page must be at least 1", "pass --page 1 or higher")
	}
	if req.PerPage == 0 {
		req.PerPage = 25
	}
	if req.PerPage < 1 || req.PerPage > 100 {
		return nil, output.Usage("invalid_per_page", "--per-page must be between 1 and 100", "pass a value like --per-page 25")
	}
	include, err := normalizeInclude(req.Include)
	if err != nil {
		return nil, err
	}

	resolved, httpClient, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("page", strconv.Itoa(req.Page))
	query.Set("per_page", strconv.Itoa(req.PerPage))
	if len(include) > 0 {
		query.Set("include", strings.Join(include, ","))
	}

	var payments []map[string]any
	resp, err := httpClient.GetJSON(ctx, "/banking/payments.json", query, &payments)
	if err != nil {
		return nil, err
	}
	return &ListResponse{
		Payments:    payments,
		RawBody:     resp.RawBody,
		Profile:     resolved.Name,
		RequestID:   resp.RequestID,
		IncludeUsed: include,
		Pagination: output.Pagination{
			Page:     req.Page,
			PerPage:  req.PerPage,
			Returned: len(payments),
			HasNext:  len(payments) == req.PerPage,
		},
	}, nil
}

func (s *Service) Get(ctx context.Context, req GetRequest) (*GetResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "payment ID is required", "pass --id <payment-id>")
	}

	resolved, httpClient, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	var payment map[string]any
	resp, err := httpClient.GetJSON(ctx, fmt.Sprintf("/banking/payments/%s.json", url.PathEscape(req.ID)), nil, &payment)
	if err != nil {
		return nil, err
	}
	return &GetResponse{
		Payment:   payment,
		RawBody:   resp.RawBody,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
	}, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*CreateResponse, error) {
	if err := validateInput(req.Input); err != nil {
		return nil, err
	}

	resolved, httpClient, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	payload := wrapPayload(req.Input)
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodPost, "/banking/payments.json", nil, payload)
		return &CreateResponse{Profile: resolved.Name, DryRun: &plan}, nil
	}

	var payment map[string]any
	resp, err := httpClient.PostJSON(ctx, "/banking/payments.json", payload, &payment)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{
		Payment:   payment,
		RawBody:   resp.RawBody,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
	}, nil
}

func (s *Service) Update(ctx context.Context, req UpdateRequest) (*UpdateResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "payment ID is required", "pass --id <payment-id>")
	}
	if err := validateInput(req.Input); err != nil {
		return nil, err
	}

	resolved, httpClient, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/banking/payments/%s.json", url.PathEscape(req.ID))
	payload := wrapPayload(req.Input)
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodPatch, path, nil, payload)
		return &UpdateResponse{Profile: resolved.Name, DryRun: &plan}, nil
	}

	var payment map[string]any
	resp, err := httpClient.PatchJSON(ctx, path, payload, &payment)
	if err != nil {
		return nil, err
	}
	return &UpdateResponse{
		Payment:   payment,
		RawBody:   resp.RawBody,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
	}, nil
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (*DeleteResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "payment ID is required", "pass --id <payment-id>")
	}

	resolved, httpClient, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/banking/payments/%s.json", url.PathEscape(req.ID))
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodDelete, path, nil, nil)
		return &DeleteResponse{ID: req.ID, Profile: resolved.Name, DryRun: &plan}, nil
	}

	resp, err := httpClient.DeleteJSON(ctx, path, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DeleteResponse{
		ID:        req.ID,
		Deleted:   true,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
		RawBody:   resp.RawBody,
	}, nil
}

func normalizeInclude(values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			include := strings.TrimSpace(part)
			if include == "" {
				continue
			}
			if include != "invoices" {
				return nil, output.Usage("invalid_include", fmt.Sprintf("unsupported payment include %q", include), "use --include invoices")
			}
			out = append(out, include)
		}
	}
	return out, nil
}

func (s *Service) resolveClient(configPath, profile string, env config.Env, timeout time.Duration, maxRetries int) (*config.ResolvedProfile, *transport.Client, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	resolved, err := config.Resolve(configPath, env, profile, s.store)
	if err != nil {
		return nil, nil, err
	}
	httpClient, err := transport.NewClient(resolved.URL, resolved.Token, timeout, maxRetries, nil)
	if err != nil {
		return nil, nil, err
	}
	return resolved, httpClient, nil
}

func validateInput(input map[string]any) error {
	if input == nil {
		return output.Usage("missing_input", "payment input is required", "pass --input -|@file.json|'{...}'")
	}
	return nil
}

func wrapPayload(input map[string]any) map[string]any {
	return map[string]any{
		"banking_payment": cloneValue(input).(map[string]any),
	}
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = cloneValue(child)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for idx, child := range typed {
			out[idx] = cloneValue(child)
		}
		return out
	default:
		return typed
	}
}
