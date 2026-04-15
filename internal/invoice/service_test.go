package invoice

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sixers/fakturownia-cli/internal/config"
)

func useTLSTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	return server
}

func TestCreateDryRunIncludesCompanionOptions(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	result, err := service.Create(context.Background(), CreateRequest{
		ConfigPath:              filepath.Join(t.TempDir(), "config.json"),
		Env:                     config.Env{URL: "https://acme.fakturownia.pl", APIToken: "token"},
		Input:                   map[string]any{"kind": "vat"},
		IdentifyOSS:             true,
		FillDefaultDescriptions: true,
		CorrectionPositions:     "full",
		DryRun:                  true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.DryRun == nil {
		t.Fatal("expected dry-run plan")
	}
	if result.DryRun.Path != "/invoices.json" {
		t.Fatalf("unexpected dry-run path: %#v", result.DryRun)
	}
	body, _ := result.DryRun.Body.(map[string]any)
	if body["identify_oss"] != "1" || body["fill_default_descriptions"] != true {
		t.Fatalf("expected companion options in dry-run body, got %#v", body)
	}
	if got := result.DryRun.Query["correction_positions"]; len(got) != 1 || got[0] != "full" {
		t.Fatalf("expected correction_positions query parameter, got %#v", result.DryRun.Query)
	}
}

func TestGetBuildsIncludeAndCorrectionQueries(t *testing.T) {
	t.Parallel()

	server := useTLSTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/invoices/100.json" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("api_token") != "token" {
			t.Fatalf("unexpected api_token %q", query.Get("api_token"))
		}
		if query.Get("include") != "descriptions" {
			t.Fatalf("unexpected include query %#v", query)
		}
		if query.Get("additional_fields[invoice]") != "cancel_reason,connected_payments" {
			t.Fatalf("unexpected additional_fields query %#v", query)
		}
		if query.Get("correction_positions") != "full" {
			t.Fatalf("unexpected correction_positions query %#v", query)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 100, "descriptions": []any{map[string]any{"content": "Uwagi"}}})
	})

	service := newServiceWithHTTPClient(nil, server.Client())
	result, err := service.Get(context.Background(), GetRequest{
		ConfigPath:        filepath.Join(t.TempDir(), "config.json"),
		Env:               config.Env{URL: server.URL, APIToken: "token"},
		ID:                "100",
		Includes:          []string{"descriptions"},
		AdditionalFields:  []string{"cancel_reason", "connected_payments"},
		CorrectionDetails: "full",
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if result.Invoice["id"] != json.Number("100") && result.Invoice["id"] != 100 {
		t.Fatalf("unexpected invoice payload: %#v", result.Invoice)
	}
}

func TestSendEmailBuildsQueryParameters(t *testing.T) {
	t.Parallel()

	server := useTLSTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/invoices/100/send_by_email.json" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("api_token") != "token" || query.Get("email_to") != "billing@example.com" || query.Get("email_cc") != "cc@example.com" {
			t.Fatalf("unexpected query values: %#v", query)
		}
		if query.Get("email_pdf") != "true" || query.Get("update_buyer_email") != "true" || query.Get("print_option") != "original" {
			t.Fatalf("unexpected boolean/print query values: %#v", query)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	service := newServiceWithHTTPClient(nil, server.Client())
	result, err := service.SendEmail(context.Background(), SendEmailRequest{
		ConfigPath:       filepath.Join(t.TempDir(), "config.json"),
		Env:              config.Env{URL: server.URL, APIToken: "token"},
		ID:               "100",
		EmailTo:          []string{"billing@example.com"},
		EmailCC:          []string{"cc@example.com"},
		EmailPDF:         true,
		UpdateBuyerEmail: true,
		PrintOption:      "original",
	})
	if err != nil {
		t.Fatalf("SendEmail() error = %v", err)
	}
	if !result.Sent || result.Response["ok"] != true {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestAddAttachmentOrchestratesCredentialsUploadAndAttach(t *testing.T) {
	t.Parallel()

	uploadCalled := false
	var attachQuery url.Values
	var server *httptest.Server
	server = useTLSTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/invoices/111/get_new_attachment_credentials.json":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"url":                   server.URL + "/upload",
				"AWSAccessKeyId":        "key-id",
				"key":                   "uploads/scan.pdf",
				"policy":                "policy-value",
				"signature":             "signature-value",
				"acl":                   "private",
				"success_action_status": "201",
			})
		case "/upload":
			uploadCalled = true
			if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "multipart/form-data;") {
				t.Fatalf("unexpected content type %q", got)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}
			if !strings.Contains(string(body), "scan.pdf") || !strings.Contains(string(body), "pdf-bytes") {
				t.Fatalf("unexpected multipart body %q", string(body))
			}
			w.WriteHeader(http.StatusNoContent)
		case "/invoices/111/add_attachment.json":
			attachQuery = r.URL.Query()
			w.WriteHeader(http.StatusCreated)
		default:
			t.Fatalf("unexpected API path %q", r.URL.Path)
		}
	})

	service := newServiceWithHTTPClient(nil, server.Client())
	result, err := service.AddAttachment(context.Background(), AddAttachmentRequest{
		ConfigPath: filepath.Join(t.TempDir(), "config.json"),
		Env:        config.Env{URL: server.URL, APIToken: "token"},
		ID:         "111",
		Name:       "scan.pdf",
		Content:    []byte("pdf-bytes"),
	})
	if err != nil {
		t.Fatalf("AddAttachment() error = %v", err)
	}
	if !uploadCalled {
		t.Fatal("expected multipart upload to be called")
	}
	if attachQuery.Get("api_token") != "token" || attachQuery.Get("file_name") != "scan.pdf" {
		t.Fatalf("unexpected add-attachment query: %#v", attachQuery)
	}
	if !result.Attached || result.Bytes != len("pdf-bytes") {
		t.Fatalf("unexpected add-attachment result: %#v", result)
	}
}

