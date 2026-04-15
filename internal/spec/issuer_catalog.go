package spec

func issuerListOutputSpec() *OutputSpec {
	return issuerBaseOutputSpec("array", []string{"issuer list"})
}

func issuerGetOutputSpec(commands ...string) *OutputSpec {
	if len(commands) == 0 {
		commands = []string{"issuer get", "issuer create", "issuer update"}
	}
	return issuerBaseOutputSpec("object", commands)
}

func issuerRequestBodySpec() *RequestBodySpec {
	return &RequestBodySpec{
		InputFlag:  "input",
		InputModes: []string{"inline_json", "@file", "stdin"},
		WrapperKey: "issuer",
		OpenEnded:  true,
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		PathSyntax: "dot_bracket",
		KnownFields: []RequestFieldSpec{
			{Path: "name", Type: "string", Description: "Issuer name", SourceSection: "Dodanie nowego wystawcy"},
			{Path: "tax_no", Type: "string", Description: "Issuer tax number", SourceSection: "Dodanie nowego wystawcy"},
		},
		Notes: []string{
			"the CLI accepts the inner issuer object, then wraps it in the upstream {\"issuer\": ...} envelope",
			"known_fields is curated from the upstream README and is not exhaustive",
		},
	}
}

func issuerBaseOutputSpec(shape string, commands []string) *OutputSpec {
	return &OutputSpec{
		Shape:      shape,
		OpenEnded:  true,
		PathSyntax: "dot_bracket",
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		DefaultColumns: []string{"id", "name", "tax_no"},
		Notes: []string{
			"known_fields is curated from the upstream README and is not exhaustive",
			"unknown upstream fields may still appear in data and can still be selected when the path syntax is valid",
		},
		KnownFields: []OutputFieldSpec{
			{Path: "id", Type: "integer", Description: "Issuer ID", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pobranie wystawcy po ID"},
			{Path: "name", Type: "string", Description: "Issuer name", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Dodanie nowego wystawcy"},
			{Path: "tax_no", Type: "string", Description: "Issuer tax number", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "conditional", SourceSection: "Dodanie nowego wystawcy"},
		},
	}
}
