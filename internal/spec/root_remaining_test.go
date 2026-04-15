package spec

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/sixers/fakturownia-cli/internal/account"
	"github.com/sixers/fakturownia-cli/internal/auth"
	"github.com/sixers/fakturownia-cli/internal/department"
	"github.com/sixers/fakturownia-cli/internal/transport"
	"github.com/sixers/fakturownia-cli/internal/user"
	"github.com/sixers/fakturownia-cli/internal/webhook"
)

type fakeExchangeAuthService struct {
	exchangeReq auth.ExchangeRequest
}

func (f *fakeExchangeAuthService) Login(context.Context, auth.LoginRequest) (*auth.LoginResult, error) {
	return &auth.LoginResult{Profile: "work", URL: "https://acme.fakturownia.pl"}, nil
}

func (f *fakeExchangeAuthService) Exchange(_ context.Context, req auth.ExchangeRequest) (*auth.ExchangeResult, error) {
	f.exchangeReq = req
	return &auth.ExchangeResult{
		Login:           req.Login,
		Email:           req.Login,
		Prefix:          "acme",
		URL:             "https://acme.fakturownia.pl",
		APITokenPresent: true,
		SavedProfile:    "acme",
		TokenStored:     true,
		RawBody:         []byte(`{"login":"user@example.com","api_token":"secret-token"}`),
	}, nil
}

func (f *fakeExchangeAuthService) Status(context.Context, auth.StatusRequest) (*auth.StatusResult, error) {
	return &auth.StatusResult{Profile: "work", URL: "https://acme.fakturownia.pl", TokenPresent: true}, nil
}

func (f *fakeExchangeAuthService) Logout(context.Context, auth.LogoutRequest) (*auth.LogoutResult, error) {
	return &auth.LogoutResult{Profile: "work", Removed: true}, nil
}

type fakeAccountCommandService struct {
	createReq account.CreateRequest
	deleteReq account.DeleteRequest
}

func (f *fakeAccountCommandService) Create(_ context.Context, req account.CreateRequest) (*account.CreateResponse, error) {
	f.createReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("POST", "/account.json", nil, req.Input)
		return &account.CreateResponse{DryRun: &plan}, nil
	}
	return &account.CreateResponse{Prefix: "acme", URL: "https://acme.fakturownia.pl", APITokenPresent: true, SavedProfile: req.SaveAs, TokenStored: req.SaveAs != ""}, nil
}

func (f *fakeAccountCommandService) Get(context.Context, account.GetRequest) (*account.GetResponse, error) {
	return &account.GetResponse{Prefix: "acme", URL: "https://acme.fakturownia.pl", APITokenPresent: true}, nil
}

func (f *fakeAccountCommandService) Delete(_ context.Context, req account.DeleteRequest) (*account.DeleteResponse, error) {
	f.deleteReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("POST", "/account/delete.json", nil, map[string]any{"api_token": "[redacted]"})
		return &account.DeleteResponse{DryRun: &plan}, nil
	}
	return &account.DeleteResponse{Code: "ok", Message: "deleted"}, nil
}

func (f *fakeAccountCommandService) Unlink(context.Context, account.UnlinkRequest) (*account.UnlinkResponse, error) {
	return &account.UnlinkResponse{Code: "ok", Message: "unlinked"}, nil
}

type fakeDepartmentCommandService struct {
	setLogoReq department.SetLogoRequest
}

func (f *fakeDepartmentCommandService) List(context.Context, department.ListRequest) (*department.ListResponse, error) {
	return &department.ListResponse{Departments: []map[string]any{}}, nil
}

func (f *fakeDepartmentCommandService) Get(context.Context, department.GetRequest) (*department.GetResponse, error) {
	return &department.GetResponse{Department: map[string]any{"id": 10}}, nil
}

func (f *fakeDepartmentCommandService) Create(context.Context, department.CreateRequest) (*department.CreateResponse, error) {
	return &department.CreateResponse{Department: map[string]any{"id": 10}}, nil
}

func (f *fakeDepartmentCommandService) Update(context.Context, department.UpdateRequest) (*department.UpdateResponse, error) {
	return &department.UpdateResponse{Department: map[string]any{"id": 10}}, nil
}

func (f *fakeDepartmentCommandService) Delete(context.Context, department.DeleteRequest) (*department.DeleteResponse, error) {
	return &department.DeleteResponse{Deleted: true}, nil
}

