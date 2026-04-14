package invoice

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
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
	ConfigPath       string
	Profile          string
	Env              config.Env
	Timeout          time.Duration
	MaxRetries       int
	Page             int
	PerPage          int
	Period           string
	DateFrom         string
	DateTo           string
	IncludePositions bool
	ClientID         string
	InvoiceIDs       []string
	Number           string
	Kinds            []string
	SearchDateType   string
	Order            string
	Income           string
}

type ListResponse struct {
	Invoices   []map[string]any
	RawBody    []byte
	Profile    string
	RequestID  string
	Pagination output.Pagination
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
	Invoice   map[string]any
	RawBody   []byte
	Profile   string
	RequestID string
}

type DownloadRequest struct {
	ConfigPath  string
	Profile     string
	Env         config.Env
	Timeout     time.Duration
	MaxRetries  int
	ID          string
	Path        string
	Dir         string
	PrintOption string
}

type DownloadResponse struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	Bytes       int    `json:"bytes"`
	PrintOption string `json:"print_option,omitempty"`
	Profile     string `json:"profile"`
	RequestID   string `json:"request_id,omitempty"`
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
	if req.Period == "more" && (req.DateFrom == "" || req.DateTo == "") {
		return nil, output.Usage("missing_date_range", "--period more requires --date-from and --date-to", "pass both dates when using --period more")
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("page", strconv.Itoa(req.Page))
	query.Set("per_page", strconv.Itoa(req.PerPage))
	if req.Period != "" {
		query.Set("period", req.Period)
	}
	if req.DateFrom != "" {
		query.Set("date_from", req.DateFrom)
	}
	if req.DateTo != "" {
		query.Set("date_to", req.DateTo)
	}
	if req.IncludePositions {
		query.Set("include_positions", "true")
	}
	if req.ClientID != "" {
		query.Set("client_id", req.ClientID)
	}
	if len(req.InvoiceIDs) > 0 {
		query.Set("invoice_ids", strings.Join(req.InvoiceIDs, ","))
	}
	if req.Number != "" {
		query.Set("number", req.Number)
	}
	for _, kind := range req.Kinds {
		if trimmed := strings.TrimSpace(kind); trimmed != "" {
			query.Add("kinds[]", trimmed)
		}
	}
	if req.SearchDateType != "" {
		query.Set("search_date_type", req.SearchDateType)
	}
	if req.Order != "" {
		query.Set("order", req.Order)
	}
	if req.Income != "" {
		query.Set("income", req.Income)
	}

	var invoices []map[string]any
	resp, err := client.GetJSON(ctx, "/invoices.json", query, &invoices)
	if err != nil {
		return nil, err
	}
	return &ListResponse{
		Invoices:  invoices,
		RawBody:   resp.RawBody,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
		Pagination: output.Pagination{
			Page:     req.Page,
			PerPage:  req.PerPage,
			Returned: len(invoices),
			HasNext:  len(invoices) == req.PerPage,
		},
	}, nil
}

func (s *Service) Get(ctx context.Context, req GetRequest) (*GetResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "invoice ID is required", "pass --id <invoice-id>")
	}
	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	var invoice map[string]any
	resp, err := client.GetJSON(ctx, fmt.Sprintf("/invoices/%s.json", url.PathEscape(req.ID)), nil, &invoice)
	if err != nil {
		return nil, err
	}
	return &GetResponse{
		Invoice:   invoice,
		RawBody:   resp.RawBody,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
	}, nil
}

func (s *Service) Download(ctx context.Context, req DownloadRequest) (*DownloadResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "invoice ID is required", "pass --id <invoice-id>")
	}
	if req.Path != "" && req.Dir != "" {
		return nil, output.Usage("path_conflict", "--path and --dir cannot be used together", "pass either --path or --dir")
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	if req.PrintOption != "" {
		query.Set("print_option", req.PrintOption)
	}
	resp, err := client.GetBinary(ctx, fmt.Sprintf("/invoices/%s.pdf", url.PathEscape(req.ID)), query)
	if err != nil {
		return nil, err
	}

	targetPath, err := targetDownloadPath(req.ID, req.Path, req.Dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return nil, output.Internal(err, "create download directory")
	}
	tempFile, err := os.CreateTemp(filepath.Dir(targetPath), "fakturownia-*.pdf")
	if err != nil {
		return nil, output.Internal(err, "create temporary download file")
	}
	tempPath := tempFile.Name()
	if _, err := tempFile.Write(resp.RawBody); err != nil {
		_ = tempFile.Close()
		return nil, output.Internal(err, "write downloaded PDF")
	}
	if err := tempFile.Close(); err != nil {
		return nil, output.Internal(err, "close downloaded PDF")
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		return nil, output.Internal(err, "move downloaded PDF into place")
	}

	return &DownloadResponse{
		ID:          req.ID,
		Path:        targetPath,
		Bytes:       len(resp.RawBody),
		PrintOption: req.PrintOption,
		Profile:     resolved.Name,
		RequestID:   resp.RequestID,
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

func targetDownloadPath(invoiceID, explicitPath, dir string) (string, error) {
	if explicitPath != "" {
		return explicitPath, nil
	}
	if dir == "" {
		dir = "."
	}
	return filepath.Join(dir, fmt.Sprintf("invoice-%s.pdf", invoiceID)), nil
}
