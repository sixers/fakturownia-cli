package warehouseaction

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sixers/fakturownia-cli/internal/config"
)

type stubTokenStore struct{}

func (stubTokenStore) Get(string) (string, error) { return "", config.ErrSecretNotFound }

func TestListBuildsQueryAndPagination(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		assertQuery(t, query, "page", "2")
		assertQuery(t, query, "per_page", "25")
		assertQuery(t, query, "warehouse_id", "1")
		assertQuery(t, query, "kind", "mm")
		assertQuery(t, query, "product_id", "7")
		assertQuery(t, query, "date_from", "2026-04-01")
		assertQuery(t, query, "date_to", "2026-04-15")
		assertQuery(t, query, "from_warehouse_document", "10")
		assertQuery(t, query, "to_warehouse_document", "11")
		assertQuery(t, query, "warehouse_document_id", "15")
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 77, "kind": "mm", "product_id": 7, "quantity": "2.0"}})
	}))
	defer server.Close()
	restore := swapDefaultTransport(server.Client().Transport)
	defer restore()

	service := NewService(stubTokenStore{})
	result, err := service.List(context.Background(), ListRequest{
		Env:                   config.Env{URL: server.URL, APIToken: "token"},
		Timeout:               5 * time.Second,
		MaxRetries:            1,
		Page:                  2,
		PerPage:               25,
		WarehouseID:           "1",
		Kind:                  "mm",
		ProductID:             "7",
		DateFrom:              "2026-04-01",
		DateTo:                "2026-04-15",
		FromWarehouseDocument: "10",
		ToWarehouseDocument:   "11",
		WarehouseDocumentID:   "15",
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Pagination.Page != 2 || result.Pagination.PerPage != 25 || result.Pagination.Returned != 1 || result.Pagination.HasNext {
		t.Fatalf("unexpected pagination: %#v", result.Pagination)
	}
}

func assertQuery(t *testing.T, query map[string][]string, key, want string) {
	t.Helper()
	got := ""
	if values := query[key]; len(values) > 0 {
		got = values[0]
	}
	if got != want {
		t.Fatalf("unexpected query %s: got %q want %q", key, got, want)
	}
}

func swapDefaultTransport(transport http.RoundTripper) func() {
	previous := http.DefaultTransport
	http.DefaultTransport = transport
	return func() {
		http.DefaultTransport = previous
	}
}
