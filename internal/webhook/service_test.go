package webhook

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
			if r.URL.Path == "/webhooks.json" {
				if r.URL.Query().Get("page") != "2" || r.URL.Query().Get("per_page") != "25" {
					t.Fatalf("unexpected query: %s", r.URL.RawQuery)
				}
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 7, "kind": "invoice:create", "url": "https://example.com/hook", "active": true}})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 7, "kind": "invoice:create", "url": "https://example.com/hook", "active": true})
		case http.MethodPost:
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if payload["api_token"] != "token" || payload["kind"] != "invoice:create" {
				t.Fatalf("unexpected create payload: %#v", payload)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 7, "kind": payload["kind"], "url": payload["url"], "active": true})
		case http.MethodPut:
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if payload["api_token"] != "token" || payload["active"] != false {
				t.Fatalf("unexpected update payload: %#v", payload)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 7, "active": false})
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
		ID:      "7",
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Webhook["kind"] != "invoice:create" {
		t.Fatalf("unexpected get response: %#v", got.Webhook)
	}

	created, err := service.Create(context.Background(), CreateRequest{
		Env:     env,
		Timeout: 5 * time.Second,
		Input: map[string]any{
			"kind":   "invoice:create",
			"url":    "https://example.com/hook",
			"active": true,
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Webhook["id"] == nil {
		t.Fatalf("unexpected create response: %#v", created.Webhook)
	}

	updated, err := service.Update(context.Background(), UpdateRequest{
		Env:     env,
		Timeout: 5 * time.Second,
		ID:      "7",
		Input:   map[string]any{"active": false},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Webhook["active"] != false {
		t.Fatalf("unexpected update response: %#v", updated.Webhook)
	}

	deleted, err := service.Delete(context.Background(), DeleteRequest{
		Env:     env,
		Timeout: 5 * time.Second,
		ID:      "7",
	})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted.Deleted {
		t.Fatalf("expected deleted response: %#v", deleted)
	}

	plan, err := service.Create(context.Background(), CreateRequest{
		Env:     env,
		Timeout: 5 * time.Second,
		Input:   map[string]any{"kind": "invoice:create"},
		DryRun:  true,
	})
	if err != nil {
		t.Fatalf("Create() dry-run error = %v", err)
	}
	if plan.DryRun == nil || plan.DryRun.Method != http.MethodPost {
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
