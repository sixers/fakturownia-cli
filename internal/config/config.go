package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/sixers/fakturownia-cli/internal/output"
)

const (
	ServiceName       = "fakturownia-cli"
	DefaultConfigName = "config.json"
)

var (
	profileNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	prefixPattern      = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
)

type File struct {
	SchemaVersion  string             `json:"schema_version"`
	DefaultProfile string             `json:"default_profile,omitempty"`
	Profiles       map[string]Profile `json:"profiles"`
}

type Profile struct {
	URL       string `json:"url"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type Env struct {
	Profile  string
	URL      string
	APIToken string
}

type TokenStore interface {
	Get(name string) (string, error)
}

type ProbeableTokenStore interface {
	TokenStore
	Probe() error
}

type ResolvedProfile struct {
	Name          string `json:"name"`
	URL           string `json:"url"`
	Token         string `json:"-"`
	ConfigPath    string `json:"config_path"`
	ProfileSource string `json:"profile_source"`
	URLSource     string `json:"url_source"`
	TokenSource   string `json:"token_source"`
	Default       bool   `json:"default"`
}

func LookupEnv() Env {
	return Env{
		Profile:  strings.TrimSpace(os.Getenv("FAKTUROWNIA_PROFILE")),
		URL:      strings.TrimSpace(os.Getenv("FAKTUROWNIA_URL")),
		APIToken: strings.TrimSpace(os.Getenv("FAKTUROWNIA_API_TOKEN")),
	}
}

func ResolveConfigPath(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	baseDir, err := os.UserConfigDir()
	if err != nil {
		return "", output.Internal(err, "resolve config directory")
	}
	return filepath.Join(baseDir, "fakturownia", DefaultConfigName), nil
}

func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &File{
				SchemaVersion: output.SchemaVersion,
				Profiles:      map[string]Profile{},
			}, nil
		}
		return nil, output.Internal(err, "read config file")
	}
	var cfg File
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, output.Internal(err, "parse config file")
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	if cfg.SchemaVersion == "" {
		cfg.SchemaVersion = output.SchemaVersion
	}
	return &cfg, nil
}

func Save(path string, cfg *File) error {
	if cfg.SchemaVersion == "" {
		cfg.SchemaVersion = output.SchemaVersion
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return output.Internal(err, "create config directory")
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return output.Internal(err, "serialize config file")
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return output.Internal(err, "write config file")
	}
	return nil
}

func Resolve(path string, env Env, profileFlag string, store TokenStore) (*ResolvedProfile, error) {
	configPath, err := ResolveConfigPath(path)
	if err != nil {
		return nil, err
	}
	cfg, err := Load(configPath)
	if err != nil {
		return nil, err
	}

	selectedName := ""
	profileSource := ""
	switch {
	case strings.TrimSpace(profileFlag) != "":
		selectedName = strings.TrimSpace(profileFlag)
		profileSource = "flag"
	case env.Profile != "":
		selectedName = env.Profile
		profileSource = "env"
	case cfg.DefaultProfile != "":
		selectedName = cfg.DefaultProfile
		profileSource = "config"
	}

	var profile Profile
	if selectedName != "" {
		found, ok := cfg.Profiles[selectedName]
		if !ok {
			return nil, output.Usage("profile_not_found", fmt.Sprintf("profile %q was not found", selectedName), "use `fakturownia auth login` or set FAKTUROWNIA_PROFILE to an existing profile")
		}
		profile = found
	}

	resolved := &ResolvedProfile{
		Name:          selectedName,
		ConfigPath:    configPath,
		ProfileSource: profileSource,
		Default:       cfg.DefaultProfile != "" && selectedName == cfg.DefaultProfile,
	}

	if env.URL != "" {
		resolved.URL = env.URL
		resolved.URLSource = "env"
	} else if profile.URL != "" {
		resolved.URL = profile.URL
		resolved.URLSource = "profile"
	}
	if resolved.URL != "" {
		normalized, err := NormalizeURL(resolved.URL)
		if err != nil {
			return nil, err
		}
		resolved.URL = normalized
	}

	if env.APIToken != "" {
		resolved.Token = env.APIToken
		resolved.TokenSource = "env"
	} else if selectedName != "" && store != nil {
		token, tokenErr := store.Get(selectedName)
		if tokenErr == nil {
			resolved.Token = token
			resolved.TokenSource = "keychain"
		}
	}

	if resolved.Name == "" && (resolved.URL != "" || resolved.Token != "") {
		resolved.Name = "env"
		resolved.ProfileSource = "env"
	}

	if resolved.URL == "" {
		return nil, output.AuthFailure("missing_url", "no Fakturownia account URL is configured", "set FAKTUROWNIA_URL or run `fakturownia auth login --prefix <account> --api-token <token>`")
	}
	if resolved.Token == "" {
		return nil, output.AuthFailure("missing_api_token", "no Fakturownia API token is configured", "set FAKTUROWNIA_API_TOKEN or run `fakturownia auth login --prefix <account> --api-token <token>`")
	}

	return resolved, nil
}

func ValidateProfileName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return output.Usage("invalid_profile", "profile name cannot be empty", "use letters, numbers, dots, underscores, or hyphens")
	}
	if !profileNamePattern.MatchString(name) {
		return output.Usage("invalid_profile", fmt.Sprintf("profile name %q is invalid", name), "use letters, numbers, dots, underscores, or hyphens")
	}
	return nil
}

func DeriveProfileName(accountURL string) string {
	parsed, err := url.Parse(accountURL)
	if err != nil {
		return "default"
	}
	host := parsed.Hostname()
	if host == "" {
		return "default"
	}
	if strings.HasSuffix(host, ".fakturownia.pl") {
		prefix := strings.TrimSuffix(host, ".fakturownia.pl")
		if prefix != "" {
			return prefix
		}
	}
	host = strings.ReplaceAll(host, ".", "-")
	if host == "" {
		return "default"
	}
	return host
}

func NormalizeURL(rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", output.Usage("missing_url", "account URL is required", "pass --url, --prefix, or set FAKTUROWNIA_URL")
	}
	if hasControlChars(trimmed) {
		return "", output.Usage("invalid_url", "account URL contains control characters", "pass a clean HTTPS URL like https://acme.fakturownia.pl")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", output.Usage("invalid_url", fmt.Sprintf("account URL %q is invalid", rawURL), "pass a clean HTTPS URL like https://acme.fakturownia.pl")
	}
	if parsed.Scheme != "https" {
		return "", output.Usage("invalid_url_scheme", "account URL must use https", "pass a URL like https://acme.fakturownia.pl")
	}
	if parsed.Host == "" {
		return "", output.Usage("invalid_url_host", "account URL must include a host", "pass a URL like https://acme.fakturownia.pl")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", output.Usage("invalid_url_query", "account URL must not include query strings or fragments", "pass only the base account URL")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", output.Usage("invalid_url_path", "account URL must not include a path", "pass only the base account URL")
	}
	parsed.Path = ""
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func NormalizePrefix(prefix string) (string, error) {
	trimmed := strings.TrimSpace(prefix)
	if trimmed == "" {
		return "", output.Usage("missing_prefix", "account prefix is required", "pass a prefix like `acme`")
	}
	if hasControlChars(trimmed) {
		return "", output.Usage("invalid_prefix", "account prefix contains control characters", "pass a clean account prefix like `acme`")
	}
	if !prefixPattern.MatchString(trimmed) {
		return "", output.Usage("invalid_prefix", fmt.Sprintf("account prefix %q is invalid", prefix), "use letters, numbers, and hyphens only")
	}
	return fmt.Sprintf("https://%s.fakturownia.pl", trimmed), nil
}

func UpsertProfile(cfg *File, name, accountURL string, now time.Time) {
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	profile := cfg.Profiles[name]
	if profile.CreatedAt == "" {
		profile.CreatedAt = now.UTC().Format(time.RFC3339)
	}
	profile.URL = accountURL
	profile.UpdatedAt = now.UTC().Format(time.RFC3339)
	cfg.Profiles[name] = profile
}

func RemoveProfile(cfg *File, name string) {
	delete(cfg.Profiles, name)
	if cfg.DefaultProfile == name {
		cfg.DefaultProfile = ""
	}
}

func hasControlChars(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}
