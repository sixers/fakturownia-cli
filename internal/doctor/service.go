package doctor

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/sixers/fakturownia-cli/internal/config"
	"github.com/sixers/fakturownia-cli/internal/output"
	"github.com/sixers/fakturownia-cli/internal/transport"
)

type Service struct {
	store          config.ProbeableTokenStore
	releaseBaseURL string
	httpClient     *http.Client
}

type RunRequest struct {
	ConfigPath            string
	Profile               string
	Env                   config.Env
	Timeout               time.Duration
	MaxRetries            int
	Version               string
	CheckReleaseIntegrity bool
}

type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

type Report struct {
	Version string  `json:"version"`
	Status  string  `json:"status"`
	Checks  []Check `json:"checks"`
}

type RunResult struct {
	Report   Report
	Warnings []output.WarningDetail
	Profile  string
}

type releaseMetadata struct {
	Assets []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func NewService(store config.ProbeableTokenStore) *Service {
	return &Service{
		store:          store,
		releaseBaseURL: "https://api.github.com/repos/sixers/fakturownia-cli",
		httpClient:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *Service) Run(ctx context.Context, req RunRequest) (*RunResult, error) {
	if req.Timeout <= 0 {
		req.Timeout = 30 * time.Second
	}
	result := &RunResult{
		Report: Report{
			Version: req.Version,
			Status:  "ok",
		},
	}

	configPath, err := config.ResolveConfigPath(req.ConfigPath)
	if err != nil {
		return nil, err
	}
	result.Report.Checks = append(result.Report.Checks, Check{
		Name:    "config-path",
		Status:  "ok",
		Message: fmt.Sprintf("using config path %s", configPath),
	})

	if err := s.store.Probe(); err != nil {
		result.Report.Checks = append(result.Report.Checks, Check{
			Name:    "credential-store",
			Status:  "error",
			Message: err.Error(),
			Hint:    "verify that the configured credential store is available for this session",
		})
		result.Report.Status = "error"
		return result, nil
	}
	result.Report.Checks = append(result.Report.Checks, Check{
		Name:    "credential-store",
		Status:  "ok",
		Message: "credential store access succeeded",
	})

	resolved, resolveErr := config.Resolve(req.ConfigPath, req.Env, req.Profile, s.store)
	if resolveErr != nil {
		result.Report.Checks = append(result.Report.Checks, Check{
			Name:    "profile",
			Status:  "error",
			Message: resolveErr.Error(),
			Hint:    "run `fakturownia auth login --prefix <account> --api-token <token>` or set the documented environment variables",
		})
		result.Report.Status = "error"
		return result, nil
	}
	result.Profile = resolved.Name
	result.Report.Checks = append(result.Report.Checks, Check{
		Name:    "profile",
		Status:  "ok",
		Message: fmt.Sprintf("resolved profile %s", resolved.Name),
	})

	if dnsErr := dnsAndTLSCheck(resolved.URL, req.Timeout); dnsErr != nil {
		result.Report.Checks = append(result.Report.Checks, Check{
			Name:    "reachability",
			Status:  "error",
			Message: dnsErr.Error(),
			Hint:    "verify DNS, outbound network access, and TLS interception settings",
		})
		result.Report.Status = "error"
	} else {
		result.Report.Checks = append(result.Report.Checks, Check{
			Name:    "reachability",
			Status:  "ok",
			Message: "DNS and TLS reachability succeeded",
		})
	}

	client, err := transport.NewClient(resolved.URL, resolved.Token, req.Timeout, req.MaxRetries, nil)
	if err != nil {
		return nil, err
	}
	var account map[string]any
	if _, err := client.GetJSON(ctx, "/account.json", nil, &account); err != nil {
		result.Report.Checks = append(result.Report.Checks, Check{
			Name:    "authenticated-api",
			Status:  "error",
			Message: err.Error(),
			Hint:    "verify the account URL and API token for the selected profile",
		})
		result.Report.Status = "error"
	} else {
		result.Report.Checks = append(result.Report.Checks, Check{
			Name:    "authenticated-api",
			Status:  "ok",
			Message: "authenticated API access succeeded",
		})
	}

	if req.CheckReleaseIntegrity {
		check, warning := s.releaseIntegrityCheck(req.Version)
		result.Report.Checks = append(result.Report.Checks, check)
		if warning != nil {
			result.Warnings = append(result.Warnings, *warning)
			if result.Report.Status == "ok" {
				result.Report.Status = "warning"
			}
		} else if check.Status == "error" {
			result.Report.Status = "error"
		}
	}

	return result, nil
}

func dnsAndTLSCheck(accountURL string, timeout time.Duration) error {
	parsed, err := url.Parse(accountURL)
	if err != nil {
		return err
	}
	host := parsed.Hostname()
	if _, err := net.LookupHost(host); err != nil {
		return err
	}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: timeout}, "tcp", net.JoinHostPort(host, "443"), &tls.Config{
		ServerName: host,
	})
	if err != nil {
		return err
	}
	return conn.Close()
}

