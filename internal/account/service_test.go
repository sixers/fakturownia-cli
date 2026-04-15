package account

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sixers/fakturownia-cli/internal/config"
)

type memoryStore struct {
	values map[string]string
}

func newMemoryStore() *memoryStore {
	return &memoryStore{values: map[string]string{}}
}

func (s *memoryStore) Get(name string) (string, error) {
	value, ok := s.values[name]
	if !ok {
		return "", config.ErrSecretNotFound
	}
	return value, nil
}

func (s *memoryStore) Set(name, value string) error {
	s.values[name] = value
	return nil
}

func (s *memoryStore) Delete(name string) error {
	delete(s.values, name)
	return nil
}

func TestCreateGetDeleteUnlinkAndDryRun(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store)

	var seen []string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/account.json":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if payload["api_token"] != "token" {
				t.Fatalf("expected api_token in create payload, got %#v", payload["api_token"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"prefix":    "acme",
				"url":       "https://acme.fakturownia.pl",
				"login":     "owner",
				"email":     "owner@example.com",
				"api_token": "new-token",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/account.json":
			if got := r.URL.Query().Get("api_token"); got != "token" {
				t.Fatalf("expected api_token query in get, got %q", got)
			}
			if got := r.URL.Query().Get("integration_token"); got != "partner" {
				t.Fatalf("expected integration_token query in get, got %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"prefix":    "acme",
				"url":       "https://acme.fakturownia.pl",
				"login":     "owner",
				"email":     "owner@example.com",
				"api_token": "present",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/account/delete.json":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if payload["api_token"] != "token" {
				t.Fatalf("expected api_token in delete payload, got %#v", payload["api_token"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"code": "ok", "message": "deleted"})
		case r.Method == http.MethodPatch && r.URL.Path == "/account/unlink.json":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if payload["api_token"] != "token" {
				t.Fatalf("expected api_token in unlink payload, got %#v", payload["api_token"])
			}
			prefixes, ok := payload["prefix"].([]any)
			if !ok || len(prefixes) != 2 {
				t.Fatalf("expected prefix array in unlink payload, got %#v", payload["prefix"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    "ok",
				"message": "unlinked",
				"result": map[string]any{
					"unlinked":     []string{"acme"},
					"not_unlinked": []string{"beta"},
				},
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	restore := swapDefaultTransport(server.Client().Transport)
	defer restore()

	env := config.Env{URL: server.URL, APIToken: "token"}
	configPath := filepath.Join(t.TempDir(), "config.json")

	created, err := service.Create(context.Background(), CreateRequest{
		ConfigPath: configPath,
		Env:        env,
		Timeout:    5 * time.Second,
		Input: map[string]any{
			"account": map[string]any{"prefix": "acme"},
			"user":    map[string]any{"login": "owner"},
			"company": map[string]any{"name": "Acme"},
		},
		SaveAs: "work",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !created.APITokenPresent || !created.TokenStored || created.SavedProfile != "work" {
		t.Fatalf("unexpected create response: %#v", created)
	}
	if token, _ := store.Get("work"); token != "new-token" {
		t.Fatalf("expected stored token new-token, got %q", token)
	}

	got, err := service.Get(context.Background(), GetRequest{
		Env:              env,
		Timeout:          5 * time.Second,
		IntegrationToken: "partner",
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Prefix != "acme" || !got.APITokenPresent {
		t.Fatalf("unexpected get response: %#v", got)
	}

	deleted, err := service.Delete(context.Background(), DeleteRequest{
		Env:     env,
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted.Message != "deleted" {
		t.Fatalf("unexpected delete response: %#v", deleted)
	}

	unlinked, err := service.Unlink(context.Background(), UnlinkRequest{
		Env:        env,
		Timeout:    5 * time.Second,
		Prefixes:   []string{"acme", "beta"},
		DryRun:     false,
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("Unlink() error = %v", err)
	}
	if len(unlinked.Result.Unlinked) != 1 || unlinked.Result.Unlinked[0] != "acme" {
		t.Fatalf("unexpected unlink response: %#v", unlinked)
	}

	plan, err := service.Unlink(context.Background(), UnlinkRequest{
		Env:      env,
		Timeout:  5 * time.Second,
		Prefixes: []string{"acme,beta"},
		DryRun:   true,
	})
	if err != nil {
		t.Fatalf("Unlink() dry-run error = %v", err)
	}
	if plan.DryRun == nil || plan.DryRun.Method != http.MethodPatch {
		t.Fatalf("unexpected dry-run plan: %#v", plan.DryRun)
	}
	body := plan.DryRun.Body.(map[string]any)
	if body["api_token"] != "[redacted]" {
		t.Fatalf("expected redacted token in dry-run body, got %#v", body["api_token"])
	}

	if joined := strings.Join(seen, ","); joined != "POST /account.json,GET /account.json,POST /account/delete.json,PATCH /account/unlink.json" {
		t.Fatalf("unexpected request sequence: %s", joined)
	}
}

func TestCreateFailsWhenSaveAsRequestedButNoTokenReturned(t *testing.T) {
	service := NewService(newMemoryStore())

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"prefix": "acme",
			"url":    "https://acme.fakturownia.pl",
			"login":  "owner",
			"email":  "owner@example.com",
		})
	}))
	defer server.Close()
	restore := swapDefaultTransport(server.Client().Transport)
	defer restore()

	_, err := service.Create(context.Background(), CreateRequest{
		Env:     config.Env{URL: server.URL, APIToken: "token"},
		Timeout: 5 * time.Second,
		Input:   map[string]any{"account": map[string]any{"prefix": "acme"}},
		SaveAs:  "work",
	})
	if err == nil {
		t.Fatal("expected missing token error")
	}
}

func swapDefaultTransport(transport http.RoundTripper) func() {
	previous := http.DefaultTransport
	http.DefaultTransport = transport
	return func() {
		http.DefaultTransport = previous
	}
}
