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
	"github.com/sixers/fakturownia-cli/internal/client"
	"github.com/sixers/fakturownia-cli/internal/doctor"
	"github.com/sixers/fakturownia-cli/internal/invoice"
	"github.com/sixers/fakturownia-cli/internal/output"
	"github.com/sixers/fakturownia-cli/internal/selfupdate"
	"github.com/sixers/fakturownia-cli/internal/transport"
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
			{
				"id":          1,
				"number":      "FV/1",
				"buyer_name":  "Acme",
				"price_gross": 100,
				"status":      "issued",
				"issue_date":  "2026-04-01",
				"positions": []any{
					map[string]any{"name": "Produkt A", "tax": "23"},
					map[string]any{"name": "Produkt B", "tax": "8"},
				},
			},
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
		Invoice: map[string]any{
			"id":     1,
			"number": "FV/1",
			"status": "issued",
			"positions": []any{
				map[string]any{"name": "Produkt A", "tax": "23"},
				map[string]any{"name": "Produkt B", "tax": "8"},
			},
		},
		RawBody:   []byte(`{"id":1,"number":"FV/1","status":"issued","positions":[{"name":"Produkt A","tax":"23"},{"name":"Produkt B","tax":"8"}]}`),
		Profile:   req.Profile,
		RequestID: "req-2",
	}, nil
}

func (f *fakeInvoiceService) Download(_ context.Context, req invoice.DownloadRequest) (*invoice.DownloadResponse, error) {
	f.downloadReq = req
	return &invoice.DownloadResponse{ID: req.ID, Path: filepath.Join(".", "invoice-"+req.ID+".pdf"), Bytes: 12, Profile: req.Profile}, nil
}

type fakeClientService struct {
	listReq   client.ListRequest
	getReq    client.GetRequest
	createReq client.CreateRequest
	updateReq client.UpdateRequest
	deleteReq client.DeleteRequest
}

func (f *fakeClientService) List(_ context.Context, req client.ListRequest) (*client.ListResponse, error) {
	f.listReq = req
	return &client.ListResponse{
		Clients: []map[string]any{
			{
				"id":       11,
				"name":     "Acme Sp. z o.o.",
				"tax_no":   "1234567890",
				"email":    "billing@acme.test",
				"city":     "Warsaw",
				"country":  "PL",
				"tag_list": []any{"vip", "b2b"},
			},
		},
		RawBody:    []byte(`[{"id":11,"name":"Acme Sp. z o.o."}]`),
		Profile:    req.Profile,
		RequestID:  "req-client-list",
		Pagination: output.Pagination{Page: req.Page, PerPage: req.PerPage, Returned: 1, HasNext: false},
	}, nil
}

func (f *fakeClientService) Get(_ context.Context, req client.GetRequest) (*client.GetResponse, error) {
	f.getReq = req
	value := req.ID
	if value == "" {
		value = req.ExternalID
	}
	return &client.GetResponse{
		Client: map[string]any{
			"id":          11,
			"name":        "Acme Sp. z o.o.",
			"email":       "billing@acme.test",
			"external_id": value,
			"tag_list":    []any{"vip", "b2b"},
		},
		RawBody:   []byte(`{"id":11,"name":"Acme Sp. z o.o.","email":"billing@acme.test"}`),
		Profile:   req.Profile,
		RequestID: "req-client-get",
	}, nil
}

func (f *fakeClientService) Create(_ context.Context, req client.CreateRequest) (*client.CreateResponse, error) {
	f.createReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("POST", "/clients.json", nil, map[string]any{"client": req.Input})
		return &client.CreateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &client.CreateResponse{
		Client: map[string]any{
			"id":    12,
			"name":  req.Input["name"],
			"email": req.Input["email"],
		},
		RawBody:   []byte(`{"id":12,"name":"New Client"}`),
		Profile:   req.Profile,
		RequestID: "req-client-create",
	}, nil
}

func (f *fakeClientService) Update(_ context.Context, req client.UpdateRequest) (*client.UpdateResponse, error) {
	f.updateReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("PUT", "/clients/"+req.ID+".json", nil, map[string]any{"client": req.Input})
		return &client.UpdateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &client.UpdateResponse{
		Client: map[string]any{
			"id":    req.ID,
			"email": req.Input["email"],
		},
		RawBody:   []byte(`{"id":12,"email":"updated@example.com"}`),
		Profile:   req.Profile,
		RequestID: "req-client-update",
	}, nil
}

