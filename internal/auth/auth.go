package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/99designs/keyring"

	"github.com/sixers/fakturownia-cli/internal/config"
	"github.com/sixers/fakturownia-cli/internal/output"
)

var ErrSecretNotFound = errors.New("secret not found")

type Store interface {
	config.ProbeableTokenStore
	Set(name, value string) error
	Delete(name string) error
}

type KeyringStore struct {
	ring keyring.Keyring
}

type MemoryStore struct {
	values map[string]string
}

type Service struct {
	store Store
	now   func() time.Time
}

type LoginRequest struct {
	ConfigPath string
	Profile    string
	URL        string
	Prefix     string
	APIToken   string
	SetDefault bool
}

type LoginResult struct {
	Profile        string `json:"profile"`
	URL            string `json:"url"`
	DefaultProfile string `json:"default_profile"`
	TokenStored    bool   `json:"token_stored"`
	ConfigPath     string `json:"config_path"`
}

type StatusRequest struct {
	ConfigPath string
	Profile    string
	Env        config.Env
}

type StatusResult struct {
	Profile       string `json:"profile"`
	URL           string `json:"url"`
	ConfigPath    string `json:"config_path"`
	TokenPresent  bool   `json:"token_present"`
	ProfileSource string `json:"profile_source,omitempty"`
	URLSource     string `json:"url_source,omitempty"`
	TokenSource   string `json:"token_source,omitempty"`
	Default       bool   `json:"default"`
}

type LogoutRequest struct {
	ConfigPath string
	Profile    string
	Env        config.Env
}

type LogoutResult struct {
	Profile    string `json:"profile"`
	ConfigPath string `json:"config_path"`
	Removed    bool   `json:"removed"`
}

func NewKeyringStore() (*KeyringStore, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName: config.ServiceName,
	})
	if err != nil {
		return nil, output.Internal(err, "open keychain")
	}
	return &KeyringStore{ring: ring}, nil
}

func (s *KeyringStore) Get(name string) (string, error) {
	item, err := s.ring.Get(name)
	if errors.Is(err, keyring.ErrKeyNotFound) {
		return "", ErrSecretNotFound
	}
	if err != nil {
		return "", output.Internal(err, "read token from keychain")
	}
	return string(item.Data), nil
}

func (s *KeyringStore) Set(name, value string) error {
	if err := s.ring.Set(keyring.Item{
		Key:  name,
		Data: []byte(value),
	}); err != nil {
		return output.Internal(err, "write token to keychain")
	}
	return nil
}

func (s *KeyringStore) Delete(name string) error {
	if err := s.ring.Remove(name); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return output.Internal(err, "delete token from keychain")
	}
	return nil
}

func (s *KeyringStore) Probe() error {
	probeKey := fmt.Sprintf("doctor-probe-%d", time.Now().UnixNano())
	if err := s.Set(probeKey, "ok"); err != nil {
		return err
	}
	if _, err := s.Get(probeKey); err != nil {
		return err
	}
	return s.Delete(probeKey)
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{values: map[string]string{}}
}

func (s *MemoryStore) Get(name string) (string, error) {
	value, ok := s.values[name]
	if !ok {
		return "", ErrSecretNotFound
	}
	return value, nil
}

func (s *MemoryStore) Set(name, value string) error {
	s.values[name] = value
	return nil
}

func (s *MemoryStore) Delete(name string) error {
	delete(s.values, name)
	return nil
}

func (s *MemoryStore) Probe() error {
	return nil
}

func NewService(store Store) *Service {
	return &Service{
		store: store,
		now:   time.Now,
	}
}

func (s *Service) Login(_ context.Context, req LoginRequest) (*LoginResult, error) {
	accountURL, err := loginURL(req.URL, req.Prefix)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.APIToken) == "" {
		return nil, output.Usage("missing_api_token", "API token is required", "pass --api-token or set FAKTUROWNIA_API_TOKEN")
	}

	profileName := strings.TrimSpace(req.Profile)
	if profileName == "" {
		profileName = config.DeriveProfileName(accountURL)
	}
	if err := config.ValidateProfileName(profileName); err != nil {
		return nil, err
	}

	configPath, err := config.ResolveConfigPath(req.ConfigPath)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	config.UpsertProfile(cfg, profileName, accountURL, s.now())
	if req.SetDefault || cfg.DefaultProfile == "" {
		cfg.DefaultProfile = profileName
	}

	if err := config.Save(configPath, cfg); err != nil {
		return nil, err
	}
	if err := s.store.Set(profileName, strings.TrimSpace(req.APIToken)); err != nil {
		return nil, err
	}

	return &LoginResult{
		Profile:        profileName,
		URL:            accountURL,
		DefaultProfile: cfg.DefaultProfile,
		TokenStored:    true,
		ConfigPath:     configPath,
	}, nil
}

func (s *Service) Status(_ context.Context, req StatusRequest) (*StatusResult, error) {
	resolved, err := config.Resolve(req.ConfigPath, req.Env, req.Profile, s.store)
	if err != nil {
		return nil, err
	}
	return &StatusResult{
		Profile:       resolved.Name,
		URL:           resolved.URL,
		ConfigPath:    resolved.ConfigPath,
		TokenPresent:  resolved.Token != "",
		ProfileSource: resolved.ProfileSource,
		URLSource:     resolved.URLSource,
		TokenSource:   resolved.TokenSource,
		Default:       resolved.Default,
	}, nil
}

func (s *Service) Logout(_ context.Context, req LogoutRequest) (*LogoutResult, error) {
	configPath, err := config.ResolveConfigPath(req.ConfigPath)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	target := strings.TrimSpace(req.Profile)
	if target == "" {
		switch {
		case req.Env.Profile != "":
			target = req.Env.Profile
		case cfg.DefaultProfile != "":
			target = cfg.DefaultProfile
		}
	}
	if target == "" {
		return nil, output.Usage("missing_profile", "no profile is selected for logout", "pass --profile or set FAKTUROWNIA_PROFILE")
	}
	if _, ok := cfg.Profiles[target]; !ok {
		return nil, output.NotFound("profile_not_found", fmt.Sprintf("profile %q was not found", target), "use `fakturownia auth status` to inspect the current configuration")
	}

	config.RemoveProfile(cfg, target)
	if err := config.Save(configPath, cfg); err != nil {
		return nil, err
	}
	if err := s.store.Delete(target); err != nil {
		return nil, err
	}

	return &LogoutResult{
		Profile:    target,
		ConfigPath: configPath,
		Removed:    true,
	}, nil
}

func loginURL(rawURL, prefix string) (string, error) {
	switch {
	case strings.TrimSpace(rawURL) != "":
		return config.NormalizeURL(rawURL)
	case strings.TrimSpace(prefix) != "":
		return config.NormalizePrefix(prefix)
	default:
		return "", output.Usage("missing_account", "either --url or --prefix is required", "pass --prefix acme or --url https://acme.fakturownia.pl")
	}
}
