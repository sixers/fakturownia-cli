package spec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type generatedSkillFile struct {
	Path    string
	Content string
}

func RenderSkillFiles() ([]generatedSkillFile, error) {
	bundle := SkillBundle()
	if err := validateSkillBundle(bundle); err != nil {
		return nil, err
	}
	recipes := Recipes()

	files := make([]generatedSkillFile, 0, len(bundle.Areas)+len(recipes)+4)
	rootContent, err := renderRootSkill(bundle)
	if err != nil {
		return nil, err
	}
	files = append(files, generatedSkillFile{Path: bundleRootSkillPath(), Content: rootContent})

	indexContent, err := renderBundleSkillsIndex(bundle)
	if err != nil {
		return nil, err
	}
	files = append(files, generatedSkillFile{Path: bundleIndexPath(), Content: indexContent})

	recipeIndexContent, err := renderRecipesIndex(recipes)
	if err != nil {
		return nil, err
	}
	files = append(files, generatedSkillFile{Path: recipeIndexPath(), Content: recipeIndexContent})

	for _, area := range bundle.Areas {
		content, err := renderAreaSkill(bundle, area)
		if err != nil {
			return nil, err
		}
		files = append(files, generatedSkillFile{Path: skillAreaPath(area), Content: content})
	}

	for _, recipe := range recipes {
		content, err := renderRecipeSkill(bundle, recipe)
		if err != nil {
			return nil, err
		}
		files = append(files, generatedSkillFile{Path: recipePath(recipe), Content: content})
	}

	docsContent, err := renderDocsSkillsIndex(bundle)
	if err != nil {
		return nil, err
	}
	files = append(files, generatedSkillFile{Path: docsSkillsPath(), Content: docsContent})

	slices.SortFunc(files, func(a, b generatedSkillFile) int {
		return strings.Compare(a.Path, b.Path)
	})
	return files, nil
}

