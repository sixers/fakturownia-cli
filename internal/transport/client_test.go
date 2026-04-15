package transport

import (
	"context"
	"io"
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

func TestClientPostJSONInjectsTokenIntoBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if payload["api_token"] != "token" {
			t.Fatalf("expected api_token in request body, got %#v", payload["api_token"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token", 5*time.Second, 0, server.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	var payload map[string]any
	if _, err := client.PostJSON(context.Background(), "/clients.json", map[string]any{"client": map[string]any{"name": "Acme"}}, &payload); err != nil {
		t.Fatalf("PostJSON() error = %v", err)
	}
}

func TestClientDeleteJSONUsesQueryToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if got := r.URL.Query().Get("api_token"); got != "token" {
			t.Fatalf("expected api_token query parameter, got %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token", 5*time.Second, 0, server.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if _, err := client.DeleteJSON(context.Background(), "/clients/1.json", nil, nil); err != nil {
		t.Fatalf("DeleteJSON() error = %v", err)
	}
}

func TestPostJSONDoesNotRetryOnServerError(t *testing.T) {
	t.Parallel()

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		http.Error(w, `{"message":"temporary"}`, http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token", 5*time.Second, 1, server.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	client.sleep = func(time.Duration) {}

	if _, err := client.PostJSON(context.Background(), "/clients.json", map[string]any{"client": map[string]any{"name": "Acme"}}, &map[string]any{}); err == nil {
		t.Fatal("expected server error")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}