func (f *fakeClientService) Delete(_ context.Context, req client.DeleteRequest) (*client.DeleteResponse, error) {
	f.deleteReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("DELETE", "/clients/"+req.ID+".json", nil, nil)
		return &client.DeleteResponse{ID: req.ID, Profile: req.Profile, DryRun: &plan}, nil
	}
	return &client.DeleteResponse{
		ID:        req.ID,
		Deleted:   true,
		Profile:   req.Profile,
		RequestID: "req-client-delete",
	}, nil
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

type fakeSelfUpdateService struct {
	updateReq selfupdate.UpdateRequest
}

func (f *fakeSelfUpdateService) Update(_ context.Context, req selfupdate.UpdateRequest) (*selfupdate.UpdateResult, error) {
	f.updateReq = req
	return &selfupdate.UpdateResult{
		RequestedVersion: normalizeRequested(req.TargetVersion),
		CurrentVersion:   req.CurrentVersion,
		TargetVersion:    "v9.9.9",
		ExecutablePath:   "/tmp/fakturownia",
		OS:               "darwin",
		Arch:             "arm64",
		ReleaseURL:       "https://github.com/sixers/fakturownia-cli/releases/tag/v9.9.9",
		AssetName:        "fakturownia_9.9.9_darwin_arm64.tar.gz",
		DownloadURL:      "https://example.test/fakturownia_9.9.9_darwin_arm64.tar.gz",
		ChecksumURL:      "https://example.test/checksums.txt",
		Updated:          !req.DryRun,
		DryRun:           req.DryRun,
		ChecksumVerified: !req.DryRun,
	}, nil
}

func normalizeRequested(value string) string {
	if strings.TrimSpace(value) == "" {
		return "latest"
	}
	return value
}

