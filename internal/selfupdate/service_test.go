package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUpdateInstallsLatestRelease(t *testing.T) {
	t.Parallel()

	archive := testArchive(t, []byte("new-binary"))
	checksum := sha256.Sum256(archive)
	checksumText := fmt.Sprintf("%s  fakturownia_1.2.3_linux_amd64.tar.gz\n", hex.EncodeToString(checksum[:]))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/latest":
			fmt.Fprintf(w, `{"tag_name":"v1.2.3","html_url":"%s/release/v1.2.3","assets":[{"name":"fakturownia_1.2.3_linux_amd64.tar.gz","browser_download_url":"%s/assets/fakturownia_1.2.3_linux_amd64.tar.gz"},{"name":"checksums.txt","browser_download_url":"%s/assets/checksums.txt"}]}`, serverURL(r), serverURL(r), serverURL(r))
		case "/assets/fakturownia_1.2.3_linux_amd64.tar.gz":
			_, _ = w.Write(archive)
		case "/assets/checksums.txt":
			_, _ = w.Write([]byte(checksumText))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	executablePath := filepath.Join(dir, "fakturownia")
	if err := os.WriteFile(executablePath, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	service := NewService()
	service.releaseBaseURL = server.URL
	service.httpClient = server.Client()
	service.resolveExecPath = func() (string, error) { return executablePath, nil }
	service.detectPlatform = func() (string, string, error) { return "linux", "amd64", nil }

	result, err := service.Update(context.Background(), UpdateRequest{
		CurrentVersion: "v1.2.2",
		Timeout:        5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !result.Updated || result.TargetVersion != "v1.2.3" || !result.ChecksumVerified {
		t.Fatalf("unexpected result: %#v", result)
	}
	body, err := os.ReadFile(executablePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(body) != "new-binary" {
		t.Fatalf("expected updated binary contents, got %q", string(body))
	}
}

func TestUpdateDryRunDoesNotReplaceExecutable(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/tags/v1.2.3":
			fmt.Fprintf(w, `{"tag_name":"v1.2.3","html_url":"%s/release/v1.2.3","assets":[{"name":"fakturownia_1.2.3_linux_amd64.tar.gz","browser_download_url":"%s/assets/fakturownia_1.2.3_linux_amd64.tar.gz"},{"name":"checksums.txt","browser_download_url":"%s/assets/checksums.txt"}]}`, serverURL(r), serverURL(r), serverURL(r))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	executablePath := filepath.Join(dir, "fakturownia")
	if err := os.WriteFile(executablePath, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	service := NewService()
	service.releaseBaseURL = server.URL
	service.httpClient = server.Client()
	service.resolveExecPath = func() (string, error) { return executablePath, nil }
	service.detectPlatform = func() (string, string, error) { return "linux", "amd64", nil }

	result, err := service.Update(context.Background(), UpdateRequest{
		CurrentVersion: "v1.2.2",
		TargetVersion:  "1.2.3",
		DryRun:         true,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !result.DryRun || result.Updated || result.TargetVersion != "v1.2.3" {
		t.Fatalf("unexpected dry-run result: %#v", result)
	}
	body, err := os.ReadFile(executablePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(body) != "old-binary" {
		t.Fatalf("expected executable to remain unchanged, got %q", string(body))
	}
}

func TestUpdateReturnsAlreadyCurrent(t *testing.T) {
	t.Parallel()

	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		fmt.Fprintf(w, `{"tag_name":"v1.2.3","html_url":"%s/release/v1.2.3","assets":[{"name":"fakturownia_1.2.3_linux_amd64.tar.gz","browser_download_url":"%s/assets/fakturownia_1.2.3_linux_amd64.tar.gz"},{"name":"checksums.txt","browser_download_url":"%s/assets/checksums.txt"}]}`, serverURL(r), serverURL(r), serverURL(r))
	}))
	defer server.Close()

	service := NewService()
	service.releaseBaseURL = server.URL
	service.httpClient = server.Client()
	service.resolveExecPath = func() (string, error) { return "/tmp/fakturownia", nil }
	service.detectPlatform = func() (string, string, error) { return "linux", "amd64", nil }

	result, err := service.Update(context.Background(), UpdateRequest{
		CurrentVersion: "1.2.3",
		TargetVersion:  "latest",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !result.AlreadyCurrent || result.Updated {
		t.Fatalf("unexpected already-current result: %#v", result)
	}
	if hits != 1 {
		t.Fatalf("expected only metadata request, got %d hits", hits)
	}
}

func TestUpdateRejectsUnsupportedPlatform(t *testing.T) {
	t.Parallel()

	service := NewService()
	service.detectPlatform = func() (string, string, error) {
		return "", "", fmt.Errorf("unsupported")
	}

	if _, err := service.Update(context.Background(), UpdateRequest{}); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported platform error, got %v", err)
	}
}

func testArchive(t *testing.T, binaryBody []byte) []byte {
	t.Helper()

	var gz bytes.Buffer
	gzw := gzip.NewWriter(&gz)
	tw := tar.NewWriter(gzw)
	header := &tar.Header{
		Name: binName,
		Mode: 0o755,
		Size: int64(len(binaryBody)),
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatalf("WriteHeader() error = %v", err)
	}
	if _, err := tw.Write(binaryBody); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("Close() tar error = %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("Close() gzip error = %v", err)
	}
	return gz.Bytes()
}

func serverURL(r *http.Request) string {
	return "http://" + r.Host
}