func (f *fakeDepartmentCommandService) SetLogo(_ context.Context, req department.SetLogoRequest) (*department.SetLogoResponse, error) {
	f.setLogoReq = req
	if req.DryRun {
		plan := transport.PlanMultipartUpload(transport.MultipartUpload{
			Method:      "PUT",
			URL:         "https://acme.fakturownia.pl/departments/10.json",
			Fields:      map[string]string{"api_token": "secret"},
			FileField:   "department[logo]",
			FileName:    req.Name,
			FileContent: req.Content,
		})
		return &department.SetLogoResponse{ID: req.ID, Name: req.Name, DryRun: &plan}, nil
	}
	return &department.SetLogoResponse{ID: req.ID, Name: req.Name, Uploaded: true, Bytes: len(req.Content)}, nil
}

type fakeUserCommandService struct {
	createReq user.CreateRequest
}

func (f *fakeUserCommandService) Create(_ context.Context, req user.CreateRequest) (*user.CreateResponse, error) {
	f.createReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("POST", "/account/add_user.json", nil, map[string]any{"integration_token": req.IntegrationToken, "user": req.Input})
		return &user.CreateResponse{DryRun: &plan}, nil
	}
	return &user.CreateResponse{Response: map[string]any{"status": "ok"}}, nil
}

type fakeWebhookCommandService struct {
	createReq webhook.CreateRequest
}

func (f *fakeWebhookCommandService) List(context.Context, webhook.ListRequest) (*webhook.ListResponse, error) {
	return &webhook.ListResponse{Webhooks: []map[string]any{}}, nil
}

func (f *fakeWebhookCommandService) Get(context.Context, webhook.GetRequest) (*webhook.GetResponse, error) {
	return &webhook.GetResponse{Webhook: map[string]any{"id": 7}}, nil
}

func (f *fakeWebhookCommandService) Create(_ context.Context, req webhook.CreateRequest) (*webhook.CreateResponse, error) {
	f.createReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("POST", "/webhooks.json", nil, req.Input)
		return &webhook.CreateResponse{DryRun: &plan}, nil
	}
	return &webhook.CreateResponse{Webhook: map[string]any{"id": 7, "kind": req.Input["kind"], "url": req.Input["url"]}}, nil
}

func (f *fakeWebhookCommandService) Update(context.Context, webhook.UpdateRequest) (*webhook.UpdateResponse, error) {
	return &webhook.UpdateResponse{Webhook: map[string]any{"id": 7}}, nil
}

func (f *fakeWebhookCommandService) Delete(context.Context, webhook.DeleteRequest) (*webhook.DeleteResponse, error) {
	return &webhook.DeleteResponse{ID: "7", Deleted: true}, nil
}

