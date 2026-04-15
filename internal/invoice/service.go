package invoice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
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
	store      config.TokenStore
	httpClient *http.Client
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

func (r *ListResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

type GetRequest struct {
	ConfigPath        string
	Profile           string
	Env               config.Env
	Timeout           time.Duration
	MaxRetries        int
	ID                string
	Includes          []string
	AdditionalFields  []string
	CorrectionDetails string
}

type GetResponse struct {
	Invoice   map[string]any
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

func (r *DownloadResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

type CreateRequest struct {
	ConfigPath              string
	Profile                 string
	Env                     config.Env
	Timeout                 time.Duration
	MaxRetries              int
	Input                   map[string]any
	IdentifyOSS             bool
	FillDefaultDescriptions bool
	CorrectionPositions     string
	GovSaveAndSend          bool
	DryRun                  bool
}

type CreateResponse struct {
	Invoice   map[string]any
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
	ConfigPath              string
	Profile                 string
	Env                     config.Env
	Timeout                 time.Duration
	MaxRetries              int
	ID                      string
	Input                   map[string]any
	IdentifyOSS             bool
	FillDefaultDescriptions bool
	CorrectionPositions     string
	GovSaveAndSend          bool
	DryRun                  bool
}

type UpdateResponse struct {
	Invoice   map[string]any
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
	Response  map[string]any         `json:"response,omitempty"`
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

type SendEmailRequest struct {
	ConfigPath       string
	Profile          string
	Env              config.Env
	Timeout          time.Duration
	MaxRetries       int
	ID               string
	EmailTo          []string
	EmailCC          []string
	EmailPDF         bool
	UpdateBuyerEmail bool
	PrintOption      string
	DryRun           bool
}

type SendEmailResponse struct {
	ID        string                 `json:"id"`
	Sent      bool                   `json:"sent"`
	Response  map[string]any         `json:"response,omitempty"`
	Profile   string                 `json:"profile"`
	RequestID string                 `json:"request_id,omitempty"`
	DryRun    *transport.RequestPlan `json:"dry_run,omitempty"`
	RawBody   []byte                 `json:"-"`
}

func (r *SendEmailResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

type SendGovRequest struct {
	ConfigPath string
	Profile    string
	Env        config.Env
	Timeout    time.Duration
	MaxRetries int
	ID         string
	DryRun     bool
}

type SendGovResponse struct {
	Invoice   map[string]any
	RawBody   []byte
	Profile   string
	RequestID string
	DryRun    *transport.RequestPlan
}

func (r *SendGovResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

type ChangeStatusRequest struct {
	ConfigPath string
	Profile    string
	Env        config.Env
	Timeout    time.Duration
	MaxRetries int
	ID         string
	Status     string
	DryRun     bool
}

type ChangeStatusResponse struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"`
	Changed   bool                   `json:"changed"`
	Response  map[string]any         `json:"response,omitempty"`
	Profile   string                 `json:"profile"`
	RequestID string                 `json:"request_id,omitempty"`
	DryRun    *transport.RequestPlan `json:"dry_run,omitempty"`
	RawBody   []byte                 `json:"-"`
}

func (r *ChangeStatusResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

type CancelRequest struct {
	ConfigPath string
	Profile    string
	Env        config.Env
	Timeout    time.Duration
	MaxRetries int
	ID         string
	Reason     string
	DryRun     bool
}

type CancelResponse struct {
	ID        string                 `json:"id"`
	Cancelled bool                   `json:"cancelled"`
	Reason    string                 `json:"reason,omitempty"`
	Response  map[string]any         `json:"response,omitempty"`
	Profile   string                 `json:"profile"`
	RequestID string                 `json:"request_id,omitempty"`
	DryRun    *transport.RequestPlan `json:"dry_run,omitempty"`
	RawBody   []byte                 `json:"-"`
}

func (r *CancelResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

type PublicLinkRequest struct {
	ConfigPath string
	Profile    string
	Env        config.Env
	Timeout    time.Duration
	MaxRetries int
	ID         string
}

type PublicLinkResponse struct {
	ID           string `json:"id"`
	Token        string `json:"token"`
	ViewURL      string `json:"view_url"`
	PDFURL       string `json:"pdf_url"`
	PDFInlineURL string `json:"pdf_inline_url"`
	Profile      string `json:"profile"`
	RequestID    string `json:"request_id,omitempty"`
}

func (r *PublicLinkResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

type AttachmentStepPlan struct {
	Name    string `json:"name"`
	Request any    `json:"request,omitempty"`
	Note    string `json:"note,omitempty"`
}

type AddAttachmentPlan struct {
	Steps []AttachmentStepPlan `json:"steps"`
}

type AddAttachmentRequest struct {
	ConfigPath string
	Profile    string
	Env        config.Env
	Timeout    time.Duration
	MaxRetries int
	ID         string
	Name       string
	Content    []byte
	DryRun     bool
}

type AddAttachmentResponse struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Bytes     int                `json:"bytes"`
	Attached  bool               `json:"attached"`
	Profile   string             `json:"profile"`
	RequestID string             `json:"request_id,omitempty"`
	DryRun    *AddAttachmentPlan `json:"dry_run,omitempty"`
}

func (r *AddAttachmentResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

type DownloadAttachmentsRequest struct {
	ConfigPath string
	Profile    string
	Env        config.Env
	Timeout    time.Duration
	MaxRetries int
	ID         string
	Path       string
	Dir        string
}

type DownloadAttachmentsResponse struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	Bytes     int    `json:"bytes"`
	Profile   string `json:"profile"`
	RequestID string `json:"request_id,omitempty"`
}

func (r *DownloadAttachmentsResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

type DownloadAttachmentRequest struct {
	ConfigPath string
	Profile    string
	Env        config.Env
	Timeout    time.Duration
	MaxRetries int
	ID         string
	Kind       string
	Path       string
	Dir        string
}

type DownloadAttachmentResponse struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	Bytes     int    `json:"bytes"`
	FileName  string `json:"file_name,omitempty"`
	Profile   string `json:"profile"`
	RequestID string `json:"request_id,omitempty"`
}

func (r *DownloadAttachmentResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

type FiscalPrintRequest struct {
	ConfigPath string
	Profile    string
	Env        config.Env
	Timeout    time.Duration
	MaxRetries int
	InvoiceIDs []string
	Printer    string
	DryRun     bool
}

type FiscalPrintResponse struct {
	InvoiceIDs []string               `json:"invoice_ids"`
	Printer    string                 `json:"printer,omitempty"`
	Submitted  bool                   `json:"submitted"`
	Profile    string                 `json:"profile"`
	RequestID  string                 `json:"request_id,omitempty"`
	DryRun     *transport.RequestPlan `json:"dry_run,omitempty"`
}

func (r *FiscalPrintResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

func NewService(store config.TokenStore) *Service {
	return &Service{store: store}
}

func newServiceWithHTTPClient(store config.TokenStore, httpClient *http.Client) *Service {
	return &Service{store: store, httpClient: httpClient}
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
	kinds := trimNonEmpty(req.Kinds)
	if len(kinds) == 1 {
		query.Set("kind", kinds[0])
	} else {
		for _, kind := range kinds {
			query.Add("kinds[]", kind)
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

	query := url.Values{}
	if len(req.Includes) > 0 {
		includes := make([]string, 0, len(req.Includes))
		for _, value := range req.Includes {
			for _, part := range strings.Split(value, ",") {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					includes = append(includes, trimmed)
				}
			}
		}
		if len(includes) > 0 {
			query.Set("include", strings.Join(includes, ","))
		}
	}
	if len(req.AdditionalFields) > 0 {
		fields := make([]string, 0, len(req.AdditionalFields))
		for _, value := range req.AdditionalFields {
			for _, part := range strings.Split(value, ",") {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					fields = append(fields, trimmed)
				}
			}
		}
		if len(fields) > 0 {
			query.Set("additional_fields[invoice]", strings.Join(fields, ","))
		}
	}
	if trimmed := strings.TrimSpace(req.CorrectionDetails); trimmed != "" {
		query.Set("correction_positions", trimmed)
	}

	var invoice map[string]any
	resp, err := client.GetJSON(ctx, fmt.Sprintf("/invoices/%s.json", url.PathEscape(req.ID)), query, &invoice)
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

	targetPath, err := artifactPath(req.ID, req.Path, req.Dir, "invoice-", ".pdf")
	if err != nil {
		return nil, err
	}
	if err := writeAtomicFile(targetPath, resp.RawBody); err != nil {
		return nil, err
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

func (s *Service) Create(ctx context.Context, req CreateRequest) (*CreateResponse, error) {
	if err := validateInput(req.Input); err != nil {
		return nil, err
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	query := companionQuery(req.CorrectionPositions)
	payload := wrapPayload(req.Input, req.IdentifyOSS, req.FillDefaultDescriptions, req.GovSaveAndSend)
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodPost, "/invoices.json", query, payload)
		return &CreateResponse{Profile: resolved.Name, DryRun: &plan}, nil
	}

	var invoice map[string]any
	resp, err := client.PostJSONQuery(ctx, "/invoices.json", query, payload, &invoice)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{
		Invoice:   invoice,
		RawBody:   resp.RawBody,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
	}, nil
}

func (s *Service) Update(ctx context.Context, req UpdateRequest) (*UpdateResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "invoice ID is required", "pass --id <invoice-id>")
	}
	if err := validateInput(req.Input); err != nil {
		return nil, err
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/invoices/%s.json", url.PathEscape(req.ID))
	query := companionQuery(req.CorrectionPositions)
	payload := wrapPayload(req.Input, req.IdentifyOSS, req.FillDefaultDescriptions, req.GovSaveAndSend)
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodPut, path, query, payload)
		return &UpdateResponse{Profile: resolved.Name, DryRun: &plan}, nil
	}

	var invoice map[string]any
	resp, err := client.PutJSONQuery(ctx, path, query, payload, &invoice)
	if err != nil {
		return nil, err
	}
	return &UpdateResponse{
		Invoice:   invoice,
		RawBody:   resp.RawBody,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
	}, nil
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (*DeleteResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "invoice ID is required", "pass --id <invoice-id>")
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/invoices/%s.json", url.PathEscape(req.ID))
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
		Response:  optionalJSONObject(resp.RawBody),
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
		RawBody:   resp.RawBody,
	}, nil
}

func (s *Service) SendEmail(ctx context.Context, req SendEmailRequest) (*SendEmailResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "invoice ID is required", "pass --id <invoice-id>")
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	if len(req.EmailTo) > 0 {
		query.Set("email_to", joinTrimmed(req.EmailTo))
	}
	if len(req.EmailCC) > 0 {
		query.Set("email_cc", joinTrimmed(req.EmailCC))
	}
	if req.EmailPDF {
		query.Set("email_pdf", "true")
	}
	if req.UpdateBuyerEmail {
		query.Set("update_buyer_email", "true")
	}
	if req.PrintOption != "" {
		query.Set("print_option", req.PrintOption)
	}
	path := fmt.Sprintf("/invoices/%s/send_by_email.json", url.PathEscape(req.ID))
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodPost, path, query, nil)
		return &SendEmailResponse{ID: req.ID, Profile: resolved.Name, DryRun: &plan}, nil
	}

	resp, err := client.PostJSONQuery(ctx, path, query, nil, nil)
	if err != nil {
		return nil, err
	}
	if appErr := invoiceEmailRemoteError(resp.RawBody); appErr != nil {
		return nil, appErr.WithRawBody(resp.RawBody)
	}
	return &SendEmailResponse{
		ID:        req.ID,
		Sent:      true,
		Response:  optionalJSONObject(resp.RawBody),
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
		RawBody:   resp.RawBody,
	}, nil
}

func (s *Service) SendGov(ctx context.Context, req SendGovRequest) (*SendGovResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "invoice ID is required", "pass --id <invoice-id>")
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("send_to_ksef", "yes")
	path := fmt.Sprintf("/invoices/%s.json", url.PathEscape(req.ID))
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodGet, path, query, nil)
		return &SendGovResponse{Profile: resolved.Name, DryRun: &plan}, nil
	}

	var invoice map[string]any
	resp, err := client.GetJSON(ctx, path, query, &invoice)
	if err != nil {
		return nil, err
	}
	return &SendGovResponse{
		Invoice:   invoice,
		RawBody:   resp.RawBody,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
	}, nil
}

