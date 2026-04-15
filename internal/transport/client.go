package transport

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sixers/fakturownia-cli/internal/output"
)

type Client struct {
	baseURL    *url.URL
	token      string
	httpClient *http.Client
	maxRetries int
	sleep      func(time.Duration)
}

type Response struct {
	StatusCode int
	RequestID  string
	RawBody    []byte
	Header     http.Header
}

type RequestPlan struct {
	Method string              `json:"method"`
	Path   string              `json:"path"`
	Query  map[string][]string `json:"query,omitempty"`
	Body   any                 `json:"body,omitempty"`
}

type MultipartUpload struct {
	URL             string
	Fields          map[string]string
	FileField       string
	FileName        string
	FileContent     []byte
	FileContentType string
}

type MultipartUploadPlan struct {
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Fields    map[string]string `json:"fields,omitempty"`
	FileField string            `json:"file_field"`
	FileName  string            `json:"file_name"`
	Bytes     int               `json:"bytes"`
}

func NewClient(baseURL, token string, timeout time.Duration, maxRetries int, httpClient *http.Client) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, output.Internal(err, "parse base URL")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	} else if timeout > 0 {
		httpClient.Timeout = timeout
	}
	return &Client{
		baseURL:    parsed,
		token:      token,
		httpClient: httpClient,
		maxRetries: maxRetries,
		sleep:      time.Sleep,
	}, nil
}

func (c *Client) GetJSON(ctx context.Context, path string, query url.Values, dest any) (*Response, error) {
	resp, err := c.do(ctx, requestOptions{
		Method:    http.MethodGet,
		Path:      path,
		Query:     query,
		Accept:    "application/json",
		Retryable: true,
	})
	if err != nil {
		return nil, err
	}
	if dest != nil {
		dec := json.NewDecoder(bytes.NewReader(resp.RawBody))
		dec.UseNumber()
		if err := dec.Decode(dest); err != nil {
			return nil, output.Internal(err, "decode upstream JSON response")
		}
	}
	return resp, nil
}

func (c *Client) GetBinary(ctx context.Context, path string, query url.Values) (*Response, error) {
	return c.GetContent(ctx, path, query, "application/pdf")
}

func (c *Client) GetContent(ctx context.Context, path string, query url.Values, accept string) (*Response, error) {
	if strings.TrimSpace(accept) == "" {
		accept = "*/*"
	}
	return c.do(ctx, requestOptions{
		Method:    http.MethodGet,
		Path:      path,
		Query:     query,
		Accept:    accept,
		Retryable: true,
	})
}

func (c *Client) PostJSON(ctx context.Context, path string, payload map[string]any, dest any) (*Response, error) {
	return c.doJSON(ctx, http.MethodPost, path, nil, payload, dest)
}

func (c *Client) PostJSONQuery(ctx context.Context, path string, query url.Values, payload map[string]any, dest any) (*Response, error) {
	return c.doJSON(ctx, http.MethodPost, path, query, payload, dest)
}

func (c *Client) PutJSON(ctx context.Context, path string, payload map[string]any, dest any) (*Response, error) {
	return c.doJSON(ctx, http.MethodPut, path, nil, payload, dest)
}

func (c *Client) PutJSONQuery(ctx context.Context, path string, query url.Values, payload map[string]any, dest any) (*Response, error) {
	return c.doJSON(ctx, http.MethodPut, path, query, payload, dest)
}

func (c *Client) PatchJSON(ctx context.Context, path string, payload map[string]any, dest any) (*Response, error) {
	return c.doJSON(ctx, http.MethodPatch, path, nil, payload, dest)
}

func (c *Client) PatchJSONQuery(ctx context.Context, path string, query url.Values, payload map[string]any, dest any) (*Response, error) {
	return c.doJSON(ctx, http.MethodPatch, path, query, payload, dest)
}

func (c *Client) DeleteJSON(ctx context.Context, path string, payload map[string]any, dest any) (*Response, error) {
	return c.doJSON(ctx, http.MethodDelete, path, nil, payload, dest)
}

