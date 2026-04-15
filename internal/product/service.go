package product

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
	ConfigPath  string
	Profile     string
	Env         config.Env
	Timeout     time.Duration
	MaxRetries  int
	Page        int
	PerPage     int
	DateFrom    string
	WarehouseID string
}

type ListResponse struct {
	Products   []map[string]any
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
	ConfigPath  string
	Profile     string
	Env         config.Env
	Timeout     time.Duration
	MaxRetries  int
	ID          string
	WarehouseID string
}

type GetResponse struct {
	Product   map[string]any
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
	Product   map[string]any
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
	Product   map[string]any
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
	if req.DateFrom != "" {
		query.Set("date_from", req.DateFrom)
	}
	if req.WarehouseID != "" {
		query.Set("warehouse_id", req.WarehouseID)
	}

	var products []map[string]any
	resp, err := httpClient.GetJSON(ctx, "/products.json", query, &products)
	if err != nil {
		return nil, err
	}
	return &ListResponse{
		Products:   products,
		RawBody:    resp.RawBody,
		Profile:    resolved.Name,
		RequestID:  resp.RequestID,
		Pagination: output.Pagination{Page: req.Page, PerPage: req.PerPage, Returned: len(products), HasNext: len(products) == req.PerPage},
	}, nil
}

func (s *Service) Get(ctx context.Context, req GetRequest) (*GetResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "product ID is required", "pass --id <product-id>")
	}
	resolved, httpClient, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	if req.WarehouseID != "" {
		query.Set("warehouse_id", req.WarehouseID)
	}

	var product map[string]any
	resp, err := httpClient.GetJSON(ctx, fmt.Sprintf("/products/%s.json", url.PathEscape(req.ID)), query, &product)
	if err != nil {
		return nil, err
	}
	return &GetResponse{
		Product:   product,
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
		plan := transport.PlanJSONRequest(http.MethodPost, "/products.json", nil, payload)
		return &CreateResponse{Profile: resolved.Name, DryRun: &plan}, nil
	}

	var product map[string]any
	resp, err := httpClient.PostJSON(ctx, "/products.json", payload, &product)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{
		Product:   product,
		RawBody:   resp.RawBody,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
	}, nil
}

func (s *Service) Update(ctx context.Context, req UpdateRequest) (*UpdateResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "product ID is required", "pass --id <product-id>")
	}
	if err := validateInput(req.Input); err != nil {
		return nil, err
	}

	resolved, httpClient, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/products/%s.json", url.PathEscape(req.ID))
	payload := wrapPayload(req.Input)
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodPut, path, nil, payload)
		return &UpdateResponse{Profile: resolved.Name, DryRun: &plan}, nil
	}

	var product map[string]any
	resp, err := httpClient.PutJSON(ctx, path, payload, &product)
	if err != nil {
		return nil, err
	}
	return &UpdateResponse{
		Product:   product,
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
	httpClient, err := transport.NewClient(resolved.URL, resolved.Token, timeout, maxRetries, nil)
	if err != nil {
		return nil, nil, err
	}
	return resolved, httpClient, nil
}

func validateInput(input map[string]any) error {
	if input == nil {
		return output.Usage("missing_input", "product input is required", "pass --input -|@file.json|'{...}'")
	}
	return nil
}

func wrapPayload(input map[string]any) map[string]any {
	return map[string]any{
		"product": cloneValue(input).(map[string]any),
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