func (s *Service) ChangeStatus(ctx context.Context, req ChangeStatusRequest) (*ChangeStatusResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "invoice ID is required", "pass --id <invoice-id>")
	}
	if strings.TrimSpace(req.Status) == "" {
		return nil, output.Usage("missing_status", "target status is required", "pass --status <status>")
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("status", req.Status)
	path := fmt.Sprintf("/invoices/%s/change_status.json", url.PathEscape(req.ID))
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodPost, path, query, nil)
		return &ChangeStatusResponse{ID: req.ID, Status: req.Status, Profile: resolved.Name, DryRun: &plan}, nil
	}

	resp, err := client.PostJSONQuery(ctx, path, query, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ChangeStatusResponse{
		ID:        req.ID,
		Status:    req.Status,
		Changed:   true,
		Response:  optionalJSONObject(resp.RawBody),
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
		RawBody:   resp.RawBody,
	}, nil
}

func (s *Service) Cancel(ctx context.Context, req CancelRequest) (*CancelResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "invoice ID is required", "pass --id <invoice-id>")
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"cancel_invoice_id": req.ID,
	}
	if trimmed := strings.TrimSpace(req.Reason); trimmed != "" {
		payload["cancel_reason"] = trimmed
	}
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodPost, "/invoices/cancel.json", nil, payload)
		return &CancelResponse{ID: req.ID, Reason: req.Reason, Profile: resolved.Name, DryRun: &plan}, nil
	}

	resp, err := client.PostJSON(ctx, "/invoices/cancel.json", payload, nil)
	if err != nil {
		return nil, err
	}
	return &CancelResponse{
		ID:        req.ID,
		Cancelled: true,
		Reason:    req.Reason,
		Response:  optionalJSONObject(resp.RawBody),
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
		RawBody:   resp.RawBody,
	}, nil
}

