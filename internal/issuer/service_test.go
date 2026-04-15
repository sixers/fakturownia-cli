package issuer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sixers/fakturownia-cli/internal/config"
)

type stubTokenStore struct{}

func (stubTokenStore) Get(string) (string, error) { return "", config.ErrSecretNotFound }

func TestListGetCreateUpdateDeleteAndDryRun(t *testing.T) {
	var seen []string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method)
		switch r.Method {
		case http.MethodGet:
			if r.URL.Path == "/issuers.json" {
				if r.URL.Query().Get("page") != "2" || r.URL.Query().Get("per_page") != "25" {
					t.Fatalf("unexpected query: %s", r.URL.RawQuery)
				}
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 3, "name": "HQ"}})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 3, "name": "HQ", "tax_no": "1234567890"})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 3, "name": "HQ"})
		case http.MethodPut:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 3, "tax_no": "1234567890"})
		case http.MethodDelete:
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
		ID:      "3",
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Issuer["tax_no"] != "1234567890" {
		t.Fatalf("unexpected issuer: %#v", got.Issuer)
	}

	created, err := service.Create(context.Background(), CreateRequest{
		Env:     env,
		Timeout: 5 * time.Second,
		Input:   map[string]any{"name": "HQ"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Issuer["id"] == nil {
		t.Fatalf("unexpected create response: %#v", created.Issuer)
	}

	updated, err := service.Update(context.Background(), UpdateRequest{
		Env:     env,
		Timeout: 5 * time.Second,
		ID:      "3",
		Input:   map[string]any{"tax_no": "1234567890"},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Issuer["tax_no"] != "1234567890" {
		t.Fatalf("unexpected update response: %#v", updated.Issuer)
	}

	deleted, err := service.Delete(context.Background(), DeleteRequest{
		Env:     env,
		Timeout: 5 * time.Second,
		ID:      "3",
	})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted.Deleted {
		t.Fatalf("expected deleted response: %#v", deleted)
	}

	plan, err := service.Update(context.Background(), UpdateRequest{
		Env:     env,
		Timeout: 5 * time.Second,
		ID:      "3",
		Input:   map[string]any{"name": "Plan"},
		DryRun:  true,
	})
	if err != nil {
		t.Fatalf("Update() dry-run error = %v", err)
	}
	if plan.DryRun == nil || plan.DryRun.Method != http.MethodPut {
		t.Fatalf("unexpected dry-run plan: %#v", plan.DryRun)
	}

	if strings.Join(seen, ",") != "GET,GET,POST,PUT,DELETE" {
		t.Fatalf("unexpected method sequence: %s", strings.Join(seen, ","))
	}
}

func swapDefaultTransport(transport http.RoundTripper) func() {
	previous := http.DefaultTransport
	http.DefaultTransport = transport
	return func() {
		http.DefaultTransport = previous
	}
}
