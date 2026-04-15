package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sixers/fakturownia-cli/internal/output"
)

const (
	repoBaseURL   = "https://api.github.com/repos/sixers/fakturownia-cli"
	checksumAsset = "checksums.txt"
	binName       = "fakturownia"
)

type Service struct {
	releaseBaseURL  string
	httpClient      *http.Client
	resolveExecPath func() (string, error)
	detectPlatform  func() (string, string, error)
}

type UpdateRequest struct {
	CurrentVersion string
	TargetVersion  string
	Timeout        time.Duration
	DryRun         bool
}

type UpdateResult struct {
	RequestedVersion string `json:"requested_version"`
	CurrentVersion   string `json:"current_version"`
	TargetVersion    string `json:"target_version"`
	ExecutablePath   string `json:"executable_path"`
	OS               string `json:"os"`
	Arch             string `json:"arch"`
	ReleaseURL       string `json:"release_url"`
	AssetName        string `json:"asset_name"`
	DownloadURL      string `json:"download_url"`
	ChecksumURL      string `json:"checksum_url"`
	AlreadyCurrent   bool   `json:"already_current"`
	Updated          bool   `json:"updated"`
	DryRun           bool   `json:"dry_run"`
	ChecksumVerified bool   `json:"checksum_verified"`
}

type releaseMetadata struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func NewService() *Service {
	return &Service{
		releaseBaseURL: repoBaseURL,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		resolveExecPath: func() (string, error) {
			executable, err := os.Executable()
			if err != nil {
				return "", err
			}
			resolved, err := filepath.EvalSymlinks(executable)
			if err == nil {
				return resolved, nil
			}
			return executable, nil
		},
		detectPlatform: detectPlatform,
	}
}

func (s *Service) Update(ctx context.Context, req UpdateRequest) (*UpdateResult, error) {
	if req.Timeout > 0 {
		s.httpClient.Timeout = req.Timeout
	}

	executablePath, err := s.resolveExecPath()
	if err != nil {
		return nil, output.Internal(err, "resolve executable path")
	}
	platformOS, platformArch, err := s.detectPlatform()
	if err != nil {
		return nil, err
	}

	metadata, err := s.fetchReleaseMetadata(ctx, req.TargetVersion)
	if err != nil {
		return nil, err
	}
	targetTag := normalizeTag(metadata.TagName)
	versionNoV := strings.TrimPrefix(targetTag, "v")
	archiveName := fmt.Sprintf("%s_%s_%s_%s.tar.gz", binName, versionNoV, platformOS, platformArch)
	assetURL := metadata.assetURL(archiveName)
	if assetURL == "" {
		return nil, output.NotFound(
			"release_asset_not_found",
			fmt.Sprintf("release %s does not publish %s", targetTag, archiveName),
			"verify that the release has linux/darwin assets for this architecture",
		)
	}
	checksumsURL := metadata.assetURL(checksumAsset)
	if checksumsURL == "" {
		return nil, output.NotFound(
			"release_checksums_not_found",
			fmt.Sprintf("release %s does not publish %s", targetTag, checksumAsset),
			"publish the release workflow so the checksum asset is available",
		)
	}

	result := &UpdateResult{
		RequestedVersion: normalizeRequestedVersion(req.TargetVersion),
		CurrentVersion:   strings.TrimSpace(req.CurrentVersion),
		TargetVersion:    targetTag,
		ExecutablePath:   executablePath,
		OS:               platformOS,
		Arch:             platformArch,
		ReleaseURL:       metadata.HTMLURL,
		AssetName:        archiveName,
		DownloadURL:      assetURL,
		ChecksumURL:      checksumsURL,
		DryRun:           req.DryRun,
	}

	if normalizeTag(req.CurrentVersion) == targetTag && req.CurrentVersion != "" {
		result.AlreadyCurrent = true
		return result, nil
	}
	if req.DryRun {
		return result, nil
	}

	archiveBody, err := s.download(ctx, assetURL)
	if err != nil {
		return nil, err
	}
	checksumsBody, err := s.download(ctx, checksumsURL)
	if err != nil {
		return nil, err
	}
	if err := verifyChecksum(archiveName, archiveBody, checksumsBody); err != nil {
		return nil, err
	}
	result.ChecksumVerified = true

	binaryBody, err := extractBinary(archiveBody)
	if err != nil {
		return nil, err
	}
	if err := replaceExecutable(executablePath, binaryBody); err != nil {
		return nil, err
	}
	result.Updated = true
	return result, nil
}

func (s *Service) fetchReleaseMetadata(ctx context.Context, requestedVersion string) (*releaseMetadata, error) {
	path := "/releases/latest"
	if normalized := normalizeRequestedVersion(requestedVersion); normalized != "latest" {
		path = "/releases/tags/" + normalizeTag(normalized)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.releaseBaseURL+path, nil)
	if err != nil {
		return nil, output.Internal(err, "build release metadata request")
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "fakturownia-cli/dev")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, output.Network("release_metadata_failed", err.Error(), "verify network access to api.github.com and retry", true).WithCause(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, output.Internal(err, "read release metadata response")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, output.NotFound(
			"release_not_found",
			fmt.Sprintf("release %s was not found", normalizeRequestedVersion(requestedVersion)),
			"check the target version with `gh release list` or use --version latest",
		).WithRawBody(body)
	}
	if resp.StatusCode >= 400 {
		return nil, output.Remote(
			"release_metadata_rejected",
			fmt.Sprintf("release metadata request failed with status %d", resp.StatusCode),
			"retry later or inspect the GitHub release API response",
			resp.StatusCode >= 500,
		).WithRawBody(body)
	}

	var metadata releaseMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return nil, output.Internal(err, "decode release metadata response")
	}
	if metadata.TagName == "" {
		return nil, output.Internal(nil, "release metadata did not include a tag name")
	}
	return &metadata, nil
}

