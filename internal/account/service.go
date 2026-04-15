package account

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sixers/fakturownia-cli/internal/config"
	"github.com/sixers/fakturownia-cli/internal/output"
	"github.com/sixers/fakturownia-cli/internal/transport"
)

type Store interface {
	config.TokenStore
	Set(name, value string) error
}

type Service struct {
	store Store
	now   func() time.Time
}

type CreateRequest struct {
	ConfigPath string
	Profile    string
	Env        config.Env
	Timeout    time.Duration
	MaxRetries int
	Input      map[string]any
	SaveAs     string
	DryRun     bool
}

type CreateResponse struct {
	Prefix          string                 `json:"prefix,omitempty"`
	URL             string                 `json:"url,omitempty"`
	Login           string                 `json:"login,omitempty"`
	Email           string                 `json:"email,omitempty"`
	APITokenPresent bool                   `json:"api_token_present"`
	SavedProfile    string                 `json:"saved_profile,omitempty"`
	TokenStored     bool                   `json:"token_stored"`
	ConfigPath      string                 `json:"config_path,omitempty"`
	Profile         string                 `json:"-"`
	RequestID       string                 `json:"request_id,omitempty"`
	DryRun          *transport.RequestPlan `json:"dry_run,omitempty"`
	RawBody         []byte                 `json:"-"`
}