func (s *Service) releaseIntegrityCheck(version string) (Check, *output.WarningDetail) {
	if version == "" || version == "dev" {
		return Check{
				Name:    "release-integrity",
				Status:  "skipped",
				Message: "release integrity check skipped for a development build",
				Hint:    "use a tagged GitHub Release build to enable checksum verification",
			}, &output.WarningDetail{
				Code:    "release_integrity_skipped",
				Message: "release integrity check is unavailable for development builds",
			}
	}

	tag := version
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/releases/tags/%s", s.releaseBaseURL, tag), nil)
	if err != nil {
		return Check{Name: "release-integrity", Status: "error", Message: err.Error()}, nil
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return Check{
				Name:    "release-integrity",
				Status:  "skipped",
				Message: err.Error(),
				Hint:    "verify network access to api.github.com",
			}, &output.WarningDetail{
				Code:    "release_integrity_skipped",
				Message: "release integrity metadata could not be loaded",
			}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return Check{
				Name:    "release-integrity",
				Status:  "skipped",
				Message: fmt.Sprintf("release metadata is unavailable for %s", tag),
				Hint:    "publish a GitHub Release before using --check-release-integrity",
			}, &output.WarningDetail{
				Code:    "release_integrity_skipped",
				Message: "release integrity metadata was not found for the current version",
			}
	}

	var metadata releaseMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return Check{Name: "release-integrity", Status: "error", Message: err.Error()}, nil
	}

	assetURL := ""
	for _, asset := range metadata.Assets {
		if asset.Name == "binary-checksums.txt" {
			assetURL = asset.BrowserDownloadURL
			break
		}
	}
	if assetURL == "" {
		return Check{
				Name:    "release-integrity",
				Status:  "skipped",
				Message: "binary-checksums.txt is not published for this release",
				Hint:    "publish the release workflow to enable checksum verification",
			}, &output.WarningDetail{
				Code:    "release_integrity_skipped",
				Message: "binary checksum metadata is not available for this release",
			}
	}

	fileReq, err := http.NewRequest(http.MethodGet, assetURL, nil)
	if err != nil {
		return Check{Name: "release-integrity", Status: "error", Message: err.Error()}, nil
	}
	fileResp, err := s.httpClient.Do(fileReq)
	if err != nil {
		return Check{
				Name:    "release-integrity",
				Status:  "skipped",
				Message: err.Error(),
				Hint:    "verify network access to GitHub release assets",
			}, &output.WarningDetail{
				Code:    "release_integrity_skipped",
				Message: "binary checksum asset could not be downloaded",
			}
	}
	defer fileResp.Body.Close()
	if fileResp.StatusCode >= 400 {
		return Check{
				Name:    "release-integrity",
				Status:  "skipped",
				Message: "binary checksum asset is unavailable",
			}, &output.WarningDetail{
				Code:    "release_integrity_skipped",
				Message: "binary checksum asset is unavailable for this release",
			}
	}

	expected, err := io.ReadAll(fileResp.Body)
	if err != nil {
		return Check{Name: "release-integrity", Status: "error", Message: err.Error()}, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return Check{Name: "release-integrity", Status: "error", Message: err.Error()}, nil
	}
	binary, err := os.ReadFile(executable)
	if err != nil {
		return Check{Name: "release-integrity", Status: "error", Message: err.Error()}, nil
	}
	actualDigest := sha256.Sum256(binary)
	actual := hex.EncodeToString(actualDigest[:])
	if !strings.Contains(string(expected), actual) {
		return Check{
			Name:    "release-integrity",
			Status:  "error",
			Message: "running binary checksum does not match the published release metadata",
			Hint:    "reinstall the binary from GitHub Releases",
		}, nil
	}
	return Check{
		Name:    "release-integrity",
		Status:  "ok",
		Message: "running binary checksum matches the published release metadata",
	}, nil
}
