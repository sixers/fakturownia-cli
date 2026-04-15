package spec

func warehouseListOutputSpec() *OutputSpec {
	return warehouseBaseOutputSpec("array", []string{"warehouse list"})
}

func warehouseGetOutputSpec(commands ...string) *OutputSpec {
	if len(commands) == 0 {
		commands = []string{"warehouse get", "warehouse create", "warehouse update"}
	}
	return warehouseBaseOutputSpec("object", commands)
}

func warehouseRequestBodySpec() *RequestBodySpec {
	return &RequestBodySpec{
		InputFlag:  "input",
		InputModes: []string{"inline_json", "@file", "stdin"},
		WrapperKey: "warehouse",
		OpenEnded:  true,
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		PathSyntax: "dot_bracket",
		KnownFields: []RequestFieldSpec{
			{Path: "name", Type: "string", Description: "Warehouse name", SourceSection: "Dodanie nowego magazynu"},
			{Path: "kind", Type: "string|null", Description: "Warehouse kind", SourceSection: "Dodanie nowego magazynu"},
			{Path: "description", Type: "string|null", Description: "Warehouse description", SourceSection: "Dodanie nowego magazynu"},
		},
		Notes: []string{
			"the CLI accepts the inner warehouse object, then wraps it in the upstream {\"warehouse\": ...} envelope",
			"known_fields is curated from the upstream README and is not exhaustive",
			"warehouse kind is left open-ended because the README examples only show null and do not publish a stable enum",
		},
	}
}

func warehouseBaseOutputSpec(shape string, commands []string) *OutputSpec {
	return &OutputSpec{
		Shape:      shape,
		OpenEnded:  true,
		PathSyntax: "dot_bracket",
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		DefaultColumns: []string{"id", "name", "kind", "description"},
		Notes: []string{
			"known_fields is curated from the upstream README and is not exhaustive",
			"unknown upstream fields may still appear in data and can still be selected when the path syntax is valid",
			"warehouse kind is left open-ended because the README examples only show null and do not publish a stable enum",
		},
		KnownFields: []OutputFieldSpec{
			{Path: "id", Type: "integer", Description: "Warehouse ID", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pobranie pojedycznego magazynu po ID"},
			{Path: "name", Type: "string", Description: "Warehouse name", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Dodanie nowego magazynu"},
			{Path: "kind", Type: "string|null", Description: "Warehouse kind", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "conditional", SourceSection: "Dodanie nowego magazynu"},
			{Path: "description", Type: "string|null", Description: "Warehouse description", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "conditional", SourceSection: "Dodanie nowego magazynu"},
		},
	}
}