func TestDownloadAttachmentsWritesZipFile(t *testing.T) {
	t.Parallel()

	server := useTLSTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/invoices/111/attachments_zip.json" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte("zip-bytes"))
	})

	service := newServiceWithHTTPClient(nil, server.Client())
	dir := t.TempDir()
	result, err := service.DownloadAttachments(context.Background(), DownloadAttachmentsRequest{
		ConfigPath: filepath.Join(dir, "config.json"),
		Env:        config.Env{URL: server.URL, APIToken: "token"},
		ID:         "111",
		Dir:        dir,
	})
	if err != nil {
		t.Fatalf("DownloadAttachments() error = %v", err)
	}
	if result.Bytes != len("zip-bytes") {
		t.Fatalf("unexpected byte count: %#v", result)
	}
	fileData, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(fileData) != "zip-bytes" {
		t.Fatalf("unexpected file content %q", string(fileData))
	}
}

func TestPublicLinkDerivesURLsFromToken(t *testing.T) {
	t.Parallel()

	server := useTLSTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 100, "token": "TOKEN-123"})
	})

	service := newServiceWithHTTPClient(nil, server.Client())
	result, err := service.PublicLink(context.Background(), PublicLinkRequest{
		ConfigPath: filepath.Join(t.TempDir(), "config.json"),
		Env:        config.Env{URL: server.URL, APIToken: "token"},
		ID:         "100",
	})
	if err != nil {
		t.Fatalf("PublicLink() error = %v", err)
	}
	if result.ViewURL != server.URL+"/invoice/TOKEN-123" || result.PDFInlineURL != server.URL+"/invoice/TOKEN-123.pdf?inline=yes" {
		t.Fatalf("unexpected public-link URLs: %#v", result)
	}
}

func TestChangeStatusBuildsQueryParameters(t *testing.T) {
	t.Parallel()

	server := useTLSTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/invoices/100/change_status.json" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("api_token") != "token" || query.Get("status") != "paid" {
			t.Fatalf("unexpected query values: %#v", query)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"changed": true})
	})

	service := newServiceWithHTTPClient(nil, server.Client())
	result, err := service.ChangeStatus(context.Background(), ChangeStatusRequest{
		ConfigPath: filepath.Join(t.TempDir(), "config.json"),
		Env:        config.Env{URL: server.URL, APIToken: "token"},
		ID:         "100",
		Status:     "paid",
	})
	if err != nil {
		t.Fatalf("ChangeStatus() error = %v", err)
	}
	if !result.Changed || result.Response["changed"] != true {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestCancelPostsReasonInBody(t *testing.T) {
	t.Parallel()

	server := useTLSTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/invoices/cancel.json" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		if payload["cancel_invoice_id"] != "222" || payload["cancel_reason"] != "Wrong document" || payload["api_token"] != "token" {
			t.Fatalf("unexpected cancel payload: %#v", payload)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"cancelled": true})
	})

	service := newServiceWithHTTPClient(nil, server.Client())
	result, err := service.Cancel(context.Background(), CancelRequest{
		ConfigPath: filepath.Join(t.TempDir(), "config.json"),
		Env:        config.Env{URL: server.URL, APIToken: "token"},
		ID:         "222",
		Reason:     "Wrong document",
	})
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if !result.Cancelled || result.Response["cancelled"] != true {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestFiscalPrintBuildsQueryParameters(t *testing.T) {
	t.Parallel()

	server := useTLSTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/invoices/fiscal_print" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("api_token") != "token" || query.Get("fiskator_name") != "PRINTER-1" {
			t.Fatalf("unexpected query values: %#v", query)
		}
		if got := query["invoice_ids[]"]; len(got) != 2 || got[0] != "100" || got[1] != "101" {
			t.Fatalf("unexpected invoice_ids query values: %#v", query)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	service := newServiceWithHTTPClient(nil, server.Client())
	result, err := service.FiscalPrint(context.Background(), FiscalPrintRequest{
		ConfigPath: filepath.Join(t.TempDir(), "config.json"),
		Env:        config.Env{URL: server.URL, APIToken: "token"},
		InvoiceIDs: []string{"100", "101"},
		Printer:    "PRINTER-1",
	})
	if err != nil {
		t.Fatalf("FiscalPrint() error = %v", err)
	}
	if !result.Submitted || result.Printer != "PRINTER-1" {
		t.Fatalf("unexpected result: %#v", result)
	}
}
