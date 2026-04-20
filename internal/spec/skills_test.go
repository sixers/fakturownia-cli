package spec

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillBundleAssignmentsAreComplete(t *testing.T) {
	t.Parallel()

	bundle := SkillBundle()
	if err := validateSkillBundle(bundle); err != nil {
		t.Fatalf("validateSkillBundle() error = %v", err)
	}

	assignments, err := skillCommandAssignments(bundle)
	if err != nil {
		t.Fatalf("skillCommandAssignments() error = %v", err)
	}
	if len(assignments) != len(Registry()) {
		t.Fatalf("expected %d assigned commands, got %d", len(Registry()), len(assignments))
	}
}

func TestGeneratedSkillFilesMatchGolden(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	cases := []string{
		bundleRootSkillPath(),
		bundleIndexPath(),
		skillAreaPath(sharedSkillArea(SkillBundle())),
		filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "auth", "SKILL.md")),
		filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "accounts", "SKILL.md")),
		filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "departments", "SKILL.md")),
		filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "issuers", "SKILL.md")),
		filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "users", "SKILL.md")),
		filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "webhooks", "SKILL.md")),
		filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "categories", "SKILL.md")),
		filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "clients", "SKILL.md")),
		filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "payments", "SKILL.md")),
		filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "bank-accounts", "SKILL.md")),
		filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "products", "SKILL.md")),
		filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "price-lists", "SKILL.md")),
		filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "invoices", "SKILL.md")),
		filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "recurrings", "SKILL.md")),
		filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "warehouses", "SKILL.md")),
		filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "warehouse-actions", "SKILL.md")),
		filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "warehouse-documents", "SKILL.md")),
		recipeIndexPath(),
		recipePath(Recipes()[0]),
		docsSkillsPath(),
	}

	for _, path := range cases {
		got, ok := byPath[path]
		if !ok {
			t.Fatalf("missing generated file %s", path)
		}
		assertGolden(t, filepath.Join("..", "..", filepath.FromSlash(path)), got)
	}
}

func TestGeneratedSkillLinksResolve(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	for _, file := range files {
		for _, link := range extractMarkdownLinks(file.Content) {
			if link == "" || strings.HasPrefix(link, "#") || strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
				continue
			}
			resolved := resolveRelativeLink(file.Path, link)
			if _, ok := byPath[resolved]; !ok {
				t.Fatalf("link %q from %s resolved to missing file %s", link, file.Path, resolved)
			}
		}
	}
}

func TestRootSkillReferencesIndexAndShared(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	root := byPath[bundleRootSkillPath()]
	if !strings.Contains(root, "references/skills-index.md") {
		t.Fatalf("expected root skill to reference bundle index: %s", root)
	}
	if !strings.Contains(root, "subskills/shared/SKILL.md") {
		t.Fatalf("expected root skill to reference shared skill: %s", root)
	}
	if !strings.Contains(root, "recipes/index.md") {
		t.Fatalf("expected root skill to reference recipes index: %s", root)
	}
}

func TestRootSkillIncludesReadinessGuidance(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	root := byPath[bundleRootSkillPath()]
	want := []string{
		"## Before You Use It",
		"fakturownia --version",
		"fakturownia auth login --prefix acme --api-token \"$FAKTUROWNIA_API_TOKEN\"",
		"fakturownia auth status --json",
		"fakturownia account get --json",
	}
	for _, needle := range want {
		if !strings.Contains(root, needle) {
			t.Fatalf("expected root skill to include %q: %s", needle, root)
		}
	}
	if strings.Contains(root, "curl -fsSL https://raw.githubusercontent.com/sixers/fakturownia-cli/master/install.sh | bash") {
		t.Fatalf("expected root skill to omit installer guidance: %s", root)
	}
}

func TestInvoicesSkillIncludesOutputDiscovery(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	invoices := byPath[filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "invoices", "SKILL.md"))]
	want := []string{
		"fakturownia schema invoice list --json",
		"fakturownia schema invoice get --json",
		"output.known_fields",
		"positions[].name",
		"send-gov",
		"gov_status",
	}
	for _, needle := range want {
		if !strings.Contains(invoices, needle) {
			t.Fatalf("expected invoices skill to include %q: %s", needle, invoices)
		}
	}
}

func TestAuthSkillIncludesExchangeGuidance(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	authSkill := byPath[filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "auth", "SKILL.md"))]
	want := []string{
		"auth exchange",
		"api_token_present",
		"`--raw`",
	}
	for _, needle := range want {
		if !strings.Contains(authSkill, needle) {
			t.Fatalf("expected auth skill to include %q: %s", needle, authSkill)
		}
	}
}

