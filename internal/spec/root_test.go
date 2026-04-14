package spec

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sixers/fakturownia-cli/internal/auth"
	"github.com/sixers/fakturownia-cli/internal/doctor"
	"github.com/sixers/fakturownia-cli/internal/invoice"
	"github.com/sixers/fakturownia-cli/internal/output"
)

type fakeAuthService struct {
	loginReq  auth.LoginRequest
	statusReq auth.StatusRequest
	logoutReq auth.LogoutRequest
}

func (f *fakeAuthService) Login(_ context.Context, req auth.LoginRequest) (*auth.LoginResult, error) {
	f.loginReq = req
	return &auth.LoginResult{Profile: req.Profile, URL: "https://acme.fakturownia.pl", DefaultProfile: req.Profile, TokenStored: true}, nil
}

func (f *fakeAuthService) Status(_ context.Context, req auth.StatusRequest) (*auth.StatusResult, error) {
	f.statusReq = req
	return &auth.StatusResult{Profile: "work", URL: "https://acme.fakturownia.pl", TokenPresent: true}, nil
}

func (f *fakeAuthService) Logout(_ context.Context, req auth.LogoutRequest) (*auth.LogoutResult, error) {
	f.logoutReq = req
	return &auth.LogoutResult{Profile: req.Profile, Removed: true}, nil
}

type fakeInvoiceService struct {
	getReq      invoice.GetRequest
	downloadReq invoice.DownloadRequest
}

func (f *fakeInvoiceService) List(_ context.Context, req invoice.ListRequest) (*invoice.ListResponse, error) {
	return &invoice.ListResponse{
		Invoices: []map[string]any{
			{"id": 1, "number": "FV/1", "buyer_name": "Acme", "price_gross": 100, "status": "issued", "issue_date": "2026-04-01"},
		},
		RawBody:    []byte(`[{"id":1}]`),
		Profile:    req.Profile,
		RequestID:  "req-1",
		Pagination: output.Pagination{Page: 1, PerPage: 25, Returned: 1, HasNext: false},
	}, nil
}

func (f *fakeInvoiceService) Get(_ context.Context, req invoice.GetRequest) (*invoice.GetResponse, error) {
	f.getReq = req
	return &invoice.GetResponse{
		Invoice:   map[string]any{"id": 1, "number": "FV/1", "status": "issued"},
		RawBody:   []byte(`{"id":1,"number":"FV/1","status":"issued"}`),
		Profile:   req.Profile,
		RequestID: "req-2",
	}, nil
}

func (f *fakeInvoiceService) Download(_ context.Context, req invoice.DownloadRequest) (*invoice.DownloadResponse, error) {
	f.downloadReq = req
	return &invoice.DownloadResponse{ID: req.ID, Path: filepath.Join(".", "invoice-"+req.ID+".pdf"), Bytes: 12, Profile: req.Profile}, nil
}

type fakeDoctorService struct {
	runReq doctor.RunRequest
}

func (f *fakeDoctorService) Run(_ context.Context, req doctor.RunRequest) (*doctor.RunResult, error) {
	f.runReq = req
	return &doctor.RunResult{
		Profile: "work",
		Report: doctor.Report{
			Version: req.Version,
			Status:  "ok",
			Checks: []doctor.Check{
				{Name: "config-path", Status: "ok", Message: "using config path"},
			},
		},
	}, nil
}

