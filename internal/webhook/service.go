package webhook

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
}

type ListResponse struct {
	Webhooks   []map[string]any
	RawBody    []byte
	Profile    string
	RequestID  string
	Pagination output.Pagination
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
	Webhook   map[string]any
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
	Webhook   map[string]any
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
	Webhook   map[string]any
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

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("page", strconv.Itoa(req.Page))
	query.Set("per_page", strconv.Itoa(req.PerPage))

	var webhooks []map[string]any
	resp, err := client.GetJSON(ctx, "/webhooks.json", query, &webhooks)
	if err != nil {
		return nil, err
	}
	return &ListResponse{
		Webhooks:  webhooks,
		RawBody:   resp.RawBody,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
		Pagination: output.Pagination{
			Page:     req.Page,
			PerPage:  req.PerPage,
			Returned: len(webhooks),
			HasNext:  len(webhooks) == req.PerPage,
		},
	}, nil
}

func (s *Service) Get(ctx context.Context, req GetRequest) (*GetResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "webhook ID is required", "pass --id <webhook-id>")
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	var webhook map[string]any
	resp, err := client.GetJSON(ctx, fmt.Sprintf("/webhooks/%s.json", url.PathEscape(req.ID)), nil, &webhook)
	if err != nil {
		return nil, err
	}
	return &GetResponse{
		Webhook:   webhook,
		RawBody:   resp.RawBody,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
	}, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*CreateResponse, error) {
	if req.Input == nil {
		return nil, output.Usage("missing_input", "webhook input is required", "pass --input -|@file.json|'{...}'")
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	payload := cloneMap(req.Input)
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodPost, "/webhooks.json", nil, payload)
		return &CreateResponse{Profile: resolved.Name, DryRun: &plan}, nil
	}

	var webhook map[string]any
	resp, err := client.PostJSON(ctx, "/webhooks.json", payload, &webhook)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{
		Webhook:   webhook,
		RawBody:   resp.RawBody,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
	}, nil
}

func (s *Service) Update(ctx context.Context, req UpdateRequest) (*UpdateResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "webhook ID is required", "pass --id <webhook-id>")
	}
	if req.Input == nil {
		return nil, output.Usage("missing_input", "webhook input is required", "pass --input -|@file.json|'{...}'")
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/webhooks/%s.json", url.PathEscape(req.ID))
	payload := cloneMap(req.Input)
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodPut, path, nil, payload)
		return &UpdateResponse{Profile: resolved.Name, DryRun: &plan}, nil
	}

	var webhook map[string]any
	resp, err := client.PutJSON(ctx, path, payload, &webhook)
	if err != nil {
		return nil, err
	}
	return &UpdateResponse{
		Webhook:   webhook,
		RawBody:   resp.RawBody,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
	}, nil
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (*DeleteResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "webhook ID is required", "pass --id <webhook-id>")
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/webhooks/%s.json", url.PathEscape(req.ID))
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodDelete, path, nil, nil)
		return &DeleteResponse{ID: req.ID, Profile: resolved.Name, DryRun: &plan}, nil
	}

	resp, err := client.DeleteJSON(ctx, path, nil, nil)
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

func (s *Service) resolveClient(configPath, profile string, env config.Env, timeout time.Duration, maxRetries int) (*config.ResolvedProfile, *transport.Client, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	resolved, err := config.Resolve(configPath, env, profile, s.store)
	if err != nil {
		return nil, nil, err
	}
	client, err := transport.NewClient(resolved.URL, resolved.Token, timeout, maxRetries, nil)
	if err != nil {
		return nil, nil, err
	}
	return resolved, client, nil
}

func cloneMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	out := make(map[string]any, len(value))
	for key, child := range value {
		out[key] = cloneValue(child)
	}
	return out
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMap(typed)
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
