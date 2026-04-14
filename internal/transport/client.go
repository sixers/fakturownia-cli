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
	resp, err := c.do(ctx, http.MethodGet, path, query, "application/json")
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
	return c.do(ctx, http.MethodGet, path, query, "application/pdf")
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, accept string) (*Response, error) {
	if query == nil {
		query = url.Values{}
	}
	query.Set("api_token", c.token)

	requestURL := *c.baseURL
	requestURL.Path = strings.TrimRight(c.baseURL.Path, "/") + path
	requestURL.RawQuery = query.Encode()

	requestID := newRequestID()
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), nil)
		if err != nil {
			return nil, output.Internal(err, "build upstream request")
		}
		req.Header.Set("Accept", accept)
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
		if resp.StatusCode >= 500 && attempt < c.maxRetries {
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