func (r *CreateResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

type GetRequest struct {
	ConfigPath       string
	Profile          string
	Env              config.Env
	Timeout          time.Duration
	MaxRetries       int
	IntegrationToken string
}

type GetResponse struct {
	Prefix          string `json:"prefix,omitempty"`
	URL             string `json:"url,omitempty"`
	Login           string `json:"login,omitempty"`
	Email           string `json:"email,omitempty"`
	APITokenPresent bool   `json:"api_token_present"`
	Profile         string `json:"-"`
	RequestID       string `json:"request_id,omitempty"`
	RawBody         []byte `json:"-"`
}

func (r *GetResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

type DeleteRequest struct {
	ConfigPath string
	Profile    string
	Env        config.Env
	Timeout    time.Duration
	MaxRetries int
	DryRun     bool
}

type DeleteResponse struct {
	Code      string                 `json:"code,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Profile   string                 `json:"profile"`
	RequestID string                 `json:"request_id,omitempty"`
	DryRun    *transport.RequestPlan `json:"dry_run,omitempty"`
	RawBody   []byte                 `json:"-"`
}

func (r *DeleteResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

type UnlinkRequest struct {
	ConfigPath       string
	Profile          string
	Env              config.Env
	Timeout          time.Duration
	MaxRetries       int
	Prefixes         []string
	IntegrationToken string
	DryRun           bool
}

type UnlinkResponse struct {
	Code      string                 `json:"code,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Result    Result                 `json:"result,omitempty"`
	Profile   string                 `json:"profile"`
	RequestID string                 `json:"request_id,omitempty"`
	DryRun    *transport.RequestPlan `json:"dry_run,omitempty"`
	RawBody   []byte                 `json:"-"`
}

type Result struct {
	Unlinked    []string `json:"unlinked,omitempty"`
	NotUnlinked []string `json:"not_unlinked,omitempty"`
}

func (r *UnlinkResponse) GetProfile() string {
	if r == nil {
		return ""
	}
	return r.Profile
}

func NewService(store Store) *Service {
	return &Service{store: store, now: time.Now}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*CreateResponse, error) {
	if req.Input == nil {
		return nil, output.Usage("missing_input", "account input is required", "pass --input -|@file.json|'{...}'")
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	payload := cloneMap(req.Input)
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodPost, "/account.json", nil, payload)
		return &CreateResponse{Profile: resolved.Name, DryRun: &plan}, nil
	}

	var upstream struct {
		Prefix   string `json:"prefix"`
		APIToken string `json:"api_token"`
		URL      string `json:"url"`
		Login    string `json:"login"`
		Email    string `json:"email"`
	}
	resp, err := client.PostJSON(ctx, "/account.json", payload, &upstream)
	if err != nil {
		return nil, err
	}

	accountURL, err := normalizeReturnedURL(upstream.URL, upstream.Prefix)
	if err != nil {
		return nil, err
	}

	result := &CreateResponse{
		Prefix:          upstream.Prefix,
		URL:             accountURL,
		Login:           upstream.Login,
		Email:           upstream.Email,
		APITokenPresent: strings.TrimSpace(upstream.APIToken) != "",
		Profile:         resolved.Name,
		RequestID:       resp.RequestID,
		RawBody:         resp.RawBody,
	}

	if saveAs := strings.TrimSpace(req.SaveAs); saveAs != "" {
		if !result.APITokenPresent {
			return nil, output.AuthFailure("missing_api_token", "account was created but no API token was returned", "rerun with --raw to inspect the upstream response or omit --save-as").WithRawBody(resp.RawBody)
		}
		if err := config.ValidateProfileName(saveAs); err != nil {
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
		config.UpsertProfile(cfg, saveAs, accountURL, s.now())
		if err := config.Save(configPath, cfg); err != nil {
			return nil, err
		}
		if err := s.store.Set(saveAs, strings.TrimSpace(upstream.APIToken)); err != nil {
			return nil, err
		}
		result.SavedProfile = saveAs
		result.TokenStored = true
		result.ConfigPath = configPath
	}

	return result, nil
}

func (s *Service) Get(ctx context.Context, req GetRequest) (*GetResponse, error) {
	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	if token := strings.TrimSpace(req.IntegrationToken); token != "" {
		query.Set("integration_token", token)
	}

	var upstream struct {
		Prefix   string `json:"prefix"`
		APIToken string `json:"api_token"`
		URL      string `json:"url"`
		Login    string `json:"login"`
		Email    string `json:"email"`
	}
	resp, err := client.GetJSON(ctx, "/account.json", query, &upstream)
	if err != nil {
		return nil, err
	}

	accountURL, err := normalizeReturnedURL(upstream.URL, upstream.Prefix)
	if err != nil {
		return nil, err
	}
	return &GetResponse{
		Prefix:          upstream.Prefix,
		URL:             accountURL,
		Login:           upstream.Login,
		Email:           upstream.Email,
		APITokenPresent: strings.TrimSpace(upstream.APIToken) != "",
		Profile:         resolved.Name,
		RequestID:       resp.RequestID,
		RawBody:         resp.RawBody,
	}, nil
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (*DeleteResponse, error) {
	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{}
	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodPost, "/account/delete.json", nil, payload)
		return &DeleteResponse{Profile: resolved.Name, DryRun: &plan}, nil
	}

	var response DeleteResponse
	resp, err := client.PostJSON(ctx, "/account/delete.json", payload, &response)
	if err != nil {
		return nil, err
	}
	response.Profile = resolved.Name
	response.RequestID = resp.RequestID
	response.RawBody = resp.RawBody
	return &response, nil
}

func (s *Service) Unlink(ctx context.Context, req UnlinkRequest) (*UnlinkResponse, error) {
	prefixes := trimNonEmpty(req.Prefixes)
	if len(prefixes) == 0 {
		return nil, output.Usage("missing_prefix", "at least one --prefix is required", "pass one or more --prefix values")
	}

	resolved, client, err := s.resolveClient(req.ConfigPath, req.Profile, req.Env, req.Timeout, req.MaxRetries)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{}
	if len(prefixes) == 1 {
		payload["prefix"] = prefixes[0]
	} else {
		values := make([]any, 0, len(prefixes))
		for _, prefix := range prefixes {
			values = append(values, prefix)
		}
		payload["prefix"] = values
	}
	if token := strings.TrimSpace(req.IntegrationToken); token != "" {
		payload["integration_token"] = token
	}

	if req.DryRun {
		plan := transport.PlanJSONRequest(http.MethodPatch, "/account/unlink.json", nil, payload)
		return &UnlinkResponse{Profile: resolved.Name, DryRun: &plan}, nil
	}

	var response UnlinkResponse
	resp, err := client.PatchJSON(ctx, "/account/unlink.json", payload, &response)
	if err != nil {
		return nil, err
	}
	response.Profile = resolved.Name
	response.RequestID = resp.RequestID
	response.RawBody = resp.RawBody
	return &response, nil
}

func (s *Service) resolveClient(configPath, profile string, env config.Env, timeout time.Duration, maxRetries int) (*config.ResolvedProfile, *transport.Client, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	resolved, err := config.Resolve(configPath, env, profile, s.store)
	if err != nil {
		return nil, nil, err
	}
	client, err := transport.NewClient(resolved.URL, resolved.Token, timeout, maxRetries, nil)
	if err != nil {
		return nil, nil, err
	}
	return resolved, client, nil
}

func normalizeReturnedURL(rawURL, prefix string) (string, error) {
	if strings.TrimSpace(rawURL) != "" {
		return config.NormalizeURL(rawURL)
	}
	if strings.TrimSpace(prefix) != "" {
		return config.NormalizePrefix(prefix)
	}
	return "", output.Remote("missing_account_url", "the API did not return an account URL", "rerun with --raw to inspect the upstream response", false)
}

func trimNonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
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
