package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/sixers/fakturownia-cli/internal/config"
	"github.com/sixers/fakturownia-cli/internal/output"
)

func TestExchangeSavesReturnedTokenByDefault(t *testing.T) {
	store := NewMemoryStore()
	service := NewService(store)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/login.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		if _, ok := payload["api_token"]; ok {
			t.Fatalf("did not expect api_token in tokenless exchange payload: %#v", payload)
		}
		if payload["login"] != "user@example.com" || payload["password"] != "secret" {
			t.Fatalf("unexpected exchange payload: %#v", payload)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"login":      "user@example.com",
			"email":      "user@example.com",
			"prefix":     "acme",
			"url":        "https://acme.fakturownia.pl",
			"first_name": "Ada",
			"last_name":  "Lovelace",
			"api_token":  "secret-token",
		})
	}))
	defer server.Close()

	service.httpClient = rewriteClient(server)
	configPath := filepath.Join(t.TempDir(), "config.json")

	result, err := service.Exchange(context.Background(), ExchangeRequest{
		ConfigPath: configPath,
		Login:      "user@example.com",
		Password:   "secret",
		Timeout:    5 * time.Second,
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("Exchange() error = %v", err)
	}
	if !result.APITokenPresent || !result.TokenStored {
		t.Fatalf("expected stored token metadata, got %#v", result)
	}
	if result.SavedProfile != "acme" {
		t.Fatalf("expected default saved profile acme, got %#v", result)
	}
	token, err := store.Get("acme")
	if err != nil {
		t.Fatalf("Get(acme) error = %v", err)
	}
	if token != "secret-token" {
		t.Fatalf("unexpected stored token %q", token)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Load(%s) error = %v", configPath, err)
	}
	if cfg.DefaultProfile != "acme" {
		t.Fatalf("expected default profile acme, got %#v", cfg.DefaultProfile)
	}
}

func TestExchangeHonorsSaveAs(t *testing.T) {
	store := NewMemoryStore()
	service := NewService(store)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"login":     "user@example.com",
			"email":     "user@example.com",
			"prefix":    "upstream",
			"url":       "https://upstream.fakturownia.pl",
			"api_token": "secret-token",
		})
	}))
	defer server.Close()

	service.httpClient = rewriteClient(server)
	configPath := filepath.Join(t.TempDir(), "config.json")

	result, err := service.Exchange(context.Background(), ExchangeRequest{
		ConfigPath: configPath,
		Login:      "user@example.com",
		Password:   "secret",
		SaveAs:     "work",
		Timeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Exchange() error = %v", err)
	}
	if result.SavedProfile != "work" {
		t.Fatalf("expected saved profile work, got %#v", result)
	}
	token, err := store.Get("work")
	if err != nil {
		t.Fatalf("Get(work) error = %v", err)
	}
	if token != "secret-token" {
		t.Fatalf("unexpected stored token %q", token)
	}
}

func TestExchangeFailsWhenNoTokenReturned(t *testing.T) {
	store := NewMemoryStore()
	service := NewService(store)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"login":  "user@example.com",
			"email":  "user@example.com",
			"prefix": "acme",
			"url":    "https://acme.fakturownia.pl",
		})
	}))
	defer server.Close()

	service.httpClient = rewriteClient(server)

	_, err := service.Exchange(context.Background(), ExchangeRequest{
		Login:    "user@example.com",
		Password: "secret",
		Timeout:  5 * time.Second,
	})
	if err == nil {
		t.Fatal("expected missing token error")
	}
	appErr := output.AsAppError(err)
	if appErr.Detail().Code != "missing_api_token" {
		t.Fatalf("expected missing_api_token, got %#v", appErr.Detail())
	}
	if len(appErr.RawBody()) == 0 {
		t.Fatal("expected raw body on missing token error")
	}
}

func rewriteClient(server *httptest.Server) *http.Client {
	target, err := url.Parse(server.URL)
	if err != nil {
		panic(err)
	}
	base := server.Client()
	return &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			clone := req.Clone(req.Context())
			clone.URL.Scheme = target.Scheme
			clone.URL.Host = target.Host
			clone.Host = target.Host
			return base.Transport.RoundTrip(clone)
		}),
		Timeout: base.Timeout,
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
