package warehousedocument

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
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "kind": "mm", "number": "MM/1/2026"}})
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
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Pagination.Page != 2 || result.Pagination.PerPage != 25 || result.Pagination.Returned != 1 || result.Pagination.HasNext {
		t.Fatalf("unexpected pagination: %#v", result.Pagination)
	}
}

func TestGetReturnsWarehouseDocument(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/warehouse_documents/15.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":          15,
			"kind":        "wz",
			"client_name": "Acme",
			"warehouse_actions": []any{
				map[string]any{"product_id": 7, "quantity": 2},
			},
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
		ID:         "15",
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if result.WarehouseDocument["kind"] != "wz" {
		t.Fatalf("unexpected warehouse document: %#v", result.WarehouseDocument)
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
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 15, "kind": "mm"})
		case http.MethodPut:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 15, "invoice_ids": []any{100, 111}})
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
			"kind": "mm",
			"warehouse_actions": []any{
				map[string]any{"product_id": 7, "quantity": 2, "warehouse2_id": 3},
			},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.WarehouseDocument["id"] == nil {
		t.Fatalf("unexpected create response: %#v", created.WarehouseDocument)
	}

	updated, err := service.Update(context.Background(), UpdateRequest{
		Env:        env,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		ID:         "15",
		Input:      map[string]any{"invoice_ids": []any{100, 111}},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	invoiceIDs, ok := updated.WarehouseDocument["invoice_ids"].([]any)
	if !ok || len(invoiceIDs) != 2 {
		t.Fatalf("unexpected update response: %#v", updated.WarehouseDocument)
	}

	deleted, err := service.Delete(context.Background(), DeleteRequest{
		Env:        env,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		ID:         "15",
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
		ID:         "15",
		Input:      map[string]any{"kind": "wz"},
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("Update() dry-run error = %v", err)
	}
	if plan.DryRun == nil || plan.DryRun.Method != http.MethodPut {
		t.Fatalf("unexpected update dry-run plan: %#v", plan.DryRun)
	}
	body := plan.DryRun.Body.(map[string]any)
	if body["api_token"] != "[redacted]" {
		t.Fatalf("expected redacted token in dry-run body: %#v", body)
	}

	deletePlan, err := service.Delete(context.Background(), DeleteRequest{
		Env:        env,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		ID:         "15",
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("Delete() dry-run error = %v", err)
	}
	if deletePlan.DryRun == nil || deletePlan.DryRun.Method != http.MethodDelete {
		t.Fatalf("unexpected delete dry-run plan: %#v", deletePlan.DryRun)
	}

	if strings.Join(seenMethods, ",") != "POST,PUT,DELETE" {
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