func TestCommandIntegration(t *testing.T) {
	authSvc := &fakeAuthService{}
	clientSvc := &fakeClientService{}
	invoiceSvc := &fakeInvoiceService{}
	doctorSvc := &fakeDoctorService{}
	selfSvc := &fakeSelfUpdateService{}

	runWithInput := func(input string, args ...string) (string, string, error) {
		var stdout, stderr bytes.Buffer
		cmd := NewRootCommand(Dependencies{
			Auth:    authSvc,
			Client:  clientSvc,
			Invoice: invoiceSvc,
			Doctor:  doctorSvc,
			Self:    selfSvc,
			Stdout:  &stdout,
			Stderr:  &stderr,
		})
		if input != "" {
			cmd.SetIn(strings.NewReader(input))
		}
		cmd.SetArgs(args)
		err := cmd.Execute()
		return stdout.String(), stderr.String(), err
	}
	run := func(args ...string) (string, string, error) {
		return runWithInput("", args...)
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

	stdout, _, err = run("client", "list", "--json")
	if err != nil {
		t.Fatalf("client list error = %v", err)
	}
	if !jsonContains(stdout, `"name": "Acme Sp. z o.o."`) {
		t.Fatalf("unexpected client list output: %s", stdout)
	}

	stdout, _, err = run("client", "get", "--external-id", "ext-123", "--json")
	if err != nil {
		t.Fatalf("client get by external-id error = %v", err)
	}
	if clientSvc.getReq.ExternalID != "ext-123" {
		t.Fatalf("expected client get to receive external ID, got %q", clientSvc.getReq.ExternalID)
	}
	if !jsonContains(stdout, `"external_id": "ext-123"`) {
		t.Fatalf("unexpected client get output: %s", stdout)
	}

	stdout, _, err = run("client", "create", "--input", `{"name":"New Client","email":"new@example.com"}`, "--json")
	if err != nil {
		t.Fatalf("client create error = %v", err)
	}
	if clientSvc.createReq.Input["name"] != "New Client" {
		t.Fatalf("expected client create input to be parsed, got %#v", clientSvc.createReq.Input)
	}
	if !jsonContains(stdout, `"id": 12`) {
		t.Fatalf("unexpected client create output: %s", stdout)
	}

	stdout, _, err = runWithInput(`{"email":"stdin@example.com"}`, "client", "update", "--id", "12", "--input", "-", "--json")
	if err != nil {
		t.Fatalf("client update error = %v", err)
	}
	if clientSvc.updateReq.Input["email"] != "stdin@example.com" {
		t.Fatalf("expected client update stdin input, got %#v", clientSvc.updateReq.Input)
	}
	if !jsonContains(stdout, `"email": "stdin@example.com"`) {
		t.Fatalf("unexpected client update output: %s", stdout)
	}

	stdout, _, err = run("client", "delete", "--id", "12", "--yes", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("client delete dry-run error = %v", err)
	}
	if !clientSvc.deleteReq.DryRun {
		t.Fatal("expected client delete dry-run flag to be forwarded")
	}
	if !jsonContains(stdout, `"method": "DELETE"`) || !jsonContains(stdout, `"[redacted]"`) {
		t.Fatalf("unexpected client delete dry-run output: %s", stdout)
	}

	stdout, _, err = run("invoice", "get", "--id", "1", "--fields", "id,number", "--json")
	if err != nil {
		t.Fatalf("invoice get error = %v", err)
	}
	if !jsonContains(stdout, `"number": "FV/1"`) || jsonContains(stdout, `"status": "issued"`) {
		t.Fatalf("unexpected invoice get projection output: %s", stdout)
	}

	stdout, _, err = run("invoice", "get", "--id", "1", "--fields", "number,positions[].name", "--json")
	if err != nil {
		t.Fatalf("invoice get nested projection error = %v", err)
	}
	if !jsonContains(stdout, `"positions": [`) || !jsonContains(stdout, `"name": "Produkt A"`) {
		t.Fatalf("unexpected nested projection output: %s", stdout)
	}

	stdout, stderr, err := run("invoice", "list", "--columns", "number,positions[].name")
	if err != nil {
		t.Fatalf("invoice list nested columns error = %v", err)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr for nested columns: %s", stderr)
	}
	if !strings.Contains(stdout, "Produkt A, Produkt B") {
		t.Fatalf("expected joined nested columns in output: %s", stdout)
	}

	stdout, _, err = run("invoice", "get", "--id", "1", "--fields", "number,custom_field", "--json")
	if err != nil {
		t.Fatalf("invoice get undocumented field warning error = %v", err)
	}
	if !jsonContains(stdout, `"code": "undocumented_field_path"`) {
		t.Fatalf("expected undocumented field warning in output: %s", stdout)
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

	stdout, _, err = run("self", "update", "--version", "v9.9.9", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("self update dry-run error = %v", err)
	}
	if !selfSvc.updateReq.DryRun || selfSvc.updateReq.TargetVersion != "v9.9.9" {
		t.Fatalf("expected self update dry-run request, got %#v", selfSvc.updateReq)
	}
	if !jsonContains(stdout, `"target_version": "v9.9.9"`) || !jsonContains(stdout, `"dry_run": true`) {
		t.Fatalf("unexpected self update output: %s", stdout)
	}
}

func TestGolden(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		args []string
		file string
	}{
		{name: "client-list-help", args: []string{"client", "list", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "client-list-help.txt")},
		{name: "schema-client-list-json", args: []string{"schema", "client", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-client-list.json")},
		{name: "schema-client-get-json", args: []string{"schema", "client", "get", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-client-get.json")},
		{name: "schema-client-create-json", args: []string{"schema", "client", "create", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-client-create.json")},
		{name: "schema-client-update-json", args: []string{"schema", "client", "update", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-client-update.json")},
		{name: "schema-client-delete-json", args: []string{"schema", "client", "delete", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-client-delete.json")},
		{name: "self-update-help", args: []string{"self", "update", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "self-update-help.txt")},
		{name: "schema-self-update-json", args: []string{"schema", "self", "update", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-self-update.json")},
		{name: "invoice-list-help", args: []string{"invoice", "list", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-list-help.txt")},
		{name: "schema-list-json", args: []string{"schema", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-list.json")},
		{name: "schema-invoice-list-json", args: []string{"schema", "invoice", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-list.json")},
		{name: "schema-invoice-get-json", args: []string{"schema", "invoice", "get", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-get.json")},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			cmd := NewRootCommand(Dependencies{
				Auth:    &fakeAuthService{},
				Client:  &fakeClientService{},
				Invoice: &fakeInvoiceService{},
				Doctor:  &fakeDoctorService{},
				Self:    &fakeSelfUpdateService{},
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
		Client:  &fakeClientService{},
		Invoice: &fakeInvoiceService{},
		Doctor:  &fakeDoctorService{},
		Self:    &fakeSelfUpdateService{},
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

func TestSchemaInvoiceListExposesKnownFields(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Auth:    &fakeAuthService{},
		Client:  &fakeClientService{},
		Invoice: &fakeInvoiceService{},
		Doctor:  &fakeDoctorService{},
		Self:    &fakeSelfUpdateService{},
		Stdout:  &stdout,
		Stderr:  &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"schema", "invoice", "list", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !jsonContains(stdout.String(), `"path": "positions[].name"`) {
		t.Fatalf("expected schema invoice list to advertise nested known fields: %s", stdout.String())
	}
}

func TestConfigFlagPassThrough(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	authSvc := &fakeAuthService{}
	cmd := NewRootCommand(Dependencies{
		Auth:    authSvc,
		Client:  &fakeClientService{},
		Invoice: &fakeInvoiceService{},
		Doctor:  &fakeDoctorService{},
		Self:    &fakeSelfUpdateService{},
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
