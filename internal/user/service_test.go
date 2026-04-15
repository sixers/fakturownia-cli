package user

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

func TestCreateBuildsWrappedPayloadAndDryRun(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/account/add_user.json" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		if payload["api_token"] != "token" || payload["integration_token"] != "partner" {
			t.Fatalf("unexpected top-level payload %#v", payload)
		}
		userPayload, ok := payload["user"].(map[string]any)
		if !ok || userPayload["email"] != "user@example.com" {
			t.Fatalf("unexpected wrapped user payload %#v", payload["user"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "user_id": 12})
	}))
	defer server.Close()
	restore := swapDefaultTransport(server.Client().Transport)
	defer restore()

	service := NewService(stubTokenStore{})
	env := config.Env{URL: server.URL, APIToken: "token"}

	created, err := service.Create(context.Background(), CreateRequest{
		Env:              env,
		Timeout:          5 * time.Second,
		IntegrationToken: "partner",
		Input: map[string]any{
			"invite": true,
			"email":  "user@example.com",
			"role":   "member",
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Response["status"] != "ok" {
		t.Fatalf("unexpected create response: %#v", created.Response)
	}

	plan, err := service.Create(context.Background(), CreateRequest{
		Env:              env,
		Timeout:          5 * time.Second,
		IntegrationToken: "partner",
		Input:            map[string]any{"email": "user@example.com"},
		DryRun:           true,
	})
	if err != nil {
		t.Fatalf("Create() dry-run error = %v", err)
	}
	if plan.DryRun == nil || plan.DryRun.Method != http.MethodPost {
		t.Fatalf("unexpected dry-run plan: %#v", plan.DryRun)
	}
}

func swapDefaultTransport(transport http.RoundTripper) func() {
	previous := http.DefaultTransport
	http.DefaultTransport = transport
	return func() {
		http.DefaultTransport = previous
	}
}
