package doctor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sixers/fakturownia-cli/internal/auth"
)

func TestReleaseIntegrityCheckSkippedForDev(t *testing.T) {
	t.Parallel()

	service := NewService(auth.NewMemoryStore())
	check, warning := service.releaseIntegrityCheck("dev")
	if check.Status != "skipped" {
		t.Fatalf("expected skipped status, got %q", check.Status)
	}
	if warning == nil {
		t.Fatal("expected warning for dev build")
	}
}

func TestReleaseIntegrityCheckMatchesChecksum(t *testing.T) {
	t.Parallel()

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	body, err := os.ReadFile(executable)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	digest := sha256.Sum256(body)
	hash := hex.EncodeToString(digest[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/tags/v1.2.3":
			fmt.Fprintf(w, `{"assets":[{"name":"binary-checksums.txt","browser_download_url":"%s/asset"}]}`, serverURL(r))
		case "/asset":
			fmt.Fprintf(w, "%s  fakturownia\n", hash)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := NewService(auth.NewMemoryStore())
	service.releaseBaseURL = server.URL
	service.httpClient = server.Client()

	check, warning := service.releaseIntegrityCheck("1.2.3")
	if warning != nil {
		t.Fatalf("expected nil warning, got %+v", warning)
	}
	if check.Status != "ok" {
		t.Fatalf("expected ok status, got %q", check.Status)
	}
}

func serverURL(r *http.Request) string {
	return "http://" + r.Host
}
