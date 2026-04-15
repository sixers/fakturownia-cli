package pricelist

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
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 8523, "name": "Dropshipper"}})
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

func TestGetReturnsPriceListWithPositions(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/price_lists/8523.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   8523,
			"name": "Dropshipper",
			"price_list_positions": []any{
				map[string]any{"id": 556438, "priceable_id": 97149307, "price_gross": "33.16"},
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
		ID:         "8523",
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if result.PriceList["name"] != "Dropshipper" {
		t.Fatalf("unexpected price list: %#v", result.PriceList)
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
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 8523, "name": "Dropshipper"})
		case http.MethodPut:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 8523, "description": "updated"})
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
			"name":     "Dropshipper",
			"currency": "PLN",
			"price_list_positions_attributes": map[string]any{
				"0": map[string]any{"priceable_id": 97149307, "price_gross": "33.16", "tax": "23"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.PriceList["id"] == nil {
		t.Fatalf("unexpected create response: %#v", created.PriceList)
	}

	updated, err := service.Update(context.Background(), UpdateRequest{
		Env:        env,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		ID:         "8523",
		Input:      map[string]any{"description": "updated"},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.PriceList["description"] != "updated" {
		t.Fatalf("unexpected update response: %#v", updated.PriceList)
	}

	deleted, err := service.Delete(context.Background(), DeleteRequest{
		Env:        env,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		ID:         "8523",
	})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted.Deleted {
		t.Fatalf("expected delete response to mark deleted: %#v", deleted)
	}

	plan, err := service.Create(context.Background(), CreateRequest{
		Env:        env,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		Input:      map[string]any{"name": "Plan"},
		DryRun:     true,
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

	deletePlan, err := service.Delete(context.Background(), DeleteRequest{
		Env:        env,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		ID:         "8523",
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
