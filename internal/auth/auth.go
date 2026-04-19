package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/99designs/keyring"

	"github.com/sixers/fakturownia-cli/internal/config"
	"github.com/sixers/fakturownia-cli/internal/output"
	"github.com/sixers/fakturownia-cli/internal/transport"
)

var ErrSecretNotFound = config.ErrSecretNotFound

const (
	envKeyringBackend  = "FAKTUROWNIA_KEYRING_BACKEND"
	envKeyringPassword = "FAKTUROWNIA_KEYRING_PASSWORD"

	keyringBackendAuto     = "auto"
	keyringBackendFile     = "file"
	keyringBackendNative   = "native"
	keyringBackendKeychain = "keychain"
)

type Store interface {
	config.ProbeableTokenStore
	Set(name, value string) error
	Delete(name string) error
}

type KeyringStore struct {
	ring keyring.Keyring
}

type SecurityStore struct {
	service      string
	securityPath string
}

type MemoryStore struct {
	values map[string]string
}

type Service struct {
	store      Store
	now        func() time.Time
	httpClient *http.Client
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

type ExchangeRequest struct {
	ConfigPath       string
	Login            string
	Password         string
	IntegrationToken string
	SaveAs           string
	Timeout          time.Duration
	MaxRetries       int
}

type ExchangeResult struct {
	Login           string `json:"login,omitempty"`
	Email           string `json:"email,omitempty"`
	Prefix          string `json:"prefix,omitempty"`
	URL             string `json:"url,omitempty"`
	FirstName       string `json:"first_name,omitempty"`
	LastName        string `json:"last_name,omitempty"`
	APITokenPresent bool   `json:"api_token_present"`
	SavedProfile    string `json:"saved_profile,omitempty"`
	TokenStored     bool   `json:"token_stored"`
	ConfigPath      string `json:"config_path,omitempty"`
	RequestID       string `json:"request_id,omitempty"`
	RawBody         []byte `json:"-"`
}

func (r *ExchangeResult) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.SavedProfile
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

func NewKeyringStore() (Store, error) {
	backend, err := resolveKeyringBackendEnv(strings.TrimSpace(os.Getenv(envKeyringBackend)))
	if err != nil {
		return nil, err
	}
	password := strings.TrimSpace(os.Getenv(envKeyringPassword))

	switch backend {
	case keyringBackendFile:
		return openKeyringStore([]keyring.BackendType{keyring.FileBackend}, password)
	case keyringBackendNative:
		if runtime.GOOS == "darwin" {
			return newSecurityStore()
		}
		return openKeyringStore(nativeBackends(false), password)
	default:
		if runtime.GOOS == "darwin" {
			return newSecurityStore()
		}
		return openKeyringStore(nativeBackends(password != ""), password)
	}
}

func newSecurityStore() (Store, error) {
	securityPath, err := exec.LookPath("security")
	if err != nil {
		return nil, output.Internal(err, "locate security command")
	}
	return &SecurityStore{
		service:      config.ServiceName,
		securityPath: securityPath,
	}, nil
}

func openKeyringStore(allowed []keyring.BackendType, password string) (Store, error) {
	cfg := keyring.Config{
		ServiceName:     config.ServiceName,
		AllowedBackends: allowed,
	}
	if containsBackend(allowed, keyring.FileBackend) {
		fileDir, err := defaultFileKeyringDir()
		if err != nil {
			return nil, err
		}
		cfg.FileDir = fileDir
		cfg.FilePasswordFunc = fileKeyringPasswordFunc(password)
	}

	ring, err := keyring.Open(cfg)
	if err != nil {
		return nil, output.Internal(err, "open credential store")
	}
	return &KeyringStore{ring: ring}, nil
}

func nativeBackends(includeFile bool) []keyring.BackendType {
	backends := make([]keyring.BackendType, 0, len(keyring.AvailableBackends()))
	for _, backend := range keyring.AvailableBackends() {
		if backend == keyring.FileBackend && !includeFile {
			continue
		}
		backends = append(backends, backend)
	}
	return backends
}

func containsBackend(backends []keyring.BackendType, want keyring.BackendType) bool {
	for _, backend := range backends {
		if backend == want {
			return true
		}
	}
	return false
}

func resolveKeyringBackendEnv(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", keyringBackendAuto:
		return keyringBackendAuto, nil
	case keyringBackendFile:
		return keyringBackendFile, nil
	case keyringBackendNative, keyringBackendKeychain:
		return keyringBackendNative, nil
	default:
		return "", output.Usage(
			"invalid_keyring_backend",
			fmt.Sprintf("unsupported %s value %q", envKeyringBackend, raw),
			fmt.Sprintf("use %s=%s, %s, %s, or %s", envKeyringBackend, keyringBackendAuto, keyringBackendNative, keyringBackendFile, keyringBackendKeychain),
		)
	}
}

func defaultFileKeyringDir() (string, error) {
	configPath, err := config.ResolveConfigPath("")
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(configPath), "keyring"), nil
}

func fileKeyringPasswordFunc(password string) keyring.PromptFunc {
	return func(_ string) (string, error) {
		if strings.TrimSpace(password) == "" {
			return "", output.Usage(
				"missing_keyring_password",
				fmt.Sprintf("file credential store requires %s", envKeyringPassword),
				fmt.Sprintf("set %s or switch %s back to %s", envKeyringPassword, envKeyringBackend, keyringBackendAuto),
			)
		}
		return password, nil
	}
}