func (s *Service) PublicLink(ctx context.Context, req PublicLinkRequest) (*PublicLinkResponse, error) {
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
	token := strings.TrimSpace(fmt.Sprint(invoice["token"]))
	if token == "" || token == "<nil>" {
		return nil, output.Remote("missing_invoice_token", "invoice response did not include a public token", "inspect the invoice JSON and verify the upstream account response", false).WithRawBody(resp.RawBody)
	}

	base := strings.TrimRight(resolved.URL, "/")
	viewURL := base + "/invoice/" + url.PathEscape(token)
	pdfURL := viewURL + ".pdf"
	return &PublicLinkResponse{
		ID:           req.ID,
		Token:        token,
		ViewURL:      viewURL,
		PDFURL:       pdfURL,
		PDFInlineURL: pdfURL + "?inline=yes",
		Profile:      resolved.Name,
		RequestID:    resp.RequestID,
	}, nil
}

func (s *Service) AddAttachment(ctx context.Context, req AddAttachmentRequest) (*AddAttachmentResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "invoice ID is required", "pass --id <invoice-id>")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, output.Usage("missing_name", "attachment name is required", "pass --name <file-name.ext> or provide a file path so the CLI can infer the name")
	}
	if len(req.Content) == 0 {
		return nil, output.Usage("missing_file", "attachment content is required", "pass --file /path/to/file or --file - with stdin data")
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	credentialsPath := fmt.Sprintf("/invoices/%s/get_new_attachment_credentials.json", url.PathEscape(req.ID))
	attachPath := fmt.Sprintf("/invoices/%s/add_attachment.json", url.PathEscape(req.ID))
	attachQuery := url.Values{}
	attachQuery.Set("file_name", name)
	if req.DryRun {
		plan := &AddAttachmentPlan{
			Steps: []AttachmentStepPlan{
				{Name: "get_credentials", Request: transport.PlanJSONRequest(http.MethodGet, credentialsPath, nil, nil)},
				{
					Name: "upload_file",
					Request: transport.MultipartUploadPlan{
						Method: http.MethodPost,
						URL:    "[from get_credentials response]",
						Fields: map[string]string{
							"AWSAccessKeyId":        "[from credentials response]",
							"key":                   "[from credentials response]",
							"policy":                "[from credentials response]",
							"signature":             "[from credentials response]",
							"acl":                   "[from credentials response]",
							"success_action_status": "[from credentials response]",
						},
						FileField: "file",
						FileName:  name,
						Bytes:     len(req.Content),
					},
				},
				{Name: "attach_to_invoice", Request: transport.PlanJSONRequest(http.MethodPost, attachPath, attachQuery, nil)},
			},
		}
		return &AddAttachmentResponse{ID: req.ID, Name: name, Bytes: len(req.Content), Profile: resolved.Name, DryRun: plan}, nil
	}

	var creds map[string]any
	if _, err := client.GetJSON(ctx, credentialsPath, nil, &creds); err != nil {
		return nil, err
	}
	uploadURL := firstString(creds, "url", "action", "post_url")
	if strings.TrimSpace(uploadURL) == "" {
		return nil, output.Remote("missing_attachment_upload_url", "attachment credentials response did not include an upload URL", "inspect the upstream response with --json or --raw on a future release", false)
	}
	fields := uploadFields(creds)
	if _, err := client.UploadMultipart(ctx, transport.MultipartUpload{
		URL:         uploadURL,
		Fields:      fields,
		FileField:   "file",
		FileName:    name,
		FileContent: req.Content,
	}); err != nil {
		return nil, err
	}

	resp, err := client.PostJSONQuery(ctx, attachPath, attachQuery, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AddAttachmentResponse{
		ID:        req.ID,
		Name:      name,
		Bytes:     len(req.Content),
		Attached:  true,
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
	}, nil
}

func (s *Service) DownloadAttachments(ctx context.Context, req DownloadAttachmentsRequest) (*DownloadAttachmentsResponse, error) {
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

	resp, err := client.GetContent(ctx, fmt.Sprintf("/invoices/%s/attachments_zip.json", url.PathEscape(req.ID)), nil, "application/zip")
	if err != nil {
		return nil, err
	}

	targetPath, err := artifactPath(req.ID, req.Path, req.Dir, "invoice-", "-attachments.zip")
	if err != nil {
		return nil, err
	}
	if err := writeAtomicFile(targetPath, resp.RawBody); err != nil {
		return nil, err
	}

	return &DownloadAttachmentsResponse{
		ID:        req.ID,
		Path:      targetPath,
		Bytes:     len(resp.RawBody),
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
	}, nil
}

func (s *Service) DownloadAttachment(ctx context.Context, req DownloadAttachmentRequest) (*DownloadAttachmentResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, output.Usage("missing_id", "invoice ID is required", "pass --id <invoice-id>")
	}
	kind := strings.TrimSpace(req.Kind)
	if kind == "" {
		return nil, output.Usage("missing_kind", "attachment kind is required", "pass --kind gov, --kind gov_upo, or another supported invoice attachment kind")
	}
	if req.Path != "" && req.Dir != "" {
		return nil, output.Usage("path_conflict", "--path and --dir cannot be used together", "pass either --path or --dir")
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("kind", kind)
	resp, err := client.GetContent(ctx, fmt.Sprintf("/invoices/%s/attachment", url.PathEscape(req.ID)), query, "*/*")
	if err != nil {
		return nil, err
	}

	filename := attachmentFilename(resp.Header)
	fallback := fmt.Sprintf("invoice-%s-%s", req.ID, sanitizeAttachmentKind(kind))
	targetPath, err := artifactPathWithSuggestedName(req.Path, req.Dir, fallback, filename)
	if err != nil {
		return nil, err
	}
	if err := writeAtomicFile(targetPath, resp.RawBody); err != nil {
		return nil, err
	}

	return &DownloadAttachmentResponse{
		ID:        req.ID,
		Kind:      kind,
		Path:      targetPath,
		Bytes:     len(resp.RawBody),
		FileName:  filepath.Base(targetPath),
		Profile:   resolved.Name,
		RequestID: resp.RequestID,
	}, nil
}

func (s *Service) FiscalPrint(ctx context.Context, req FiscalPrintRequest) (*FiscalPrintResponse, error) {
	ids := trimNonEmpty(req.InvoiceIDs)
	if len(ids) == 0 {
		return nil, output.Usage("missing_invoice_ids", "at least one --invoice-id is required", "pass one or more --invoice-id values")
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	for _, id := range ids {
		query.Add("invoice_ids[]", id)
	}
	if trimmed := strings.TrimSpace(req.Printer); trimmed != "" {
		query.Set("fiskator_name", trimmed)
	}
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodGet, "/invoices/fiscal_print", query, nil)
		return &FiscalPrintResponse{InvoiceIDs: ids, Printer: req.Printer, Profile: resolved.Name, DryRun: &plan}, nil
	}

	resp, err := client.GetContent(ctx, "/invoices/fiscal_print", query, "*/*")
	if err != nil {
		return nil, err
	}
	return &FiscalPrintResponse{
		InvoiceIDs: ids,
		Printer:    req.Printer,
		Submitted:  true,
		Profile:    resolved.Name,
		RequestID:  resp.RequestID,
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
	client, err := transport.NewClient(resolved.URL, resolved.Token, timeout, maxRetries, s.httpClient)
	if err != nil {
		return nil, nil, err
	}
	return resolved, client, nil
}

func validateInput(input map[string]any) error {
	if input == nil {
		return output.Usage("missing_input", "invoice input is required", "pass --input -|@file.json|'{...}'")
	}
	return nil
}

func wrapPayload(input map[string]any, identifyOSS, fillDefaultDescriptions, govSaveAndSend bool) map[string]any {
	payload := map[string]any{
		"invoice": cloneValue(input).(map[string]any),
	}
	if identifyOSS {
		payload["identify_oss"] = "1"
	}
	if fillDefaultDescriptions {
		payload["fill_default_descriptions"] = true
	}
	if govSaveAndSend {
		payload["gov_save_and_send"] = true
	}
	return payload
}

func companionQuery(correctionPositions string) url.Values {
	query := url.Values{}
	if trimmed := strings.TrimSpace(correctionPositions); trimmed != "" {
		query.Set("correction_positions", trimmed)
	}
	if len(query) == 0 {
		return nil
	}
	return query
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

func optionalJSONObject(raw []byte) map[string]any {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	var value any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&value); err != nil {
		return nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return object
}

func invoiceEmailRemoteError(raw []byte) *output.AppError {
	body := optionalJSONObject(raw)
	if body == nil {
		return nil
	}
	status := strings.TrimSpace(fmt.Sprint(body["status"]))
	message := strings.TrimSpace(fmt.Sprint(body["message"]))
	if status == "error" && strings.Contains(message, "brak numeru KSeF") {
		return output.Remote("ksef_number_required", message, "send the invoice to KSeF first and wait for `gov_id` before retrying the email", false)
	}
	return nil
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(fmt.Sprint(values[key]))
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func uploadFields(creds map[string]any) map[string]string {
	fields := make(map[string]string)
	for key, value := range creds {
		switch key {
		case "url", "action", "post_url":
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				fields[key] = typed
			}
		case json.Number:
			fields[key] = typed.String()
		case bool:
			if typed {
				fields[key] = "true"
			} else {
				fields[key] = "false"
			}
		default:
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" && text != "<nil>" {
				fields[key] = text
			}
		}
	}
	return fields
}

func artifactPath(invoiceID, explicitPath, dir, prefix, suffix string) (string, error) {
	if explicitPath != "" {
		return explicitPath, nil
	}
	if dir == "" {
		dir = "."
	}
	return filepath.Join(dir, prefix+invoiceID+suffix), nil
}

func artifactPathWithSuggestedName(explicitPath, dir, fallbackName, suggestedName string) (string, error) {
	if explicitPath != "" {
		return explicitPath, nil
	}
	if dir == "" {
		dir = "."
	}
	name := filepath.Base(strings.TrimSpace(suggestedName))
	if name == "" || name == "." || name == string(filepath.Separator) {
		name = fallbackName
	}
	return filepath.Join(dir, name), nil
}

func writeAtomicFile(targetPath string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return output.Internal(err, "create download directory")
	}
	tempFile, err := os.CreateTemp(filepath.Dir(targetPath), "fakturownia-*")
	if err != nil {
		return output.Internal(err, "create temporary download file")
	}
	tempPath := tempFile.Name()
	if _, err := tempFile.Write(content); err != nil {
		_ = tempFile.Close()
		return output.Internal(err, "write downloaded file")
	}
	if err := tempFile.Close(); err != nil {
		return output.Internal(err, "close downloaded file")
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		return output.Internal(err, "move downloaded file into place")
	}
	return nil
}

func joinTrimmed(values []string) string {
	return strings.Join(trimNonEmpty(values), ",")
}

func attachmentFilename(header http.Header) string {
	contentDisposition := strings.TrimSpace(header.Get("Content-Disposition"))
	if contentDisposition == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(params["filename"])
}

func sanitizeAttachmentKind(kind string) string {
	trimmed := strings.TrimSpace(kind)
	if trimmed == "" {
		return "attachment"
	}
	var b strings.Builder
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "attachment"
	}
	return out
}

func trimNonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
	}
	return out
}