func PlanJSONRequest(method, path string, query url.Values, payload map[string]any) RequestPlan {
	plannedQuery := cloneQuery(query)
	plannedBody := cloneMap(payload)

	if plannedBody != nil {
		plannedBody["api_token"] = "[redacted]"
	}
	if plannedQuery != nil || method == http.MethodDelete {
		if plannedQuery == nil {
			plannedQuery = url.Values{}
		}
		plannedQuery.Set("api_token", "[redacted]")
	}

	return RequestPlan{
		Method: method,
		Path:   path,
		Query:  valuesToMap(plannedQuery),
		Body:   plannedBody,
	}
}

func PlanMultipartUpload(upload MultipartUpload) MultipartUploadPlan {
	fields := make(map[string]string, len(upload.Fields))
	for key, value := range upload.Fields {
		fields[key] = value
	}
	return MultipartUploadPlan{
		Method:    http.MethodPost,
		URL:       upload.URL,
		Fields:    fields,
		FileField: upload.FileField,
		FileName:  upload.FileName,
		Bytes:     len(upload.FileContent),
	}
}

type requestOptions struct {
	Method      string
	Path        string
	Query       url.Values
	Accept      string
	ContentType string
	Body        []byte
	Retryable   bool
}

func (c *Client) UploadMultipart(ctx context.Context, upload MultipartUpload) (*Response, error) {
	if strings.TrimSpace(upload.URL) == "" {
		return nil, output.Usage("missing_upload_url", "upload URL is required", "retry fetching attachment credentials")
	}
	fileField := strings.TrimSpace(upload.FileField)
	if fileField == "" {
		fileField = "file"
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range upload.Fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, output.Internal(err, "write multipart field")
		}
	}
	part, err := writer.CreateFormFile(fileField, upload.FileName)
	if err != nil {
		return nil, output.Internal(err, "create multipart file field")
	}
	if _, err := part.Write(upload.FileContent); err != nil {
		return nil, output.Internal(err, "write multipart file body")
	}
	if err := writer.Close(); err != nil {
		return nil, output.Internal(err, "close multipart request body")
	}

	requestID := newRequestID()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upload.URL, bytes.NewReader(body.Bytes()))
	if err != nil {
		return nil, output.Internal(err, "build multipart upload request")
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "fakturownia-cli/dev")
	req.Header.Set("X-Request-ID", requestID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, output.Network("request_failed", err.Error(), "verify network access and retry with a higher --timeout-ms if needed", shouldRetryTransport(err)).WithCause(err)
	}
	rawBody, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		return nil, output.Internal(readErr, "read multipart upload response body")
	}
	if closeErr != nil {
		return nil, output.Internal(closeErr, "close multipart upload response body")
	}

	response := &Response{
		StatusCode: resp.StatusCode,
		RequestID:  headerRequestID(resp.Header, requestID),
		RawBody:    rawBody,
		Header:     resp.Header.Clone(),
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return response, nil
	}
	return nil, mapHTTPError(resp.StatusCode, rawBody).WithRawBody(rawBody)
}

func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, payload map[string]any, dest any) (*Response, error) {
	body := cloneMap(payload)
	if body != nil {
		body["api_token"] = c.token
	}

	opts := requestOptions{
		Method:    method,
		Path:      path,
		Query:     query,
		Accept:    "application/json",
		Retryable: method == http.MethodGet,
	}
	if len(body) > 0 {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, output.Internal(err, "encode upstream JSON request body")
		}
		opts.Body = raw
		opts.ContentType = "application/json"
	} else if method == http.MethodDelete {
		opts.Query = url.Values{}
		opts.Query.Set("api_token", c.token)
	}

	resp, err := c.do(ctx, opts)
	if err != nil {
		return nil, err
	}
	if dest == nil || len(resp.RawBody) == 0 {
		return resp, nil
	}
	dec := json.NewDecoder(bytes.NewReader(resp.RawBody))
	dec.UseNumber()
	if err := dec.Decode(dest); err != nil {
		return nil, output.Internal(err, "decode upstream JSON response")
	}
	return resp, nil
}