func (s *KeyringStore) Get(name string) (string, error) {
	item, err := s.ring.Get(name)
	if errors.Is(err, keyring.ErrKeyNotFound) {
		return "", ErrSecretNotFound
	}
	if err != nil {
		return "", wrapStoreError(err, "read token from credential store")
	}
	return string(item.Data), nil
}

func (s *KeyringStore) Set(name, value string) error {
	if err := s.ring.Set(keyring.Item{
		Key:  name,
		Data: []byte(value),
	}); err != nil {
		return wrapStoreError(err, "write token to credential store")
	}
	return nil
}

func (s *SecurityStore) Get(name string) (string, error) {
	stdout, stderr, err := s.run("find-generic-password", "-a", name, "-s", s.service, "-w")
	if err != nil {
		if securityItemNotFound(err, stderr) {
			return "", ErrSecretNotFound
		}
		return "", output.Internal(wrapSecurityError(err, stderr), "read token from keychain")
	}
	return strings.TrimRight(stdout, "\r\n"), nil
}

func (s *SecurityStore) Set(name, value string) error {
	_, stderr, err := s.run("add-generic-password", "-U", "-a", name, "-s", s.service, "-l", s.service, "-T", s.securityPath, "-w", value)
	if err != nil {
		return output.Internal(wrapSecurityError(err, stderr), "write token to keychain")
	}
	return nil
}

func (s *SecurityStore) Delete(name string) error {
	_, stderr, err := s.run("delete-generic-password", "-a", name, "-s", s.service)
	if err != nil && !securityItemNotFound(err, stderr) {
		return output.Internal(wrapSecurityError(err, stderr), "delete token from keychain")
	}
	return nil
}

func (s *SecurityStore) Probe() error {
	probeKey := fmt.Sprintf("doctor-probe-%d", time.Now().UnixNano())
	if err := s.Set(probeKey, "ok"); err != nil {
		return err
	}
	if _, err := s.Get(probeKey); err != nil {
		return err
	}
	return s.Delete(probeKey)
}

func (s *KeyringStore) Delete(name string) error {
	if err := s.ring.Remove(name); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return wrapStoreError(err, "delete token from credential store")
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

func (s *SecurityStore) run(args ...string) (string, string, error) {
	cmd := exec.Command(s.securityPath, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func securityItemNotFound(err error, stderr string) bool {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 44 {
		return true
	}
	return strings.Contains(stderr, "could not be found")
}

func wrapSecurityError(err error, stderr string) error {
	message := strings.TrimSpace(stderr)
	if message == "" {
		return err
	}
	return fmt.Errorf("%s: %w", message, err)
}

func wrapStoreError(err error, message string) error {
	var appErr *output.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return output.Internal(err, message)
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

func (s *Service) Exchange(ctx context.Context, req ExchangeRequest) (*ExchangeResult, error) {
	login := strings.TrimSpace(req.Login)
	if login == "" {
		return nil, output.Usage("missing_login", "login is required", "pass --login <login-or-email>")
	}
	password := strings.TrimSpace(req.Password)
	if password == "" {
		return nil, output.Usage("missing_password", "password is required", "pass --password <password>")
	}
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	client, err := transport.NewClient("https://app.fakturownia.pl", "", timeout, req.MaxRetries, s.httpClient)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"login":    login,
		"password": password,
	}
	if integrationToken := strings.TrimSpace(req.IntegrationToken); integrationToken != "" {
		payload["integration_token"] = integrationToken
	}

	var upstream struct {
		Login     string `json:"login"`
		Email     string `json:"email"`
		Prefix    string `json:"prefix"`
		URL       string `json:"url"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		APIToken  string `json:"api_token"`
	}
	resp, err := client.PostJSON(ctx, "/login.json", payload, &upstream)
	if err != nil {
		return nil, err
	}

	accountURL := strings.TrimSpace(upstream.URL)
	if accountURL == "" && strings.TrimSpace(upstream.Prefix) != "" {
		accountURL, err = config.NormalizePrefix(upstream.Prefix)
		if err != nil {
			return nil, err
		}
	}
	if accountURL == "" {
		return nil, output.Remote("missing_account_url", "login succeeded but the API did not return an account URL", "rerun with --raw to inspect the upstream response", false).WithRawBody(resp.RawBody)
	}
	accountURL, err = config.NormalizeURL(accountURL)
	if err != nil {
		return nil, err
	}

	result := &ExchangeResult{
		Login:           upstream.Login,
		Email:           upstream.Email,
		Prefix:          upstream.Prefix,
		URL:             accountURL,
		FirstName:       upstream.FirstName,
		LastName:        upstream.LastName,
		APITokenPresent: strings.TrimSpace(upstream.APIToken) != "",
		RequestID:       resp.RequestID,
		RawBody:         resp.RawBody,
	}
	if !result.APITokenPresent {
		return nil, output.AuthFailure("missing_api_token", "login succeeded but the user does not have an API token", "generate an API token in Fakturownia and retry `fakturownia auth exchange`").WithRawBody(resp.RawBody)
	}

	profileName := strings.TrimSpace(req.SaveAs)
	if profileName == "" {
		profileName = strings.TrimSpace(upstream.Prefix)
	}
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
	if cfg.DefaultProfile == "" {
		cfg.DefaultProfile = profileName
	}
	if err := config.Save(configPath, cfg); err != nil {
		return nil, err
	}
	if err := s.store.Set(profileName, strings.TrimSpace(upstream.APIToken)); err != nil {
		return nil, err
	}

	result.SavedProfile = profileName
	result.TokenStored = true
	result.ConfigPath = configPath
	return result, nil
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
