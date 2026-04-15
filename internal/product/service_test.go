package product

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
		if query.Get("page") != "2" || query.Get("per_page") != "25" {
			t.Fatalf("unexpected pagination query: %s", r.URL.RawQuery)
		}
		if query.Get("date_from") != "2025-11-01" || query.Get("warehouse_id") != "7" {
			t.Fatalf("unexpected filter query: %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "name": "Widget"}})
	}))
	defer server.Close()
	restore := swapDefaultTransport(server.Client().Transport)
	defer restore()

	service := NewService(stubTokenStore{})
	result, err := service.List(context.Background(), ListRequest{
		Env:         config.Env{URL: server.URL, APIToken: "token"},
		Timeout:     5 * time.Second,
		MaxRetries:  1,
		Page:        2,
		PerPage:     25,
		DateFrom:    "2025-11-01",
		WarehouseID: "7",
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Pagination.Page != 2 || result.Pagination.PerPage != 25 || result.Pagination.Returned != 1 || result.Pagination.HasNext {
		t.Fatalf("unexpected pagination: %#v", result.Pagination)
	}
}

func TestGetSupportsWarehouseID(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("warehouse_id") != "3" {
			t.Fatalf("expected warehouse_id query, got %q", r.URL.Query().Get("warehouse_id"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 10, "name": "Warehouse Product"})
	}))
	defer server.Close()
	restore := swapDefaultTransport(server.Client().Transport)
	defer restore()

	service := NewService(stubTokenStore{})
	result, err := service.Get(context.Background(), GetRequest{
		Env:         config.Env{URL: server.URL, APIToken: "token"},
		Timeout:     5 * time.Second,
		MaxRetries:  0,
		ID:          "10",
		WarehouseID: "3",
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if result.Product["name"] != "Warehouse Product" {
		t.Fatalf("unexpected product: %#v", result.Product)
	}
}

func TestCreateUpdateAndDryRun(t *testing.T) {
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
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 11, "name": "Created Product"})
		case http.MethodPut:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 11, "price_gross": "102"})
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
		Input:      map[string]any{"name": "Created Product"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Product["id"] == nil {
		t.Fatalf("unexpected create response: %#v", created.Product)
	}

	updated, err := service.Update(context.Background(), UpdateRequest{
		Env:        env,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		ID:         "11",
		Input:      map[string]any{"price_gross": "102"},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Product["price_gross"] != "102" {
		t.Fatalf("unexpected update response: %#v", updated.Product)
	}

	plan, err := service.Create(context.Background(), CreateRequest{
		Env:        env,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		Input: map[string]any{
			"name": "Bundle",
			"package_products_details": map[string]any{
				"0": map[string]any{"id": 5, "quantity": 1},
			},
		},
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("Create() dry-run error = %v", err)
	}
	if plan.DryRun == nil || plan.DryRun.Method != http.MethodPost {
		t.Fatalf("unexpected dry-run plan: %#v", plan.DryRun)
	}
	body := plan.DryRun.Body.(map[string]any)
	if body["api_token"] != "[redacted]" {
		t.Fatalf("expected redacted token in dry-run body: %#v", body)
	}

	if strings.Join(seenMethods, ",") != "POST,PUT" {
		t.Fatalf("unexpected method sequence: %v", seenMethods)
	}
}

func swapDefaultTransport(transport http.RoundTripper) func() {
	previous := http.DefaultTransport
	http.DefaultTransport = transport
	return func() {
		http.DefaultTransport = previous
	}
}
