package transport

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

func TestClientPostJSONQueryUsesQueryTokenForEmptyBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.URL.Query().Get("status"); got != "paid" {
			t.Fatalf("expected status query parameter, got %q", got)
		}
		if got := r.URL.Query().Get("api_token"); got != "token" {
			t.Fatalf("expected api_token query parameter, got %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token", 5*time.Second, 0, server.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if _, err := client.PostJSONQuery(context.Background(), "/invoices/1/change_status.json", url.Values{"status": {"paid"}}, nil, &map[string]any{}); err != nil {
		t.Fatalf("PostJSONQuery() error = %v", err)
	}
}

func TestClientPutJSONQueryPreservesBodyAndQuery(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		if got := r.URL.Query().Get("correction_positions"); got != "full" {
			t.Fatalf("expected correction_positions query parameter, got %q", got)
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
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token", 5*time.Second, 0, server.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if _, err := client.PutJSONQuery(context.Background(), "/invoices/1.json", url.Values{"correction_positions": {"full"}}, map[string]any{"invoice": map[string]any{"buyer_name": "Acme"}}, &map[string]any{}); err != nil {
		t.Fatalf("PutJSONQuery() error = %v", err)
	}
}

func TestUploadMultipartPostsFieldsAndFile(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("ParseMediaType() error = %v", err)
		}
		if !strings.HasPrefix(mediaType, "multipart/") {
			t.Fatalf("expected multipart content type, got %q", mediaType)
		}
		reader := multipart.NewReader(r.Body, params["boundary"])
		fields := map[string]string{}
		fileFound := false
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("NextPart() error = %v", err)
			}
			data, err := io.ReadAll(part)
			if err != nil {
				t.Fatalf("ReadAll(part) error = %v", err)
			}
			if part.FormName() == "file" {
				fileFound = true
				if part.FileName() != "scan.pdf" || string(data) != "pdf-bytes" {
					t.Fatalf("unexpected uploaded file: name=%q body=%q", part.FileName(), string(data))
				}
				continue
			}
			fields[part.FormName()] = string(data)
		}
		if !fileFound {
			t.Fatal("expected multipart file field")
		}
		if fields["key"] != "abc" || fields["policy"] != "xyz" {
			t.Fatalf("unexpected multipart fields: %#v", fields)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token", 5*time.Second, 0, server.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if _, err := client.UploadMultipart(context.Background(), MultipartUpload{
		URL:         server.URL,
		Fields:      map[string]string{"key": "abc", "policy": "xyz"},
		FileField:   "file",
		FileName:    "scan.pdf",
		FileContent: []byte("pdf-bytes"),
	}); err != nil {
		t.Fatalf("UploadMultipart() error = %v", err)
	}
}