func TestSharedSkillIncludesReadinessGuidance(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	shared := byPath[skillAreaPath(sharedSkillArea(SkillBundle()))]
	want := []string{
		"## Verify, Authenticate, And Smoke Test",
		"fakturownia --version",
		"fakturownia auth login --prefix acme --api-token \"$FAKTUROWNIA_API_TOKEN\"",
		"fakturownia auth status --json",
		"fakturownia account get --json",
	}
	for _, needle := range want {
		if !strings.Contains(shared, needle) {
			t.Fatalf("expected shared skill to include %q: %s", needle, shared)
		}
	}
	if strings.Contains(shared, "curl -fsSL https://raw.githubusercontent.com/sixers/fakturownia-cli/master/install.sh | bash") {
		t.Fatalf("expected shared skill to omit installer guidance: %s", shared)
	}
}

func TestAccountsSkillIncludesFullObjectDiscovery(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	accounts := byPath[filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "accounts", "SKILL.md"))]
	want := []string{
		"schema account create --json",
		"full top-level account request object",
		"`--raw`",
	}
	for _, needle := range want {
		if !strings.Contains(accounts, needle) {
			t.Fatalf("expected accounts skill to include %q: %s", needle, accounts)
		}
	}
}

func TestClientsSkillIncludesRequestDiscovery(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	clients := byPath[filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "clients", "SKILL.md"))]
	want := []string{
		"fakturownia schema client list --json",
		"fakturownia schema client create --json",
		"request_body_schema",
		"`--input` accepts inline JSON, `@file`, or `-` for stdin",
	}
	for _, needle := range want {
		if !strings.Contains(clients, needle) {
			t.Fatalf("expected clients skill to include %q: %s", needle, clients)
		}
	}
}

func TestCategoriesSkillIncludesRequestDiscovery(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	categories := byPath[filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "categories", "SKILL.md"))]
	want := []string{
		"fakturownia schema category list --json",
		"fakturownia schema category create --json",
		"request_body_schema",
		"`--input` accepts inline JSON, `@file`, or `-` for stdin",
	}
	for _, needle := range want {
		if !strings.Contains(categories, needle) {
			t.Fatalf("expected categories skill to include %q: %s", needle, categories)
		}
	}
}

func TestPaymentsSkillIncludesRequestDiscovery(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	payments := byPath[filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "payments", "SKILL.md"))]
	want := []string{
		"fakturownia schema payment list --json",
		"fakturownia schema payment create --json",
		"`--include invoices`",
		"invoice_ids[]",
	}
	for _, needle := range want {
		if !strings.Contains(payments, needle) {
			t.Fatalf("expected payments skill to include %q: %s", needle, payments)
		}
	}
}

func TestBankAccountsSkillIncludesRequestDiscovery(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	bankAccounts := byPath[filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "bank-accounts", "SKILL.md"))]
	want := []string{
		"fakturownia schema bank-account list --json",
		"fakturownia schema bank-account create --json",
		"bank_account_version_departments[]",
		"buyer_mass_payment_code",
	}
	for _, needle := range want {
		if !strings.Contains(bankAccounts, needle) {
			t.Fatalf("expected bank-accounts skill to include %q: %s", needle, bankAccounts)
		}
	}
}

func TestRecurringsSkillIncludesRequestDiscovery(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	recurrings := byPath[filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "recurrings", "SKILL.md"))]
	want := []string{
		"fakturownia schema recurring list --json",
		"fakturownia schema recurring create --json",
		"request_body_schema",
		"`--input` accepts inline JSON, `@file`, or `-` for stdin",
	}
	for _, needle := range want {
		if !strings.Contains(recurrings, needle) {
			t.Fatalf("expected recurrings skill to include %q: %s", needle, recurrings)
		}
	}
}

func TestProductsSkillIncludesRequestDiscovery(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	products := byPath[filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "products", "SKILL.md"))]
	want := []string{
		"fakturownia schema product list --json",
		"fakturownia schema product create --json",
		"request_body_schema",
		"package_products_details",
	}
	for _, needle := range want {
		if !strings.Contains(products, needle) {
			t.Fatalf("expected products skill to include %q: %s", needle, products)
		}
	}
}

func TestPriceListsSkillIncludesRequestDiscovery(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	priceLists := byPath[filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "price-lists", "SKILL.md"))]
	want := []string{
		"fakturownia schema price-list list --json",
		"fakturownia schema price-list create --json",
		"price_list_positions_attributes",
	}
	for _, needle := range want {
		if !strings.Contains(priceLists, needle) {
			t.Fatalf("expected price-lists skill to include %q: %s", needle, priceLists)
		}
	}
}