func (s *Service) download(ctx context.Context, assetURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL, nil)
	if err != nil {
		return nil, output.Internal(err, "build release asset request")
	}
	req.Header.Set("User-Agent", "fakturownia-cli/dev")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, output.Network("release_asset_failed", err.Error(), "verify network access to GitHub release assets and retry", true).WithCause(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, output.Internal(err, "read release asset response")
	}
	if resp.StatusCode >= 400 {
		return nil, output.Remote(
			"release_asset_rejected",
			fmt.Sprintf("release asset request failed with status %d", resp.StatusCode),
			"retry later or verify that the release asset exists",
			resp.StatusCode >= 500,
		).WithRawBody(body)
	}
	return body, nil
}

func detectPlatform() (string, string, error) {
	switch runtime.GOOS {
	case "linux", "darwin":
	default:
		return "", "", output.Usage(
			"unsupported_os",
			fmt.Sprintf("self update does not support %s", runtime.GOOS),
			"install a release archive manually from GitHub Releases",
		)
	}

	switch runtime.GOARCH {
	case "amd64", "arm64":
	default:
		return "", "", output.Usage(
			"unsupported_arch",
			fmt.Sprintf("self update does not support %s", runtime.GOARCH),
			"install a matching release archive manually from GitHub Releases",
		)
	}
	return runtime.GOOS, runtime.GOARCH, nil
}

func normalizeRequestedVersion(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "latest"
	}
	return trimmed
}

func normalizeTag(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.EqualFold(trimmed, "latest") {
		return trimmed
	}
	if strings.HasPrefix(trimmed, "v") {
		return trimmed
	}
	return "v" + trimmed
}

func (m *releaseMetadata) assetURL(name string) string {
	for _, asset := range m.Assets {
		if asset.Name == name {
			return asset.BrowserDownloadURL
		}
	}
	return ""
}

func verifyChecksum(archiveName string, archiveBody, checksumBody []byte) error {
	expected, err := expectedChecksum(string(checksumBody), archiveName)
	if err != nil {
		return err
	}
	actualDigest := sha256.Sum256(archiveBody)
	actual := hex.EncodeToString(actualDigest[:])
	if actual != expected {
		return output.Remote(
			"checksum_mismatch",
			fmt.Sprintf("downloaded archive checksum did not match %s", checksumAsset),
			"retry the update or reinstall from GitHub Releases",
			false,
		)
	}
	return nil
}

func expectedChecksum(contents, archiveName string) (string, error) {
	for _, line := range strings.Split(contents, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		if name == archiveName {
			return fields[0], nil
		}
	}
	return "", output.NotFound(
		"checksum_entry_not_found",
		fmt.Sprintf("%s did not include a checksum entry for %s", checksumAsset, archiveName),
		"verify the release assets and retry",
	)
}

func extractBinary(archiveBody []byte) ([]byte, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(archiveBody))
	if err != nil {
		return nil, output.Internal(err, "open release archive")
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil, output.NotFound("binary_missing", "release archive did not contain the fakturownia binary", "verify the selected release asset and retry")
		}
		if err != nil {
			return nil, output.Internal(err, "read release archive")
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(header.Name) != binName {
			continue
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			return nil, output.Internal(err, "extract binary from release archive")
		}
		if len(body) == 0 {
			return nil, output.Internal(nil, "release archive contained an empty fakturownia binary")
		}
		return body, nil
	}
}

func replaceExecutable(targetPath string, binaryBody []byte) error {
	dir := filepath.Dir(targetPath)
	currentMode := os.FileMode(0o755)
	if info, err := os.Stat(targetPath); err == nil {
		currentMode = info.Mode().Perm()
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return output.Internal(err, "inspect current executable")
	}

	tempFile, err := os.CreateTemp(dir, "."+filepath.Base(targetPath)+".tmp-*")
	if err != nil {
		if os.IsPermission(err) {
			return output.NewAppError(4, "permission", "permission_denied", fmt.Sprintf("cannot create a temporary file in %s", dir), false, "rerun from a writable install location or use elevated permissions to reinstall")
		}
		return output.Internal(err, "create temporary executable")
	}
	tempPath := tempFile.Name()
	success := false
	defer func() {
		_ = tempFile.Close()
		if !success {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := tempFile.Write(binaryBody); err != nil {
		return output.Internal(err, "write updated binary")
	}
	if err := tempFile.Chmod(currentMode); err != nil {
		return output.Internal(err, "set updated binary permissions")
	}
	if err := tempFile.Close(); err != nil {
		return output.Internal(err, "close updated binary")
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		if os.IsPermission(err) {
			return output.NewAppError(4, "permission", "permission_denied", fmt.Sprintf("cannot replace %s", targetPath), false, "rerun from a writable install location or use elevated permissions to reinstall")
		}
		return output.Internal(err, "replace current executable")
	}
	success = true
	return nil
}
