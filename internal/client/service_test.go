package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sixers/fakturownia-cli/internal/config"
)

type stubTokenStore struct{}

func (stubTokenStore) Get(string) (string, error) { return "", config.ErrSecretNotFound }

func TestListBuildsQueryAndPagination(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("page") != "2" || query.Get("per_page") != "25" {
			t.Fatalf("unexpected pagination query: %s", r.URL.RawQuery)
		}
		if query.Get("name") != "Acme" || query.Get("email") != "billing@example.com" || query.Get("shortcut") != "AC" || query.Get("tax_no") != "123" || query.Get("external_id") != "ext-1" {
			t.Fatalf("unexpected filter query: %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "name": "Acme"}})
	}))
	defer server.Close()
	restore := swapDefaultTransport(server.Client().Transport)
	defer restore()

	service := NewService(stubTokenStore{})
	result, err := service.List(context.Background(), ListRequest{
		Env:        config.Env{URL: server.URL, APIToken: "token"},
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		Page:       2,
		PerPage:    25,
		Name:       "Acme",
		Email:      "billing@example.com",
		Shortcut:   "AC",
		TaxNo:      "123",
		ExternalID: "ext-1",
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Pagination.Page != 2 || result.Pagination.PerPage != 25 || result.Pagination.Returned != 1 || result.Pagination.HasNext {
		t.Fatalf("unexpected pagination: %#v", result.Pagination)
	}
}

func TestGetByExternalIDMatchCounts(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		expectErr  bool
		expectCode string
	}{
		{name: "none", body: `[]`, expectErr: true, expectCode: "client_not_found"},
		{name: "one", body: `[{"id":1,"external_id":"ext-1"}]`, expectErr: false},
		{name: "many", body: `[{"id":1},{"id":2}]`, expectErr: true, expectCode: "multiple_clients_matched"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			restore := swapDefaultTransport(server.Client().Transport)
			defer restore()

			service := NewService(stubTokenStore{})
			result, err := service.Get(context.Background(), GetRequest{
				Env:        config.Env{URL: server.URL, APIToken: "token"},
				Timeout:    5 * time.Second,
				MaxRetries: 0,
				ExternalID: "ext-1",
			})
			if tc.expectErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if got := err.Error(); !strings.Contains(got, tc.expectCode) && !strings.Contains(got, "matched external_id") {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if result.Client["external_id"] != "ext-1" {
				t.Fatalf("unexpected client: %#v", result.Client)
			}
		})
	}
}

func TestCreateUpdateDeleteAndDryRun(t *testing.T) {
	var seenMethods []string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenMethods = append(seenMethods, r.Method)
		switch r.Method {
		case http.MethodPost:
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if payload["api_token"] != "token" {
				t.Fatalf("expected api_token in POST body, got %#v", payload["api_token"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 10, "name": "Created"})
		case http.MethodPut:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 10, "email": "updated@example.com"})
		case http.MethodDelete:
			if r.URL.Query().Get("api_token") != "token" {
				t.Fatalf("expected api_token query on DELETE, got %q", r.URL.Query().Get("api_token"))
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()
	restore := swapDefaultTransport(server.Client().Transport)
	defer restore()

	service := NewService(stubTokenStore{})
	env := config.Env{URL: server.URL, APIToken: "token"}

	created, err := service.Create(context.Background(), CreateRequest{
		Env:        env,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		Input:      map[string]any{"name": "Created"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Client["id"] == nil {
		t.Fatalf("unexpected create response: %#v", created.Client)
	}

	updated, err := service.Update(context.Background(), UpdateRequest{
		Env:        env,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		ID:         "10",
		Input:      map[string]any{"email": "updated@example.com"},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Client["email"] != "updated@example.com" {
		t.Fatalf("unexpected update response: %#v", updated.Client)
	}

	deleted, err := service.Delete(context.Background(), DeleteRequest{
		Env:        env,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		ID:         "10",
	})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted.Deleted {
		t.Fatalf("unexpected delete response: %#v", deleted)
	}

	plan, err := service.Create(context.Background(), CreateRequest{
		Env:        env,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		Input:      map[string]any{"name": "Created"},
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("Create() dry-run error = %v", err)
	}
	if plan.DryRun == nil || plan.DryRun.Method != http.MethodPost {
		t.Fatalf("unexpected dry-run plan: %#v", plan.DryRun)
	}
	if plan.DryRun.Body.(map[string]any)["api_token"] != "[redacted]" {
		t.Fatalf("expected redacted token in dry-run body: %#v", plan.DryRun.Body)
	}

	if strings.Join(seenMethods, ",") != "POST,PUT,DELETE" {
		t.Fatalf("unexpected method sequence: %v", seenMethods)
	}
}

func TestParseInputSources(t *testing.T) {
	t.Parallel()

	inline, err := ParseInput(`{"name":"Acme"}`, nil)
	if err != nil || inline["name"] != "Acme" {
		t.Fatalf("ParseInput(inline) = %#v, %v", inline, err)
	}

	stdin, err := ParseInput("-", strings.NewReader(`{"name":"stdin"}`))
	if err != nil || stdin["name"] != "stdin" {
		t.Fatalf("ParseInput(stdin) = %#v, %v", stdin, err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "client.json")
	if err := os.WriteFile(path, []byte(`{"name":"file"}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	fromFile, err := ParseInput("@"+path, nil)
	if err != nil || fromFile["name"] != "file" {
		t.Fatalf("ParseInput(file) = %#v, %v", fromFile, err)
	}
}

func swapDefaultTransport(transport http.RoundTripper) func() {
	previous := http.DefaultTransport
	http.DefaultTransport = transport
	return func() {
		http.DefaultTransport = previous
	}
}

func TestParseInputRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		raw   string
		stdin io.Reader
	}{
		{name: "empty", raw: ""},
		{name: "array", raw: `[]`},
		{name: "invalid", raw: `{`},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseInput(tc.raw, tc.stdin); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
