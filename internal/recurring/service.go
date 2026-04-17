package recurring

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
}

type ListResponse struct {
	Recurrings []map[string]any
	RawBody    []byte
	Profile    string
	RequestID  string
}

func (r *ListResponse) GetProfile() string {
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
	Recurring map[string]any
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
	Recurring map[string]any
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
	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	var recurrings []map[string]any
	resp, err := client.GetJSON(ctx, "/recurrings.json", nil, &recurrings)
	if err != nil {
		return nil, err
	}
	return &ListResponse{
		Recurrings: recurrings,
		RawBody:    resp.RawBody,
		Profile:    resolved.Name,
		RequestID:  resp.RequestID,
	}, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*CreateResponse, error) {
	if err := validateInput(req.Input); err != nil {
		return nil, err
	}
	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{"recurring": cloneValue(req.Input).(map[string]any)}
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodPost, "/recurrings.json", nil, payload)
		return &CreateResponse{Profile: resolved.Name, DryRun: &plan}, nil
	}

	var recurring map[string]any
	resp, err := client.PostJSON(ctx, "/recurrings.json", payload, &recurring)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{
		Recurring: recurring,
		RawBody:   resp.RawBody,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
	}, nil
}

func (s *Service) Update(ctx context.Context, req UpdateRequest) (*UpdateResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "recurring ID is required", "pass --id <recurring-id>")
	}
	if err := validateInput(req.Input); err != nil {
		return nil, err
	}
	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/recurrings/%s.json", url.PathEscape(req.ID))
	payload := map[string]any{"recurring": cloneValue(req.Input).(map[string]any)}
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodPut, path, nil, payload)
		return &UpdateResponse{Profile: resolved.Name, DryRun: &plan}, nil
	}

	var recurring map[string]any
	resp, err := client.PutJSON(ctx, path, payload, &recurring)
	if err != nil {
		return nil, err
	}
	return &UpdateResponse{
		Recurring: recurring,
		RawBody:   resp.RawBody,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
	}, nil
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (*DeleteResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "recurring ID is required", "pass --id <recurring-id>")
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/recurrings/%s.json", url.PathEscape(req.ID))
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

func validateInput(input map[string]any) error {
	if input == nil {
		return output.Usage("missing_input", "recurring input is required", "pass --input -|@file.json|'{...}'")
	}
	return nil
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
