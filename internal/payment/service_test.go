package payment

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

func TestListBuildsQueryAndPagination(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("page") != "2" || query.Get("per_page") != "25" || query.Get("include") != "invoices" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 555, "name": "Payment 001", "price": 100.05}})
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
		Include:    []string{"invoices"},
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Pagination.Page != 2 || result.Pagination.PerPage != 25 || result.Pagination.Returned != 1 || result.Pagination.HasNext {
		t.Fatalf("unexpected pagination: %#v", result.Pagination)
	}
	if strings.Join(result.IncludeUsed, ",") != "invoices" {
		t.Fatalf("unexpected include tracking: %#v", result.IncludeUsed)
	}
}

func TestGetReturnsPayment(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/banking/payments/555.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         555,
			"name":       "Payment 001",
			"price":      100.05,
			"invoice_id": nil,
			"paid":       true,
			"kind":       "api",
		})
	}))
	defer server.Close()
	restore := swapDefaultTransport(server.Client().Transport)
	defer restore()

	service := NewService(stubTokenStore{})
	result, err := service.Get(context.Background(), GetRequest{
		Env:        config.Env{URL: server.URL, APIToken: "token"},
		Timeout:    5 * time.Second,
		MaxRetries: 0,
		ID:         "555",
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if result.Payment["name"] != "Payment 001" {
		t.Fatalf("unexpected payment: %#v", result.Payment)
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
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 555, "name": "Payment 001"})
		case http.MethodPatch:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 555, "name": "New payment name", "price": 100})
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

	created, err := service.Create(context.Background(), CreateRequest{
		Env:        env,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		Input: map[string]any{
			"name":  "Payment 001",
			"price": 100.05,
			"paid":  true,
			"kind":  "api",
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Payment["id"] == nil {
		t.Fatalf("unexpected create response: %#v", created.Payment)
	}

	updated, err := service.Update(context.Background(), UpdateRequest{
		Env:        env,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		ID:         "555",
		Input:      map[string]any{"name": "New payment name", "price": 100},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Payment["name"] != "New payment name" {
		t.Fatalf("unexpected update response: %#v", updated.Payment)
	}

	deleted, err := service.Delete(context.Background(), DeleteRequest{
		Env:        env,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		ID:         "555",
	})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted.Deleted {
		t.Fatalf("expected delete response to mark deleted: %#v", deleted)
	}

	plan, err := service.Update(context.Background(), UpdateRequest{
		Env:        env,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		ID:         "555",
		Input:      map[string]any{"name": "plan"},
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("Update() dry-run error = %v", err)
	}
	if plan.DryRun == nil || plan.DryRun.Method != http.MethodPatch {
		t.Fatalf("unexpected dry-run plan: %#v", plan.DryRun)
	}
	body := plan.DryRun.Body.(map[string]any)
	if body["api_token"] != "[redacted]" {
		t.Fatalf("expected redacted token in dry-run body: %#v", body)
	}

	deletePlan, err := service.Delete(context.Background(), DeleteRequest{
		Env:        env,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		ID:         "555",
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("Delete() dry-run error = %v", err)
	}
	if deletePlan.DryRun == nil || deletePlan.DryRun.Method != http.MethodDelete {
		t.Fatalf("unexpected delete dry-run plan: %#v", deletePlan.DryRun)
	}

	if strings.Join(seenMethods, ",") != "POST,PATCH,DELETE" {
		t.Fatalf("unexpected method sequence: %v", seenMethods)
	}
}

func TestListRejectsUnsupportedInclude(t *testing.T) {
	service := NewService(stubTokenStore{})
	_, err := service.List(context.Background(), ListRequest{
		Env:     config.Env{URL: "https://example.com", APIToken: "token"},
		Page:    1,
		PerPage: 25,
		Include: []string{"foo"},
	})
	if err == nil {
		t.Fatal("expected include validation error")
	}
}

func swapDefaultTransport(transport http.RoundTripper) func() {
	previous := http.DefaultTransport
	http.DefaultTransport = transport
	return func() {
		http.DefaultTransport = previous
	}
}
