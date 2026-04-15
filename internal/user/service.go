package user

import (
	"context"
	"net/http"
	"time"

	"github.com/sixers/fakturownia-cli/internal/config"
	"github.com/sixers/fakturownia-cli/internal/output"
	"github.com/sixers/fakturownia-cli/internal/transport"
)

type Service struct {
	store config.TokenStore
}

type CreateRequest struct {
	ConfigPath       string
	Profile          string
	Env              config.Env
	Timeout          time.Duration
	MaxRetries       int
	IntegrationToken string
	Input            map[string]any
	DryRun           bool
}

type CreateResponse struct {
	Response  map[string]any
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

func NewService(store config.TokenStore) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*CreateResponse, error) {
	if req.Input == nil {
		return nil, output.Usage("missing_input", "user input is required", "pass --input -|@file.json|'{...}'")
	}
	if req.IntegrationToken == "" {
		return nil, output.Usage("missing_integration_token", "integration token is required", "pass --integration-token <token>")
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"integration_token": req.IntegrationToken,
		"user":              cloneValue(req.Input).(map[string]any),
	}
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodPost, "/account/add_user.json", nil, payload)
		return &CreateResponse{Profile: resolved.Name, DryRun: &plan}, nil
	}

	var response map[string]any
	resp, err := client.PostJSON(ctx, "/account/add_user.json", payload, &response)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{
		Response:  response,
		RawBody:   resp.RawBody,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
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