func TestCommandIntegration(t *testing.T) {
	authSvc := &fakeAuthService{}
	invoiceSvc := &fakeInvoiceService{}
	doctorSvc := &fakeDoctorService{}

	run := func(args ...string) (string, string, error) {
		var stdout, stderr bytes.Buffer
		cmd := NewRootCommand(Dependencies{
			Auth:    authSvc,
			Invoice: invoiceSvc,
			Doctor:  doctorSvc,
			Stdout:  &stdout,
			Stderr:  &stderr,
		})
		cmd.SetArgs(args)
		err := cmd.Execute()
		return stdout.String(), stderr.String(), err
	}

	_, _, err := run("auth", "login", "--profile", "work", "--prefix", "acme", "--api-token", "token", "--json")
	if err != nil {
		t.Fatalf("auth login error = %v", err)
	}
	if authSvc.loginReq.Profile != "work" {
		t.Fatalf("expected login profile to come from --profile, got %q", authSvc.loginReq.Profile)
	}

	stdout, _, err := run("invoice", "list", "--json")
	if err != nil {
		t.Fatalf("invoice list error = %v", err)
	}
	if !jsonContains(stdout, `"status": "success"`) {
		t.Fatalf("unexpected invoice list output: %s", stdout)
	}

	stdout, _, err = run("invoice", "get", "--id", "1", "--fields", "id,number", "--json")
	if err != nil {
		t.Fatalf("invoice get error = %v", err)
	}
	if !jsonContains(stdout, `"number": "FV/1"`) || jsonContains(stdout, `"status": "issued"`) {
		t.Fatalf("unexpected invoice get projection output: %s", stdout)
	}

	stdout, _, err = run("invoice", "download", "--id", "1", "--json")
	if err != nil {
		t.Fatalf("invoice download error = %v", err)
	}
	if !jsonContains(stdout, `"path": "invoice-1.pdf"`) {
		t.Fatalf("unexpected invoice download output: %s", stdout)
	}

	stdout, _, err = run("doctor", "run", "--check-release-integrity", "--json")
	if err != nil {
		t.Fatalf("doctor run error = %v", err)
	}
	if !doctorSvc.runReq.CheckReleaseIntegrity {
		t.Fatal("expected --check-release-integrity to be forwarded")
	}
	if !jsonContains(stdout, `"status": "ok"`) {
		t.Fatalf("unexpected doctor output: %s", stdout)
	}
}

func TestGolden(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		args []string
		file string
	}{
		{name: "invoice-list-help", args: []string{"invoice", "list", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-list-help.txt")},
		{name: "schema-list-json", args: []string{"schema", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-list.json")},
		{name: "schema-invoice-list-json", args: []string{"schema", "invoice", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-list.json")},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			cmd := NewRootCommand(Dependencies{
				Auth:    &fakeAuthService{},
				Invoice: &fakeInvoiceService{},
				Doctor:  &fakeDoctorService{},
				Stdout:  &stdout,
				Stderr:  &stderr,
			})
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Execute() error = %v, stderr = %s", err, stderr.String())
			}

			got := stdout.String()
			if got == "" {
				got = stderr.String()
			}
			assertGolden(t, tc.file, got)
		})
	}
}

func assertGolden(t *testing.T, path, got string) {
	t.Helper()

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	normalizedWant := normalizeGoldenText(string(want))
	normalizedGot := normalizeGoldenText(got)
	if normalizedWant != normalizedGot {
		t.Fatalf("golden mismatch for %s\nwant:\n%s\ngot:\n%s", path, normalizedWant, normalizedGot)
	}
}

func jsonContains(body, needle string) bool {
	return bytes.Contains([]byte(body), []byte(needle))
}

func normalizeGoldenText(value string) string {
	return strings.ReplaceAll(value, "\r\n", "\n")
}

func TestSchemaListUsesRows(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Auth:    &fakeAuthService{},
		Invoice: &fakeInvoiceService{},
		Doctor:  &fakeDoctorService{},
		Stdout:  &stdout,
		Stderr:  &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"schema", "list", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var envelope struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(envelope.Data) == 0 {
		t.Fatal("expected schema list data")
	}
	if _, ok := envelope.Data[0]["noun"]; !ok {
		t.Fatalf("expected noun field in schema list row")
	}
}

func TestConfigFlagPassThrough(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	authSvc := &fakeAuthService{}
	cmd := NewRootCommand(Dependencies{
		Auth:    authSvc,
		Invoice: &fakeInvoiceService{},
		Doctor:  &fakeDoctorService{},
		Stdout:  &stdout,
		Stderr:  &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"auth", "status", "--config", filepath.Join(t.TempDir(), "config.json"), "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if authSvc.statusReq.ConfigPath == "" {
		t.Fatal("expected config path to be forwarded")
	}
}

func TestGlobalEnvContractDocumented(t *testing.T) {
	t.Parallel()

	spec, ok := FindCommand("invoice", "list")
	if !ok {
		t.Fatal("missing command spec")
	}
	names := make([]string, 0, len(spec.EnvVars))
	for _, env := range spec.EnvVars {
		names = append(names, env.Name)
	}
	expected := []string{"FAKTUROWNIA_PROFILE", "FAKTUROWNIA_URL", "FAKTUROWNIA_API_TOKEN"}
	for _, name := range expected {
		found := false
		for _, got := range names {
			if got == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected env var %s in spec", name)
		}
	}
}
