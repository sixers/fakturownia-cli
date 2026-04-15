package department

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sixers/fakturownia-cli/internal/config"
)

type stubTokenStore struct{}

func (stubTokenStore) Get(string) (string, error) { return "", config.ErrSecretNotFound }

func TestListGetCreateUpdateDeleteAndSetLogo(t *testing.T) {
	var seen []string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/departments.json":
			if r.URL.Query().Get("page") != "2" || r.URL.Query().Get("per_page") != "25" {
				t.Fatalf("unexpected list query: %s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 10, "name": "Sales"}})
		case r.Method == http.MethodGet && r.URL.Path == "/departments/10.json":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 10, "name": "Sales", "shortcut": "SALES"})
		case r.Method == http.MethodPost && r.URL.Path == "/departments.json":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 10, "name": "Sales"})
		case r.Method == http.MethodPut && r.URL.Path == "/departments/10.json" && strings.HasPrefix(r.Header.Get("Content-Type"), "application/json"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 10, "shortcut": "SALES"})
		case r.Method == http.MethodDelete && r.URL.Path == "/departments/10.json":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut && r.URL.Path == "/departments/10.json":
			mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil {
				t.Fatalf("ParseMediaType() error = %v", err)
			}
			if !strings.HasPrefix(mediaType, "multipart/") {
				t.Fatalf("expected multipart type, got %q", mediaType)
			}
			reader := multipart.NewReader(r.Body, params["boundary"])
			fields := map[string]string{}
			fileFound := false
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("NextPart() error = %v", err)
				}
				if part.FileName() != "" {
					if part.FormName() != "department[logo]" {
						t.Fatalf("unexpected file field %q", part.FormName())
					}
					if part.FileName() != "logo.png" {
						t.Fatalf("unexpected file name %q", part.FileName())
					}
					fileFound = true
					continue
				}
				data, err := io.ReadAll(part)
				if err != nil {
					t.Fatalf("ReadAll() error = %v", err)
				}
				fields[part.FormName()] = string(data)
			}
			if fields["api_token"] != "token" || !fileFound {
				t.Fatalf("unexpected multipart fields %#v file=%v", fields, fileFound)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 10, "name": "Sales", "logo_url": "https://example.test/logo.png"})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	restore := swapDefaultTransport(server.Client().Transport)
	defer restore()

	service := NewService(stubTokenStore{})
	env := config.Env{URL: server.URL, APIToken: "token"}

	listed, err := service.List(context.Background(), ListRequest{
		Env:     env,
		Timeout: 5 * time.Second,
		Page:    2,
		PerPage: 25,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if listed.Pagination.Page != 2 || listed.Pagination.Returned != 1 {
		t.Fatalf("unexpected pagination: %#v", listed.Pagination)
	}

	got, err := service.Get(context.Background(), GetRequest{
		Env:     env,
		Timeout: 5 * time.Second,
		ID:      "10",
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Department["name"] != "Sales" {
		t.Fatalf("unexpected get response: %#v", got.Department)
	}

	created, err := service.Create(context.Background(), CreateRequest{
		Env:     env,
		Timeout: 5 * time.Second,
		Input:   map[string]any{"name": "Sales"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Department["id"] == nil {
		t.Fatalf("unexpected create response: %#v", created.Department)
	}

	updated, err := service.Update(context.Background(), UpdateRequest{
		Env:     env,
		Timeout: 5 * time.Second,
		ID:      "10",
		Input:   map[string]any{"shortcut": "SALES"},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Department["shortcut"] != "SALES" {
		t.Fatalf("unexpected update response: %#v", updated.Department)
	}

	deleted, err := service.Delete(context.Background(), DeleteRequest{
		Env:     env,
		Timeout: 5 * time.Second,
		ID:      "10",
	})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted.Deleted {
		t.Fatalf("expected deleted response: %#v", deleted)
	}

	logo, err := service.SetLogo(context.Background(), SetLogoRequest{
		Env:     env,
		Timeout: 5 * time.Second,
		ID:      "10",
		Name:    "logo.png",
		Content: []byte("png-bytes"),
	})
	if err != nil {
		t.Fatalf("SetLogo() error = %v", err)
	}
	if !logo.Uploaded || logo.Department["logo_url"] == nil {
		t.Fatalf("unexpected set-logo response: %#v", logo)
	}

	plan, err := service.SetLogo(context.Background(), SetLogoRequest{
		Env:     env,
		Timeout: 5 * time.Second,
		ID:      "10",
		Name:    "logo.png",
		Content: []byte("png-bytes"),
		DryRun:  true,
	})
	if err != nil {
		t.Fatalf("SetLogo() dry-run error = %v", err)
	}
	if plan.DryRun == nil || plan.DryRun.Method != http.MethodPut {
		t.Fatalf("unexpected set-logo dry-run plan: %#v", plan.DryRun)
	}
	if plan.DryRun.Fields["api_token"] != "[redacted]" {
		t.Fatalf("expected redacted token in multipart dry-run fields: %#v", plan.DryRun.Fields)
	}
}

func swapDefaultTransport(transport http.RoundTripper) func() {
	previous := http.DefaultTransport
	http.DefaultTransport = transport
	return func() {
		http.DefaultTransport = previous
	}
}