func TestWarehousesSkillIncludesRequestDiscovery(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	warehouses := byPath[filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "warehouses", "SKILL.md"))]
	want := []string{
		"fakturownia schema warehouse list --json",
		"fakturownia schema warehouse create --json",
		"request_body_schema",
		"`--input` accepts inline JSON, `@file`, or `-` for stdin",
	}
	for _, needle := range want {
		if !strings.Contains(warehouses, needle) {
			t.Fatalf("expected warehouses skill to include %q: %s", needle, warehouses)
		}
	}
}

func TestWarehouseActionsSkillIncludesFilterDiscovery(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	actions := byPath[filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "warehouse-actions", "SKILL.md"))]
	want := []string{
		"fakturownia schema warehouse-action list --json",
		"`--warehouse-id`",
		"`--warehouse-document-id`",
		"read-only",
	}
	for _, needle := range want {
		if !strings.Contains(actions, needle) {
			t.Fatalf("expected warehouse-actions skill to include %q: %s", needle, actions)
		}
	}
}

func TestSharedSkillIncludesSelfUpdateGuidance(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	shared := byPath[filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "shared", "SKILL.md"))]
	want := []string{
		"fakturownia self update",
		"--dry-run --json",
		"--version vX.Y.Z",
	}
	for _, needle := range want {
		if !strings.Contains(shared, needle) {
			t.Fatalf("expected shared skill to include %q: %s", needle, shared)
		}
	}
}

func TestWarehouseDocumentsSkillIncludesRequestDiscovery(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	warehouseDocs := byPath[filepath.ToSlash(filepath.Join("skills", "fakturownia", "subskills", "warehouse-documents", "SKILL.md"))]
	want := []string{
		"fakturownia schema warehouse-document list --json",
		"fakturownia schema warehouse-document create --json",
		"invoice_ids[]",
		"warehouse_actions[]",
	}
	for _, needle := range want {
		if !strings.Contains(warehouseDocs, needle) {
			t.Fatalf("expected warehouse-documents skill to include %q: %s", needle, warehouseDocs)
		}
	}
}

func TestRecipeIndexIncludesInvoiceAndRecurringRecipes(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	index := byPath[recipeIndexPath()]
	want := []string{
		"fakturownia-invoice-minimal",
		"fakturownia-recurring-definition",
		"Create a minimal invoice when you already know the client and product IDs.",
		"Manage recurring invoice definitions through the dedicated recurring noun.",
	}
	for _, needle := range want {
		if !strings.Contains(index, needle) {
			t.Fatalf("expected recipe index to include %q: %s", needle, index)
		}
	}
}

func TestSkillFrontmatterMatchesMetadata(t *testing.T) {
	t.Parallel()

	files, err := RenderSkillFiles()
	if err != nil {
		t.Fatalf("RenderSkillFiles() error = %v", err)
	}
	byPath := generatedFileMap(files)

	bundle := SkillBundle()
	rootMeta := parseSkillFrontmatter(t, byPath[bundleRootSkillPath()])
	if rootMeta["name"] != bundle.Name {
		t.Fatalf("expected root skill name %q, got %q", bundle.Name, rootMeta["name"])
	}
	if rootMeta["description"] != bundle.Description {
		t.Fatalf("expected root skill description %q, got %q", bundle.Description, rootMeta["description"])
	}

	for _, area := range bundle.Areas {
		meta := parseSkillFrontmatter(t, byPath[skillAreaPath(area)])
		if meta["name"] != area.Name {
			t.Fatalf("expected area skill name %q, got %q", area.Name, meta["name"])
		}
		if meta["description"] != area.Description {
			t.Fatalf("expected area skill description %q, got %q", area.Description, meta["description"])
		}
	}
}

func parseSkillFrontmatter(t *testing.T, content string) map[string]string {
	t.Helper()

	lines := strings.Split(content, "\n")
	if len(lines) < 3 || lines[0] != "---" {
		t.Fatalf("missing frontmatter: %q", content)
	}

	meta := map[string]string{}
	for _, line := range lines[1:] {
		if line == "---" {
			return meta
		}
		if strings.HasPrefix(line, "name: ") {
			meta["name"] = strings.Trim(strings.TrimPrefix(line, "name: "), "\"")
		}
		if strings.HasPrefix(line, "description: ") {
			meta["description"] = strings.Trim(strings.TrimPrefix(line, "description: "), "\"")
		}
	}
	t.Fatalf("frontmatter was not terminated")
	return nil
}
