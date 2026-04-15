package spec

func categoryListOutputSpec() *OutputSpec {
	return categoryBaseOutputSpec("array", []string{"category list"})
}

func categoryGetOutputSpec(commands ...string) *OutputSpec {
	if len(commands) == 0 {
		commands = []string{"category get", "category create", "category update"}
	}
	return categoryBaseOutputSpec("object", commands)
}

func categoryRequestBodySpec() *RequestBodySpec {
	return &RequestBodySpec{
		InputFlag:  "input",
		InputModes: []string{"inline_json", "@file", "stdin"},
		WrapperKey: "category",
		OpenEnded:  true,
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		PathSyntax: "dot_bracket",
		KnownFields: []RequestFieldSpec{
			{Path: "name", Type: "string", Description: "Category name", SourceSection: "Dodanie nowej kategorii"},
			{Path: "description", Type: "string|null", Description: "Category description", SourceSection: "Dodanie nowej kategorii"},
		},
		Notes: []string{
			"the CLI accepts the inner category object, then wraps it in the upstream {\"category\": ...} envelope",
			"known_fields is curated from the upstream README and is not exhaustive",
		},
	}
}

func categoryBaseOutputSpec(shape string, commands []string) *OutputSpec {
	return &OutputSpec{
		Shape:      shape,
		OpenEnded:  true,
		PathSyntax: "dot_bracket",
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		DefaultColumns: []string{"id", "name", "description"},
		Notes: []string{
			"known_fields is curated from the upstream README and is not exhaustive",
			"unknown upstream fields may still appear in data and can still be selected when the path syntax is valid",
		},
		KnownFields: []OutputFieldSpec{
			{Path: "id", Type: "integer", Description: "Category ID", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pobranie pojedycznej kategorii po ID"},
			{Path: "name", Type: "string", Description: "Category name", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Dodanie nowej kategorii"},
			{Path: "description", Type: "string|null", Description: "Category description", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "conditional", SourceSection: "Dodanie nowej kategorii"},
		},
	}
}
