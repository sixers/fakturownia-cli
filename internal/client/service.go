package client

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
	Name       string
	Email      string
	Shortcut   string
	TaxNo      string
	ExternalID string
}

type ListResponse struct {
	Clients    []map[string]any
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
	ExternalID string
}

type GetResponse struct {
	Client    map[string]any
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
	Client    map[string]any
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
	Client    map[string]any
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

	resolved, httpClient, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("page", strconv.Itoa(req.Page))
	query.Set("per_page", strconv.Itoa(req.PerPage))
	if req.Name != "" {
		query.Set("name", req.Name)
	}
	if req.Email != "" {
		query.Set("email", req.Email)
	}
	if req.Shortcut != "" {
		query.Set("shortcut", req.Shortcut)
	}
	if req.TaxNo != "" {
		query.Set("tax_no", req.TaxNo)
	}
	if req.ExternalID != "" {
		query.Set("external_id", req.ExternalID)
	}

	var clients []map[string]any
	resp, err := httpClient.GetJSON(ctx, "/clients.json", query, &clients)
	if err != nil {
		return nil, err
	}

	return &ListResponse{
		Clients:   clients,
		RawBody:   resp.RawBody,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
		Pagination: output.Pagination{
			Page:     req.Page,
			PerPage:  req.PerPage,
			Returned: len(clients),
			HasNext:  len(clients) == req.PerPage,
		},
	}, nil
}

func (s *Service) Get(ctx context.Context, req GetRequest) (*GetResponse, error) {
	hasID := strings.TrimSpace(req.ID) != ""
	hasExternalID := strings.TrimSpace(req.ExternalID) != ""
	if hasID == hasExternalID {
		return nil, output.Usage("invalid_client_selector", "exactly one of --id or --external-id is required", "pass either --id <client-id> or --external-id <external-id>")
	}

	resolved, httpClient, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	if hasID {
		var client map[string]any
		resp, err := httpClient.GetJSON(ctx, fmt.Sprintf("/clients/%s.json", url.PathEscape(req.ID)), nil, &client)
		if err != nil {
			return nil, err
		}
		return &GetResponse{
			Client:    client,
			RawBody:   resp.RawBody,
			Profile:   resolved.Name,
			RequestID: resp.RequestID,
		}, nil
	}

	query := url.Values{}
	query.Set("external_id", req.ExternalID)
	var clients []map[string]any
	resp, err := httpClient.GetJSON(ctx, "/clients.json", query, &clients)
	if err != nil {
		return nil, err
	}
	switch len(clients) {
	case 0:
		return nil, output.NotFound("client_not_found", fmt.Sprintf("no client matched external_id %q", req.ExternalID), "verify the external ID and retry").WithRawBody(resp.RawBody)
	case 1:
		return &GetResponse{
			Client:    clients[0],
			RawBody:   resp.RawBody,
			Profile:   resolved.Name,
			RequestID: resp.RequestID,
		}, nil
	default:
		return nil, output.Conflict("multiple_clients_matched", fmt.Sprintf("multiple clients matched external_id %q", req.ExternalID), "use `client list --external-id ...` to inspect matches or query by --id").WithRawBody(resp.RawBody)
	}
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
		plan := transport.PlanJSONRequest(http.MethodPost, "/clients.json", nil, payload)
		return &CreateResponse{Profile: resolved.Name, DryRun: &plan}, nil
	}

	var client map[string]any
	resp, err := httpClient.PostJSON(ctx, "/clients.json", payload, &client)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{
		Client:    client,
		RawBody:   resp.RawBody,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
	}, nil
}

func (s *Service) Update(ctx context.Context, req UpdateRequest) (*UpdateResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "client ID is required", "pass --id <client-id>")
	}
	if err := validateInput(req.Input); err != nil {
		return nil, err
	}

	resolved, httpClient, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/clients/%s.json", url.PathEscape(req.ID))
	payload := wrapPayload(req.Input)
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodPut, path, nil, payload)
		return &UpdateResponse{Profile: resolved.Name, DryRun: &plan}, nil
	}

	var client map[string]any
	resp, err := httpClient.PutJSON(ctx, path, payload, &client)
	if err != nil {
		return nil, err
	}
	return &UpdateResponse{
		Client:    client,
		RawBody:   resp.RawBody,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
	}, nil
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (*DeleteResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "client ID is required", "pass --id <client-id>")
	}

	resolved, httpClient, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/clients/%s.json", url.PathEscape(req.ID))
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodDelete, path, nil, nil)
		return &DeleteResponse{
			ID:      req.ID,
			Deleted: false,
			Profile: resolved.Name,
			DryRun:  &plan,
		}, nil
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
		return output.Usage("missing_input", "client input is required", "pass --input -|@file.json|'{...}'")
	}
	return nil
}

func wrapPayload(input map[string]any) map[string]any {
	return map[string]any{
		"client": cloneValue(input).(map[string]any),
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
