package recurring

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/sixers/fakturownia-cli/internal/config"
)

func useTLSTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	server := httptest.NewTLSServer(handler)
	oldTransport := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	t.Cleanup(func() {
		http.DefaultTransport = oldTransport
		server.Close()
	})
	return server
}

func TestListReadsRecurringDefinitions(t *testing.T) {
	server := useTLSTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/recurrings.json" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("api_token"); got != "token" {
			t.Fatalf("unexpected api_token %q", got)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": 7, "name": "Monthly", "every": "1m"},
		})
	})

	service := NewService(nil)
	result, err := service.List(context.Background(), ListRequest{
		ConfigPath: filepath.Join(t.TempDir(), "config.json"),
		Env:        config.Env{URL: server.URL, APIToken: "token"},
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result.Recurrings) != 1 || result.Recurrings[0]["name"] != "Monthly" {
		t.Fatalf("unexpected list result: %#v", result)
	}
}

func TestCreateDryRunWrapsRecurringObject(t *testing.T) {
	service := NewService(nil)
	result, err := service.Create(context.Background(), CreateRequest{
		ConfigPath: filepath.Join(t.TempDir(), "config.json"),
		Env:        config.Env{URL: "https://acme.fakturownia.pl", APIToken: "token"},
		Input:      map[string]any{"name": "Monthly", "invoice_id": 1},
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.DryRun == nil {
		t.Fatal("expected dry-run plan")
	}
	body, _ := result.DryRun.Body.(map[string]any)
	recurring, _ := body["recurring"].(map[string]any)
	if recurring["name"] != "Monthly" || body["api_token"] != "[redacted]" {
		t.Fatalf("unexpected dry-run payload: %#v", body)
	}
}

func TestUpdateSendsWrappedPayload(t *testing.T) {
	server := useTLSTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("unexpected method %q", r.Method)
		}
		if r.URL.Path != "/recurrings/9.json" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		recurring, _ := payload["recurring"].(map[string]any)
		if recurring["next_invoice_date"] != "2026-05-01" {
			t.Fatalf("unexpected recurring payload: %#v", payload)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 9, "next_invoice_date": "2026-05-01"})
	})

	service := NewService(nil)
	result, err := service.Update(context.Background(), UpdateRequest{
		ConfigPath: filepath.Join(t.TempDir(), "config.json"),
		Env:        config.Env{URL: server.URL, APIToken: "token"},
		ID:         "9",
		Input:      map[string]any{"next_invoice_date": "2026-05-01"},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if result.Recurring["next_invoice_date"] != "2026-05-01" {
		t.Fatalf("unexpected update result: %#v", result)
	}
}

func TestDeleteAndDeleteDryRun(t *testing.T) {
	server := useTLSTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected method %q", r.Method)
		}
		if r.URL.Path != "/recurrings/9.json" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})

	service := NewService(nil)
	deleted, err := service.Delete(context.Background(), DeleteRequest{
		ConfigPath: filepath.Join(t.TempDir(), "config.json"),
		Env:        config.Env{URL: server.URL, APIToken: "token"},
		ID:         "9",
	})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted.Deleted || deleted.ID != "9" {
		t.Fatalf("unexpected delete result: %#v", deleted)
	}

	plan, err := service.Delete(context.Background(), DeleteRequest{
		ConfigPath: filepath.Join(t.TempDir(), "config.json"),
		Env:        config.Env{URL: "https://acme.fakturownia.pl", APIToken: "token"},
		ID:         "9",
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("Delete() dry-run error = %v", err)
	}
	if plan.DryRun == nil || plan.DryRun.Method != http.MethodDelete || plan.DryRun.Path != "/recurrings/9.json" {
		t.Fatalf("unexpected dry-run plan: %#v", plan.DryRun)
	}
}