func executeRoot(t *testing.T, deps Dependencies, input string, args ...string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	deps.Stdout = &stdout
	deps.Stderr = &stderr
	cmd := NewRootCommand(deps)
	if input != "" {
		cmd.SetIn(strings.NewReader(input))
	}
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func TestAuthExchangeCommandUsesStructuredSanitizedOutput(t *testing.T) {
	authSvc := &fakeExchangeAuthService{}

	stdout, _, err := executeRoot(t, Dependencies{Auth: authSvc}, "", "auth", "exchange", "--login", "user@example.com", "--password", "secret", "--integration-token", "partner", "--save-as", "work", "--json")
	if err != nil {
		t.Fatalf("auth exchange error = %v", err)
	}
	if authSvc.exchangeReq.Login != "user@example.com" || authSvc.exchangeReq.IntegrationToken != "partner" || authSvc.exchangeReq.SaveAs != "work" {
		t.Fatalf("unexpected forwarded request: %#v", authSvc.exchangeReq)
	}
	if !jsonContains(stdout, `"api_token_present": true`) {
		t.Fatalf("expected sanitized token metadata in output: %s", stdout)
	}
	if strings.Contains(stdout, "secret-token") {
		t.Fatalf("did not expect raw token in structured output: %s", stdout)
	}
}

func TestAccountCreateCommandAcceptsFullTopLevelInput(t *testing.T) {
	accountSvc := &fakeAccountCommandService{}

	stdout, _, err := executeRoot(t, Dependencies{Account: accountSvc}, `{"account":{"prefix":"acme"},"user":{"login":"owner"},"company":{"name":"Acme"}}`, "account", "create", "--input", "-", "--save-as", "work", "--json")
	if err != nil {
		t.Fatalf("account create error = %v", err)
	}
	if _, ok := accountSvc.createReq.Input["account"]; !ok {
		t.Fatalf("expected full top-level account object, got %#v", accountSvc.createReq.Input)
	}
	if !jsonContains(stdout, `"saved_profile": "work"`) {
		t.Fatalf("unexpected account create output: %s", stdout)
	}
}

func TestDepartmentSetLogoCommandReadsStdinAndRequiresName(t *testing.T) {
	departmentSvc := &fakeDepartmentCommandService{}

	_, stderr, err := executeRoot(t, Dependencies{Department: departmentSvc}, "png-bytes", "department", "set-logo", "--id", "10", "--file", "-")
	if err == nil {
		t.Fatal("expected missing name error")
	}
	if !strings.Contains(stderr, "--name is required") {
		t.Fatalf("expected missing name error, got stdout/stderr: %q", stderr)
	}

	stdout, _, err := executeRoot(t, Dependencies{Department: departmentSvc}, "png-bytes", "department", "set-logo", "--id", "10", "--file", "-", "--name", "logo.png", "--json")
	if err != nil {
		t.Fatalf("department set-logo error = %v", err)
	}
	if departmentSvc.setLogoReq.Name != "logo.png" || string(departmentSvc.setLogoReq.Content) != "png-bytes" {
		t.Fatalf("unexpected forwarded logo request: %#v", departmentSvc.setLogoReq)
	}
	if !jsonContains(stdout, `"uploaded": true`) {
		t.Fatalf("unexpected set-logo output: %s", stdout)
	}
}

func TestWebhookCreateCommandUsesFullTopLevelInput(t *testing.T) {
	webhookSvc := &fakeWebhookCommandService{}

	stdout, _, err := executeRoot(t, Dependencies{Webhook: webhookSvc}, `{"kind":"invoice:create","url":"https://example.com/hook","active":true}`, "webhook", "create", "--input", "-", "--json")
	if err != nil {
		t.Fatalf("webhook create error = %v", err)
	}
	if _, ok := webhookSvc.createReq.Input["kind"]; !ok {
		t.Fatalf("expected full top-level webhook input, got %#v", webhookSvc.createReq.Input)
	}
	if _, ok := webhookSvc.createReq.Input["webhook"]; ok {
		t.Fatalf("did not expect webhook wrapper in input %#v", webhookSvc.createReq.Input)
	}
	if !jsonContains(stdout, `"kind": "invoice:create"`) {
		t.Fatalf("unexpected webhook create output: %s", stdout)
	}
}

func TestUserCreateCommandForwardsIntegrationToken(t *testing.T) {
	userSvc := &fakeUserCommandService{}

	stdout, _, err := executeRoot(t, Dependencies{User: userSvc}, `{"invite":true,"email":"user@example.com","role":"member"}`, "user", "create", "--input", "-", "--integration-token", "partner", "--json")
	if err != nil {
		t.Fatalf("user create error = %v", err)
	}
	if userSvc.createReq.IntegrationToken != "partner" {
		t.Fatalf("expected integration token to be forwarded, got %#v", userSvc.createReq)
	}
	if !jsonContains(stdout, `"status": "ok"`) {
		t.Fatalf("unexpected user create output: %s", stdout)
	}
}

func TestSchemaRequestBodyModesForWrappedAndFullObjectCommands(t *testing.T) {
	accountSchema, _, err := executeRoot(t, Dependencies{}, "", "schema", "account", "create", "--json")
	if err != nil {
		t.Fatalf("schema account create error = %v", err)
	}
	if strings.Contains(accountSchema, `"wrapper_key":`) {
		t.Fatalf("did not expect wrapper_key for account create: %s", accountSchema)
	}
	if !jsonContains(accountSchema, `"path": "account.prefix"`) {
		t.Fatalf("expected full top-level account fields in schema: %s", accountSchema)
	}

	departmentSchema, _, err := executeRoot(t, Dependencies{}, "", "schema", "department", "create", "--json")
	if err != nil {
		t.Fatalf("schema department create error = %v", err)
	}
	if !jsonContains(departmentSchema, `"wrapper_key": "department"`) {
		t.Fatalf("expected department wrapper key in schema: %s", departmentSchema)
	}

	webhookSchema, _, err := executeRoot(t, Dependencies{}, "", "schema", "webhook", "create", "--json")
	if err != nil {
		t.Fatalf("schema webhook create error = %v", err)
	}
	if strings.Contains(webhookSchema, `"wrapper_key":`) {
		t.Fatalf("did not expect wrapper key for webhook create: %s", webhookSchema)
	}
	if !jsonContains(webhookSchema, `"path": "kind"`) {
		t.Fatalf("expected webhook request field in schema: %s", webhookSchema)
	}
}

func TestAccountDeleteRequiresYes(t *testing.T) {
	stdout, stderr, err := executeRoot(t, Dependencies{Account: &fakeAccountCommandService{}}, "", "account", "delete")
	if err == nil {
		t.Fatal("expected confirmation error")
	}
	text := stdout + stderr
	if !strings.Contains(text, "--yes is required for account delete") {
		t.Fatalf("expected confirmation message, got: %s", text)
	}
}