func (c *Client) do(ctx context.Context, opts requestOptions) (*Response, error) {
	query := cloneQuery(opts.Query)
	if query == nil {
		query = url.Values{}
	}
	if opts.Method == http.MethodGet || len(opts.Body) == 0 {
		query.Set("api_token", c.token)
	}

	requestURL := *c.baseURL
	requestURL.Path = strings.TrimRight(c.baseURL.Path, "/") + opts.Path
	requestURL.RawQuery = query.Encode()

	requestID := newRequestID()
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, opts.Method, requestURL.String(), bytes.NewReader(opts.Body))
		if err != nil {
			return nil, output.Internal(err, "build upstream request")
		}
		req.Header.Set("Accept", opts.Accept)
		if opts.ContentType != "" {
			req.Header.Set("Content-Type", opts.ContentType)
		}
		req.Header.Set("User-Agent", "fakturownia-cli/dev")
		req.Header.Set("X-Request-ID", requestID)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if shouldRetryTransport(err) && attempt < c.maxRetries {
				if waitErr := c.wait(ctx, attempt); waitErr != nil {
					return nil, waitErr
				}
				continue
			}
			return nil, output.Network("request_failed", err.Error(), "verify network access and retry with a higher --timeout-ms if needed", shouldRetryTransport(err)).WithCause(err)
		}

		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return nil, output.Internal(readErr, "read upstream response body")
		}
		if closeErr != nil {
			return nil, output.Internal(closeErr, "close upstream response body")
		}

		response := &Response{
			StatusCode: resp.StatusCode,
			RequestID:  headerRequestID(resp.Header, requestID),
			RawBody:    body,
			Header:     resp.Header.Clone(),
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return response, nil
		}

		appErr := mapHTTPError(resp.StatusCode, body)
		if opts.Retryable && resp.StatusCode >= 500 && attempt < c.maxRetries {
			lastErr = appErr
			if waitErr := c.wait(ctx, attempt); waitErr != nil {
				return nil, waitErr
			}
			continue
		}
		return nil, appErr.WithRawBody(body)
	}

	return nil, output.Internal(lastErr, "request failed")
}

func (c *Client) wait(ctx context.Context, attempt int) error {
	backoff := time.Duration(200*(1<<attempt)) * time.Millisecond
	timer := time.NewTimer(backoff)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return output.Network("timeout", ctx.Err().Error(), "increase --timeout-ms or lower --max-retries", true).WithCause(ctx.Err())
	case <-timer.C:
		return nil
	}
}

func shouldRetryTransport(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF)
}

func headerRequestID(header http.Header, fallback string) string {
	for _, key := range []string{"X-Request-Id", "X-Request-ID"} {
		if value := strings.TrimSpace(header.Get(key)); value != "" {
			return value
		}
	}
	return fallback
}

func mapHTTPError(statusCode int, body []byte) *output.AppError {
	message := parseErrorMessage(body)
	if message == "" {
		message = fmt.Sprintf("upstream request failed with status %d", statusCode)
	}

	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return output.AuthFailure("auth_failed", message, "verify the stored API token and account URL")
	case http.StatusNotFound:
		return output.NotFound("not_found", message, "verify the resource ID and account prefix")
	case http.StatusConflict:
		return output.Conflict("conflict", message, "refresh the resource state and retry")
	default:
		retryable := statusCode >= 500
		return output.Remote("upstream_error", message, "inspect the upstream response with --raw when supported", retryable)
	}
}

func parseErrorMessage(body []byte) string {
	var generic map[string]any
	if err := json.Unmarshal(body, &generic); err != nil {
		return strings.TrimSpace(string(body))
	}
	for _, key := range []string{"message", "error", "errors"} {
		if value, ok := generic[key]; ok {
			return strings.TrimSpace(fmt.Sprint(value))
		}
	}
	return ""
}

func newRequestID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func cloneQuery(query url.Values) url.Values {
	if query == nil {
		return nil
	}
	cloned := make(url.Values, len(query))
	for key, values := range query {
		copied := make([]string, len(values))
		copy(copied, values)
		cloned[key] = copied
	}
	return cloned
}

func valuesToMap(values url.Values) map[string][]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string][]string, len(values))
	for key, raw := range values {
		copied := make([]string, len(raw))
		copy(copied, raw)
		out[key] = copied
	}
	return out
}

func cloneMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	out := make(map[string]any, len(value))
	for key, child := range value {
		out[key] = cloneValue(child)
	}
	return out
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMap(typed)
	case []any:
		out := make([]any, len(typed))
		for idx, child := range typed {
			out[idx] = cloneValue(child)
		}
		return out
	default:
		return typed
	}
}