func GenerateSkillFiles(repoRoot string) error {
	files, err := RenderSkillFiles()
	if err != nil {
		return err
	}
	for _, file := range files {
		target := filepath.Join(repoRoot, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, []byte(file.Content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func CheckSkillFiles(repoRoot string) error {
	files, err := RenderSkillFiles()
	if err != nil {
		return err
	}
	var mismatches []string
	for _, file := range files {
		target := filepath.Join(repoRoot, filepath.FromSlash(file.Path))
		current, readErr := os.ReadFile(target)
		if readErr != nil {
			mismatches = append(mismatches, fmt.Sprintf("%s: %v", file.Path, readErr))
			continue
		}
		if normalizeGeneratedText(string(current)) != normalizeGeneratedText(file.Content) {
			mismatches = append(mismatches, file.Path)
		}
	}
	if len(mismatches) > 0 {
		slices.Sort(mismatches)
		return fmt.Errorf("generated skill files are out of date:\n%s", strings.Join(mismatches, "\n"))
	}
	return nil
}

func renderRootSkill(bundle SkillBundleSpec) (string, error) {
	var b strings.Builder
	writeSkillFrontmatter(&b, bundle.Name, bundle.Description, skillDocMeta{
		Category:      "bundle",
		CLIHelp:       bundle.CLIHelp,
		RequiresBins:  bundle.RequiresBins,
		DiscoveryHint: bundle.DiscoveryHint,
	})
	writeGeneratedHeader(&b)
	b.WriteString("# ")
	b.WriteString(bundle.Title)
	b.WriteString("\n\n")
	b.WriteString("Use this as the entrypoint for the generated Fakturownia CLI skill bundle.\n\n")
	b.WriteString("## Start Here\n\n")
	b.WriteString("1. Open the [skills index](references/skills-index.md) to choose the right subskill.\n")
	b.WriteString("2. Read [fakturownia-shared](subskills/shared/SKILL.md) for auth prerequisites, global flags, output behavior, and schema discovery.\n")
	b.WriteString("3. Then open the subskill that matches the task area.\n\n")
	b.WriteString("## Subskills\n\n")
	for _, area := range bundle.Areas {
		fmt.Fprintf(&b, "- [%s](subskills/%s/SKILL.md): %s\n", area.Name, area.Key, area.Description)
	}
	b.WriteString("\n")
	b.WriteString("## Recipes\n\n")
	b.WriteString("- Open the [recipes index](recipes/index.md) for higher-level invoice and recurring workflows from the upstream README.\n\n")
	b.WriteString("## CLI Entry Point\n\n")
	fmt.Fprintf(&b, "```bash\n%s\n```\n", bundle.CLIHelp)
	return b.String(), nil
}

func renderBundleSkillsIndex(bundle SkillBundleSpec) (string, error) {
	var b strings.Builder
	writeGeneratedHeader(&b)
	b.WriteString("# Skills Index\n\n")
	b.WriteString("Start with [fakturownia-shared](../subskills/shared/SKILL.md), then open the skill that matches the task.\n\n")
	writeIndexSection(&b, "Core", "Core workflow and support skills for auth, schema discovery, and diagnostics.", skillsByCategory(bundle, "core"), "../subskills")
	writeIndexSection(&b, "API Areas", "Task-focused API area skills.", skillsByCategory(bundle, "api-area"), "../subskills")
	writeRecipeIndexSection(&b, Recipes(), "../recipes")
	return b.String(), nil
}

func renderDocsSkillsIndex(bundle SkillBundleSpec) (string, error) {
	var b strings.Builder
	writeGeneratedHeader(&b)
	b.WriteString("# Skills Index\n\n")
	b.WriteString("Generated skill docs for the installable `fakturownia` bundle.\n\n")
	b.WriteString("## Bundle\n\n")
	b.WriteString("| Skill | Description |\n")
	b.WriteString("| --- | --- |\n")
	fmt.Fprintf(&b, "| [%s](../%s) | %s |\n\n", bundle.Name, bundleRootSkillPath(), bundle.Description)
	writeIndexSection(&b, "Core", "Core workflow and support skills.", skillsByCategory(bundle, "core"), "../skills/fakturownia/subskills")
	writeIndexSection(&b, "API Areas", "Task-focused API area skills.", skillsByCategory(bundle, "api-area"), "../skills/fakturownia/subskills")
	writeRecipeIndexSection(&b, Recipes(), "../skills/fakturownia/recipes")
	return b.String(), nil
}

func renderAreaSkill(bundle SkillBundleSpec, area SkillAreaSpec) (string, error) {
	commandSpecs, err := commandsForSkillArea(area)
	if err != nil {
		return "", err
	}
	exampleSpecs, err := exampleCommandsForSkillArea(area)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	writeSkillFrontmatter(&b, area.Name, area.Description, skillDocMeta{
		Bundle:        bundle.Name,
		Category:      area.Category,
		Prerequisite:  relatedSkillName(bundle, area.Prerequisite),
		RelatedSkills: relatedSkillNames(bundle, area),
		CommandRefs:   formatCommandRefs(area.CommandRefs),
		CLIHelp:       area.CLIHelp,
		RequiresBins:  area.RequiresBins,
		DiscoveryHint: area.DiscoveryHint,
	})
	writeGeneratedHeader(&b)
	b.WriteString("# ")
	b.WriteString(area.Title)
	b.WriteString("\n\n")
	if area.Prerequisite != "" {
		fmt.Fprintf(&b, "> **PREREQUISITE:** Read [`%s`](%s) first.\n\n", relatedSkillName(bundle, area.Prerequisite), relativeSkillLink(area, area.Prerequisite))
	}
	if len(area.WhenToUse) > 0 {
		b.WriteString("## Use This Skill When\n\n")
		for _, line := range area.WhenToUse {
			fmt.Fprintf(&b, "- %s\n", line)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Covered Commands\n\n")
	if len(commandSpecs) == 0 {
		b.WriteString("- This skill documents shared CLI behavior rather than owning a specific command group.\n\n")
	} else {
		for _, spec := range commandSpecs {
			fmt.Fprintf(&b, "- `%s` — %s\n", skillAreaCommandTitle(spec), spec.Short)
		}
		b.WriteString("\n")
	}

	flags := flagsForCommands(commandSpecs)
	if area.Key == "shared" {
		flags = GlobalFlags()
	}
	writeFlagSection(&b, "Notable Flags", flags, area.Key == "shared")

	envVars := envVarsForCommands(commandSpecs)
	if area.Key == "shared" {
		envVars = envVarsForCommands(Registry())
	}
	writeEnvSection(&b, envVars)

	if area.Key == "shared" {
		b.WriteString("## Output and Discovery\n\n")
		b.WriteString("- `--json` or `--output json` writes the structured envelope to stdout.\n")
		b.WriteString("- `--fields` projects JSON envelope `data` fields.\n")
		b.WriteString("- `--columns` only changes human table rendering.\n")
		b.WriteString("- `--raw` emits the upstream JSON body when the command supports it.\n")
		b.WriteString("- `--quiet` emits bare values when exactly one field or column remains.\n")
		b.WriteString("- Use `fakturownia schema list --json` and `fakturownia schema <noun> <verb> --json` before constructing calls programmatically.\n\n")
		b.WriteString("## Binary Maintenance\n\n")
		b.WriteString("- Use `fakturownia self update` to replace the running binary with the latest GitHub Release build.\n")
		b.WriteString("- Add `--version vX.Y.Z` to pin a specific release.\n")
		b.WriteString("- Add `--dry-run --json` to preview the download URLs and target install path without modifying the binary.\n\n")
	}

	if area.Key == "auth" {
		writeAuthDiscoverySection(&b)
	}

	if area.Key == "accounts" {
		writeAccountsDiscoverySection(&b)
	}

	if area.Key == "departments" {
		writeDepartmentsDiscoverySection(&b)
	}

	if area.Key == "issuers" {
		writeIssuersDiscoverySection(&b)
	}

	if area.Key == "users" {
		writeUsersDiscoverySection(&b)
	}

	if area.Key == "invoices" {
		writeInvoicesDiscoverySection(&b)
	}

	if area.Key == "recurrings" {
		writeRecurringsDiscoverySection(&b)
	}

	if area.Key == "clients" {
		writeClientsDiscoverySection(&b)
	}

	if area.Key == "categories" {
		writeCategoriesDiscoverySection(&b)
	}

	if area.Key == "payments" {
		writePaymentsDiscoverySection(&b)
	}

	if area.Key == "products" {
		writeProductsDiscoverySection(&b)
	}

	if area.Key == "warehouses" {
		writeWarehousesDiscoverySection(&b)
	}

	if area.Key == "warehouse-actions" {
		writeWarehouseActionsDiscoverySection(&b)
	}

	if area.Key == "webhooks" {
		writeWebhooksDiscoverySection(&b)
	}

	if area.Key == "schema" {
		b.WriteString("## Schema Output\n\n")
		b.WriteString("- `schema list` enumerates supported commands.\n")
		b.WriteString("- `schema <noun> <verb>` returns flags, env vars, examples, exit codes, output modes, and output schema details.\n")
		b.WriteString("- For account, auth exchange, category, client, department, invoice, issuer, payment, product, price-list, recurring, warehouse, warehouse-action, warehouse-document, and webhook commands, inspect `output.known_fields`, `path_syntax`, and the generated `data_schema` before building `--fields` selectors.\n")
		b.WriteString("- For account, category, client, department, invoice, issuer, payment, product, price-list, recurring, user, warehouse, warehouse-document, and webhook write commands, inspect `request_body_schema` before constructing `--input` payloads.\n\n")
	}

	if area.Key == "doctor" {
		b.WriteString("## Diagnostics Focus\n\n")
		b.WriteString("- `doctor run` checks config-path resolution, keychain access, DNS/TLS reachability, and authenticated API access.\n")
		b.WriteString("- Add `--check-release-integrity` when you need checksum verification against the published release.\n\n")
	}

	writeAreaRecipesSection(&b, area)

	examples := examplesForCommands(exampleSpecs)
	if len(examples) > 0 {
		b.WriteString("## Examples\n\n```bash\n")
		for _, example := range examples {
			b.WriteString(example)
			b.WriteString("\n")
		}
		b.WriteString("```\n\n")
	}

	related := relatedAreas(bundle, area)
	if len(related) > 0 {
		b.WriteString("## Related Skills\n\n")
		for _, skill := range related {
			fmt.Fprintf(&b, "- [%s](%s)\n", skill.Name, relativeSkillLink(area, skill.Key))
		}
	}

	return b.String(), nil
}

func renderRecipesIndex(recipes []RecipeSpec) (string, error) {
	var b strings.Builder
	writeGeneratedHeader(&b)
	b.WriteString("# Recipes\n\n")
	b.WriteString("Generated workflow recipes for common invoice and recurring tasks. Start with the relevant area skill, then open the matching recipe.\n\n")
	writeRecipeIndexSection(&b, recipes, ".")
	return b.String(), nil
}

func renderRecipeSkill(bundle SkillBundleSpec, recipe RecipeSpec) (string, error) {
	var b strings.Builder
	writeSkillFrontmatter(&b, recipe.Name, recipe.Description, skillDocMeta{
		Bundle:        bundle.Name,
		Category:      "recipe",
		RelatedSkills: recipeRelatedSkillNames(bundle, recipe),
		CommandRefs:   formatCommandRefs(recipe.CommandRefs),
	})
	writeGeneratedHeader(&b)
	b.WriteString("# ")
	b.WriteString(recipe.Title)
	b.WriteString("\n\n")
	if area, ok := skillAreaByKey(bundle, recipe.AreaKey); ok {
		fmt.Fprintf(&b, "> Read [%s](%s) first.\n\n", area.Name, relativeRecipeAreaLink(recipe, area))
	}
	b.WriteString(recipe.Markdown)
	if !strings.HasSuffix(recipe.Markdown, "\n") {
		b.WriteString("\n")
	}
	if len(recipe.RelatedSkills) > 0 {
		b.WriteString("\n## Related Skills\n\n")
		for _, key := range recipe.RelatedSkills {
			if area, ok := skillAreaByKey(bundle, key); ok {
				fmt.Fprintf(&b, "- [%s](%s)\n", area.Name, relativeRecipeAreaLink(recipe, area))
			}
		}
	}
	return b.String(), nil
}

func writeInvoicesDiscoverySection(b *strings.Builder) {
	b.WriteString("## Output Discovery\n\n")
	b.WriteString("- Use `fakturownia schema invoice list --json` and `fakturownia schema invoice get --json` before building selectors.\n")
	b.WriteString("- Read `output.known_fields` to discover README-backed invoice field names.\n")
	b.WriteString("- Nested selectors use `dot_bracket` paths such as `positions[].name`.\n")
	b.WriteString("- Use `fakturownia schema invoice create --json` and `fakturownia schema invoice update --json` before building invoice payloads.\n")
	b.WriteString("- `output.known_fields` is curated, not exhaustive, so valid undocumented paths may still work.\n\n")
}

func writeAuthDiscoverySection(b *strings.Builder) {
	b.WriteString("## Credential Exchange\n\n")
	b.WriteString("- Use `fakturownia auth exchange --login ... --password ... --json` to exchange login credentials for account metadata and an API token when the upstream account has one.\n")
	b.WriteString("- Structured output is sanitized and reports `api_token_present` instead of exposing the token directly.\n")
	b.WriteString("- Use `--raw` only when you explicitly need the exact upstream login response, including secrets.\n")
	b.WriteString("- By default, `auth exchange` stores the returned token under the returned prefix; add `--save-as` to override the saved profile name.\n\n")
}

func writeAccountsDiscoverySection(b *strings.Builder) {
	b.WriteString("## Output and Request Discovery\n\n")
	b.WriteString("- Use `fakturownia schema account get --json` to inspect sanitized account output fields such as `prefix`, `url`, `login`, `email`, and `api_token_present`.\n")
	b.WriteString("- Use `fakturownia schema account create --json` before building the full top-level account request object.\n")
	b.WriteString("- Unlike most CRUD nouns, `account create` accepts the full upstream request object, including top-level `account`, `user`, `company`, and optional `integration_token`.\n")
	b.WriteString("- Structured output is sanitized and omits the raw returned API token; use `--raw` only when you explicitly need the exact upstream response.\n\n")
}

func writeDepartmentsDiscoverySection(b *strings.Builder) {
	b.WriteString("## Output and Request Discovery\n\n")
	b.WriteString("- Use `fakturownia schema department list --json` and `fakturownia schema department get --json` before building selectors.\n")
	b.WriteString("- Read `output.known_fields` to discover README-backed department fields such as `name`, `shortcut`, and `tax_no`.\n")
	b.WriteString("- Use `fakturownia schema department create --json` and `fakturownia schema department update --json` to inspect `request_body_schema` and accepted `--input` modes.\n")
	b.WriteString("- `--input` accepts inline JSON, `@file`, or `-` for stdin, and the CLI wraps the inner object into the upstream `department` envelope.\n")
	b.WriteString("- `department set-logo` uploads multipart content using `department[logo]`; when using `--file -`, pass `--name` as well.\n\n")
}

func writeIssuersDiscoverySection(b *strings.Builder) {
	b.WriteString("## Output and Request Discovery\n\n")
	b.WriteString("- Use `fakturownia schema issuer list --json` and `fakturownia schema issuer get --json` before building selectors.\n")
	b.WriteString("- Read `output.known_fields` to discover README-backed issuer output fields such as `name` and `tax_no`.\n")
	b.WriteString("- Use `fakturownia schema issuer create --json` and `fakturownia schema issuer update --json` to inspect `request_body_schema` and accepted `--input` modes.\n")
	b.WriteString("- `--input` accepts inline JSON, `@file`, or `-` for stdin, and the CLI wraps the inner object into the upstream `issuer` envelope.\n\n")
}

func writeUsersDiscoverySection(b *strings.Builder) {
	b.WriteString("## Request Discovery\n\n")
	b.WriteString("- Use `fakturownia schema user create --json` before building invite or password-based user payloads.\n")
	b.WriteString("- The CLI accepts the inner `user` object and wraps it into the upstream `{ \"user\": ... }` envelope.\n")
	b.WriteString("- Pass `--integration-token` separately; the active profile supplies `api_token` automatically.\n")
	b.WriteString("- Inspect request fields such as `invite`, `email`, `password`, `role`, and `department_ids[]` before constructing the payload.\n\n")
}

func writeRecurringsDiscoverySection(b *strings.Builder) {
	b.WriteString("## Output and Request Discovery\n\n")
	b.WriteString("- Use `fakturownia schema recurring list --json` to inspect README-backed recurring output fields such as `name`, `every`, and `next_invoice_date`.\n")
	b.WriteString("- Use `fakturownia schema recurring create --json` and `fakturownia schema recurring update --json` to inspect `request_body_schema` and accepted `--input` modes.\n")
	b.WriteString("- `--input` accepts inline JSON, `@file`, or `-` for stdin, and the CLI wraps the inner object into the upstream `recurring` envelope.\n\n")
}

func writeClientsDiscoverySection(b *strings.Builder) {
	b.WriteString("## Output and Request Discovery\n\n")
	b.WriteString("- Use `fakturownia schema client list --json` and `fakturownia schema client get --json` before building selectors.\n")
	b.WriteString("- Read `output.known_fields` to discover README-backed client output fields such as `name`, `tax_no`, or `tag_list[]`.\n")
	b.WriteString("- Use `fakturownia schema client create --json` and `fakturownia schema client update --json` to inspect `request_body_schema` and accepted `--input` modes.\n")
	b.WriteString("- `--input` accepts inline JSON, `@file`, or `-` for stdin, and the CLI wraps the inner object into the upstream `client` envelope.\n\n")
}

func writeCategoriesDiscoverySection(b *strings.Builder) {
	b.WriteString("## Output and Request Discovery\n\n")
	b.WriteString("- Use `fakturownia schema category list --json` and `fakturownia schema category get --json` before building selectors.\n")
	b.WriteString("- Read `output.known_fields` to discover README-backed category output fields such as `name` and `description`.\n")
	b.WriteString("- Use `fakturownia schema category create --json` and `fakturownia schema category update --json` to inspect `request_body_schema` and accepted `--input` modes.\n")
	b.WriteString("- `--input` accepts inline JSON, `@file`, or `-` for stdin, and the CLI wraps the inner object into the upstream `category` envelope.\n\n")
}

func writePaymentsDiscoverySection(b *strings.Builder) {
	b.WriteString("## Output and Request Discovery\n\n")
	b.WriteString("- Use `fakturownia schema payment list --json` and `fakturownia schema payment get --json` before building selectors.\n")
	b.WriteString("- Read `output.known_fields` to discover README-backed payment output fields such as `name`, `price`, `paid`, `kind`, and conditional `invoices[]`.\n")
	b.WriteString("- Use `fakturownia schema payment create --json` and `fakturownia schema payment update --json` to inspect `request_body_schema` and accepted `--input` modes.\n")
	b.WriteString("- `--input` accepts inline JSON, `@file`, or `-` for stdin, and the CLI wraps the inner object into the upstream `banking_payment` envelope.\n")
	b.WriteString("- Use `--include invoices` on `payment list` when you need the README-backed include mode, and inspect request fields such as `invoice_id` and `invoice_ids[]` before creating or updating payments.\n\n")
}

func writeProductsDiscoverySection(b *strings.Builder) {
	b.WriteString("## Output and Request Discovery\n\n")
	b.WriteString("- Use `fakturownia schema product list --json` and `fakturownia schema product get --json` before building selectors.\n")
	b.WriteString("- Read `output.known_fields` to discover README-backed product output fields such as `name`, `code`, `tag_list[]`, `gtu_codes[]`, or `stock_level`.\n")
	b.WriteString("- Use `fakturownia schema product create --json` and `fakturownia schema product update --json` to inspect `request_body_schema` and accepted `--input` modes.\n")
	b.WriteString("- `--input` accepts inline JSON, `@file`, or `-` for stdin, and the CLI wraps the inner object into the upstream `product` envelope.\n")
	b.WriteString("- Package-product payloads use `package_products_details` as an open object whose values contain `id` and `quantity`.\n\n")
}

func writeWarehousesDiscoverySection(b *strings.Builder) {
	b.WriteString("## Output and Request Discovery\n\n")
	b.WriteString("- Use `fakturownia schema warehouse list --json` and `fakturownia schema warehouse get --json` before building selectors.\n")
	b.WriteString("- Read `output.known_fields` to discover conservative README-backed warehouse fields such as `name`, `kind`, and `description`.\n")
	b.WriteString("- Use `fakturownia schema warehouse create --json` and `fakturownia schema warehouse update --json` to inspect `request_body_schema` and accepted `--input` modes.\n")
	b.WriteString("- `--input` accepts inline JSON, `@file`, or `-` for stdin, and the CLI wraps the inner object into the upstream `warehouse` envelope.\n\n")
}

func writeWarehouseActionsDiscoverySection(b *strings.Builder) {
	b.WriteString("## Output Discovery\n\n")
	b.WriteString("- Use `fakturownia schema warehouse-action list --json` before building selectors.\n")
	b.WriteString("- Read `output.known_fields` to discover conservative README-backed action fields such as `product_id`, `quantity`, `warehouse_id`, `warehouse_document_id`, and `warehouse2_id`.\n")
	b.WriteString("- `warehouse-action list` exposes only explicit first-class filter flags in v1: `--warehouse-id`, `--kind`, `--product-id`, `--date-from`, `--date-to`, `--from-warehouse-document`, `--to-warehouse-document`, and `--warehouse-document-id`.\n")
	b.WriteString("- There is no request body schema for warehouse actions in v1 because the CLI exposes this noun as read-only.\n\n")
}

func writeWebhooksDiscoverySection(b *strings.Builder) {
	b.WriteString("## Output and Request Discovery\n\n")
	b.WriteString("- Use `fakturownia schema webhook list --json` and `fakturownia schema webhook get --json` before building selectors.\n")
	b.WriteString("- Read `output.known_fields` to discover conservative webhook fields such as `kind`, `url`, `api_token`, `active`, `created_at`, and `updated_at`.\n")
	b.WriteString("- Use `fakturownia schema webhook create --json` and `fakturownia schema webhook update --json` to inspect `request_body_schema` before building webhook payloads.\n")
	b.WriteString("- Unlike most CRUD nouns, webhook create and update accept the full top-level request object because the direct README documents endpoints but not a wrapper key.\n")
	b.WriteString("- Start from the curated `kind` values in schema output and keep the payload conservative rather than inventing undocumented fields.\n\n")
}

func writeAreaRecipesSection(b *strings.Builder, area SkillAreaSpec) {
	recipes := recipesForArea(area.Key)
	if len(recipes) == 0 {
		return
	}
	b.WriteString("## Recipes\n\n")
	for _, recipe := range recipes {
		fmt.Fprintf(b, "- [%s](%s): %s\n", recipe.Name, relativeAreaRecipeLink(area, recipe), recipe.Description)
	}
	b.WriteString("\n")
}

func writeGeneratedHeader(b *strings.Builder) {
	b.WriteString(generatedByLine)
	b.WriteString("\n\n")
}

type skillDocMeta struct {
	Bundle        string
	Category      string
	Prerequisite  string
	RelatedSkills []string
	CommandRefs   []string
	CLIHelp       string
	RequiresBins  []string
	DiscoveryHint string
}

func writeSkillFrontmatter(b *strings.Builder, name, description string, meta skillDocMeta) {
	b.WriteString("---\n")
	writeYAMLString(b, "name", name, 0)
	writeYAMLString(b, "description", description, 0)
	if hasSkillMeta(meta) {
		b.WriteString("metadata:\n")
		writeYAMLString(b, "bundle", meta.Bundle, 2)
		writeYAMLString(b, "category", meta.Category, 2)
		writeYAMLString(b, "prerequisite", meta.Prerequisite, 2)
		writeYAMLStringList(b, "related_skills", meta.RelatedSkills, 2)
		writeYAMLStringList(b, "command_refs", meta.CommandRefs, 2)
		writeYAMLString(b, "cli_help", meta.CLIHelp, 2)
		writeYAMLStringList(b, "requires_bins", meta.RequiresBins, 2)
		writeYAMLString(b, "discovery_hint", meta.DiscoveryHint, 2)
	}
	b.WriteString("---\n\n")
}

func hasSkillMeta(meta skillDocMeta) bool {
	return meta.Bundle != "" ||
		meta.Category != "" ||
		meta.Prerequisite != "" ||
		len(meta.RelatedSkills) > 0 ||
		len(meta.CommandRefs) > 0 ||
		meta.CLIHelp != "" ||
		len(meta.RequiresBins) > 0 ||
		meta.DiscoveryHint != ""
}

func writeYAMLString(b *strings.Builder, key, value string, indent int) {
	if strings.TrimSpace(value) == "" {
		return
	}
	fmt.Fprintf(b, "%s%s: %s\n", strings.Repeat(" ", indent), key, yamlString(value))
}

func writeYAMLStringList(b *strings.Builder, key string, values []string, indent int) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(b, "%s%s:\n", strings.Repeat(" ", indent), key)
	for _, value := range values {
		fmt.Fprintf(b, "%s- %s\n", strings.Repeat(" ", indent+2), yamlString(value))
	}
}

func yamlString(value string) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		return strconv.Quote(value)
	}
	return strings.TrimSpace(b.String())
}

func writeFlagSection(b *strings.Builder, title string, flags []FlagSpec, includeOutputNote bool) {
	b.WriteString("## ")
	b.WriteString(title)
	b.WriteString("\n\n")
	if len(flags) == 0 {
		b.WriteString("- No area-specific local flags. Use the shared global flags instead.\n\n")
		return
	}
	for _, flag := range flags {
		var parts []string
		if flag.Required {
			parts = append(parts, "required")
		}
		if flag.Default != "" {
			parts = append(parts, "default "+fmt.Sprintf("`%s`", flag.Default))
		}
		if len(flag.Enum) > 0 {
			parts = append(parts, "enum "+fmt.Sprintf("`%s`", strings.Join(flag.Enum, "`, `")))
		}
		suffix := ""
		if len(parts) > 0 {
			suffix = " (" + strings.Join(parts, ", ") + ")"
		}
		fmt.Fprintf(b, "- `--%s`%s: %s\n", flag.Name, suffix, flag.Description)
	}
	if includeOutputNote {
		b.WriteString("- `--json` aliases `--output json`; see this skill's output section for the shared behavior.\n")
	}
	b.WriteString("\n")
}

func writeEnvSection(b *strings.Builder, envVars []EnvVarSpec) {
	b.WriteString("## Environment\n\n")
	if len(envVars) == 0 {
		b.WriteString("- No area-specific environment variables. Use the shared skill for global CLI discovery.\n\n")
		return
	}
	for _, env := range envVars {
		fmt.Fprintf(b, "- `%s`: %s\n", env.Name, env.Description)
	}
	b.WriteString("\n")
}

func formatCommandRefs(refs []SkillCommandRef) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		out = append(out, strings.TrimSpace(ref.Noun+" "+ref.Verb))
	}
	return out
}

func relatedSkillName(bundle SkillBundleSpec, key string) string {
	if key == "" {
		return ""
	}
	area, ok := skillAreaByKey(bundle, key)
	if !ok {
		return key
	}
	return area.Name
}

func relatedSkillNames(bundle SkillBundleSpec, area SkillAreaSpec) []string {
	related := relatedAreas(bundle, area)
	names := make([]string, 0, len(related))
	for _, skill := range related {
		names = append(names, skill.Name)
	}
	return names
}

func relativeSkillLink(from SkillAreaSpec, toKey string) string {
	fromPath := path.Join("subskills", from.Key, "SKILL.md")
	toPath := path.Join("subskills", toKey, "SKILL.md")
	rel, err := filepath.Rel(filepath.FromSlash(path.Dir(fromPath)), filepath.FromSlash(toPath))
	if err != nil {
		return "../" + toKey + "/SKILL.md"
	}
	return filepath.ToSlash(rel)
}

func writeIndexSection(b *strings.Builder, title, intro string, areas []SkillAreaSpec, base string) {
	b.WriteString("## ")
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(intro)
	b.WriteString("\n\n")
	b.WriteString("| Skill | Description |\n")
	b.WriteString("| --- | --- |\n")
	for _, area := range areas {
		link := path.Join(base, area.Key, "SKILL.md")
		fmt.Fprintf(b, "| [%s](%s) | %s |\n", area.Name, link, area.Description)
	}
	b.WriteString("\n")
}

func writeRecipeIndexSection(b *strings.Builder, recipes []RecipeSpec, base string) {
	b.WriteString("## Recipes\n\n")
	b.WriteString("Generated workflow recipes for common invoice and recurring tasks.\n\n")
	b.WriteString("| Recipe | Description |\n")
	b.WriteString("| --- | --- |\n")
	for _, recipe := range recipes {
		link := path.Join(base, recipe.Key, "SKILL.md")
		fmt.Fprintf(b, "| [%s](%s) | %s |\n", recipe.Name, link, recipe.Description)
	}
	b.WriteString("\n")
}

func generatedFileMap(files []generatedSkillFile) map[string]string {
	out := make(map[string]string, len(files))
	for _, file := range files {
		out[file.Path] = file.Content
	}
	return out
}

func resolveRelativeLink(fromFile, link string) string {
	clean := path.Clean(path.Join(path.Dir(filepath.ToSlash(fromFile)), link))
	return strings.TrimPrefix(clean, "./")
}

func extractMarkdownLinks(content string) []string {
	var links []string
	for _, line := range strings.Split(content, "\n") {
		remaining := line
		for {
			start := strings.Index(remaining, "](")
			if start == -1 {
				break
			}
			remaining = remaining[start+2:]
			end := strings.Index(remaining, ")")
			if end == -1 {
				break
			}
			links = append(links, remaining[:end])
			remaining = remaining[end+1:]
		}
	}
	return links
}

func normalizeGeneratedText(value string) string {
	return strings.ReplaceAll(value, "\r\n", "\n")
}

func recipesForArea(areaKey string) []RecipeSpec {
	out := make([]RecipeSpec, 0)
	for _, recipe := range Recipes() {
		if recipe.AreaKey == areaKey {
			out = append(out, recipe)
		}
	}
	return out
}

func recipeRelatedSkillNames(bundle SkillBundleSpec, recipe RecipeSpec) []string {
	names := make([]string, 0, len(recipe.RelatedSkills))
	for _, key := range recipe.RelatedSkills {
		if area, ok := skillAreaByKey(bundle, key); ok {
			names = append(names, area.Name)
		}
	}
	return names
}

func relativeAreaRecipeLink(area SkillAreaSpec, recipe RecipeSpec) string {
	fromPath := path.Join("subskills", area.Key, "SKILL.md")
	toPath := path.Join("recipes", recipe.Key, "SKILL.md")
	rel, err := filepath.Rel(filepath.FromSlash(path.Dir(fromPath)), filepath.FromSlash(toPath))
	if err != nil {
		return "../../recipes/" + recipe.Key + "/SKILL.md"
	}
	return filepath.ToSlash(rel)
}

func relativeRecipeAreaLink(recipe RecipeSpec, area SkillAreaSpec) string {
	fromPath := path.Join("recipes", recipe.Key, "SKILL.md")
	toPath := path.Join("subskills", area.Key, "SKILL.md")
	rel, err := filepath.Rel(filepath.FromSlash(path.Dir(fromPath)), filepath.FromSlash(toPath))
	if err != nil {
		return "../../subskills/" + area.Key + "/SKILL.md"
	}
	return filepath.ToSlash(rel)
}
