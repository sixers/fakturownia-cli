package warehouseaction

import (
	"context"
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
	ConfigPath            string
	Profile               string
	Env                   config.Env
	Timeout               time.Duration
	MaxRetries            int
	Page                  int
	PerPage               int
	WarehouseID           string
	Kind                  string
	ProductID             string
	DateFrom              string
	DateTo                string
	FromWarehouseDocument string
	ToWarehouseDocument   string
	WarehouseDocumentID   string
}

type ListResponse struct {
	WarehouseActions []map[string]any
	RawBody          []byte
	Profile          string
	RequestID        string
	Pagination       output.Pagination
}

func (r *ListResponse) GetProfile() string {
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
	setQueryIfPresent(query, "warehouse_id", req.WarehouseID)
	setQueryIfPresent(query, "kind", req.Kind)
	setQueryIfPresent(query, "product_id", req.ProductID)
	setQueryIfPresent(query, "date_from", req.DateFrom)
	setQueryIfPresent(query, "date_to", req.DateTo)
	setQueryIfPresent(query, "from_warehouse_document", req.FromWarehouseDocument)
	setQueryIfPresent(query, "to_warehouse_document", req.ToWarehouseDocument)
	setQueryIfPresent(query, "warehouse_document_id", req.WarehouseDocumentID)

	var warehouseActions []map[string]any
	resp, err := httpClient.GetJSON(ctx, "/warehouse_actions.json", query, &warehouseActions)
	if err != nil {
		return nil, err
	}
	return &ListResponse{
		WarehouseActions: warehouseActions,
		RawBody:          resp.RawBody,
		Profile:          resolved.Name,
		RequestID:        resp.RequestID,
		Pagination: output.Pagination{
			Page:     req.Page,
			PerPage:  req.PerPage,
			Returned: len(warehouseActions),
			HasNext:  len(warehouseActions) == req.PerPage,
		},
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

func setQueryIfPresent(query url.Values, key, value string) {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		query.Set(key, trimmed)
	}
}
