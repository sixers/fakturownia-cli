package config

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

type stubStore struct {
	values map[string]string
	err    error
}

func (s stubStore) Get(name string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.values[name], nil
}

func TestResolvePrecedence(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := &File{
		SchemaVersion:  "fakturownia-cli/v1alpha1",
		DefaultProfile: "default",
		Profiles: map[string]Profile{
			"default": {URL: "https://default.fakturownia.pl"},
			"other":   {URL: "https://other.fakturownia.pl"},
		},
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	resolved, err := Resolve(path, Env{
		Profile:  "other",
		URL:      "https://env.fakturownia.pl",
		APIToken: "env-token",
	}, "default", stubStore{values: map[string]string{
		"default": "default-token",
		"other":   "other-token",
	}})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if resolved.Name != "default" {
		t.Fatalf("expected flag-selected profile, got %q", resolved.Name)
	}
	if resolved.URL != "https://env.fakturownia.pl" {
		t.Fatalf("expected env URL override, got %q", resolved.URL)
	}
	if resolved.Token != "env-token" {
		t.Fatalf("expected env token override, got %q", resolved.Token)
	}
}

func TestNormalizePrefixAndURLValidation(t *testing.T) {
	t.Parallel()

	url, err := NormalizePrefix("acme")
	if err != nil {
		t.Fatalf("NormalizePrefix() error = %v", err)
	}
	if url != "https://acme.fakturownia.pl" {
		t.Fatalf("unexpected normalized prefix URL: %q", url)
	}

	if _, err := NormalizeURL("https://acme.fakturownia.pl/path"); err == nil {
		t.Fatal("expected path validation error")
	}
	if _, err := NormalizeURL("https://acme.fakturownia.pl?q=1"); err == nil {
		t.Fatal("expected query validation error")
	}
}

func TestUpsertProfile(t *testing.T) {
	t.Parallel()

	cfg := &File{Profiles: map[string]Profile{}}
	now := time.Unix(1700000000, 0).UTC()
	UpsertProfile(cfg, "work", "https://work.fakturownia.pl", now)
	if cfg.Profiles["work"].URL != "https://work.fakturownia.pl" {
		t.Fatalf("profile URL was not saved")
	}
}

func TestResolvePropagatesStoreErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := &File{
		SchemaVersion:  "fakturownia-cli/v1alpha1",
		DefaultProfile: "default",
		Profiles: map[string]Profile{
			"default": {URL: "https://default.fakturownia.pl"},
		},
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	expected := errors.New("read token from keychain")
	_, err := Resolve(path, Env{}, "", stubStore{err: expected})
	if !errors.Is(err, expected) {
		t.Fatalf("Resolve() error = %v, want wrapped %v", err, expected)
	}
}
