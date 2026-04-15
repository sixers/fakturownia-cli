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

	files := make([]generatedSkillFile, 0, len(bundle.Areas)+3)
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

	for _, area := range bundle.Areas {
		content, err := renderAreaSkill(bundle, area)
		if err != nil {
			return nil, err
		}
		files = append(files, generatedSkillFile{Path: skillAreaPath(area), Content: content})
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
	}

	if area.Key == "invoices" {
		writeInvoicesDiscoverySection(&b)
	}

	if area.Key == "schema" {
		b.WriteString("## Schema Output\n\n")
		b.WriteString("- `schema list` enumerates supported commands.\n")
		b.WriteString("- `schema <noun> <verb>` returns flags, env vars, examples, exit codes, output modes, and output schema details.\n")
		b.WriteString("- For invoice commands, inspect `output.known_fields`, `path_syntax`, and the generated `data_schema` before building `--fields` selectors.\n\n")
	}

	if area.Key == "doctor" {
		b.WriteString("## Diagnostics Focus\n\n")
		b.WriteString("- `doctor run` checks config-path resolution, keychain access, DNS/TLS reachability, and authenticated API access.\n")
		b.WriteString("- Add `--check-release-integrity` when you need checksum verification against the published release.\n\n")
	}

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

func writeInvoicesDiscoverySection(b *strings.Builder) {
	b.WriteString("## Output Discovery\n\n")
	b.WriteString("- Use `fakturownia schema invoice list --json` and `fakturownia schema invoice get --json` before building selectors.\n")
	b.WriteString("- Read `output.known_fields` to discover README-backed invoice field names.\n")
	b.WriteString("- Nested selectors use `dot_bracket` paths such as `positions[].name`.\n")
	b.WriteString("- `output.known_fields` is curated, not exhaustive, so valid undocumented paths may still work.\n\n")
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
