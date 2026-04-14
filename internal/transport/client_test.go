package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientRetriesOnServerError(t *testing.T) {
	t.Parallel()

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			http.Error(w, `{"message":"temporary"}`, http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token", 5*time.Second, 1, server.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	client.sleep = func(time.Duration) {}

	var payload map[string]any
	if _, err := client.GetJSON(context.Background(), "/invoices.json", nil, &payload); err != nil {
		t.Fatalf("GetJSON() error = %v", err)
	}

	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestClientMapsNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"missing"}`, http.StatusNotFound)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token", 5*time.Second, 0, server.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if _, err := client.GetJSON(context.Background(), "/invoices/1.json", nil, &map[string]any{}); err == nil {
		t.Fatal("expected not found error")
	}
}
