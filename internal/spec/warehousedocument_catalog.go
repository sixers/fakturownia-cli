package spec

func warehouseDocumentListOutputSpec() *OutputSpec {
	return warehouseDocumentBaseOutputSpec("array", []string{"warehouse-document list"}, false)
}

func warehouseDocumentGetOutputSpec(commands ...string) *OutputSpec {
	if len(commands) == 0 {
		commands = []string{"warehouse-document get", "warehouse-document create", "warehouse-document update"}
	}
	return warehouseDocumentBaseOutputSpec("object", commands, true)
}

func warehouseDocumentRequestBodySpec() *RequestBodySpec {
	return &RequestBodySpec{
		InputFlag:  "input",
		InputModes: []string{"inline_json", "@file", "stdin"},
		WrapperKey: "warehouse_document",
		OpenEnded:  true,
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		PathSyntax:  "dot_bracket",
		KnownFields: warehouseDocumentRequestFields(),
		Notes: []string{
			"the CLI accepts the inner warehouse document object, then wraps it in the upstream {\"warehouse_document\": ...} envelope",
			"known_fields is curated from the upstream README and is not exhaustive",
			"warehouse document variants such as mm, pz, and wz are selected through the payload kind field rather than separate CLI verbs",
		},
	}
}

func warehouseDocumentBaseOutputSpec(shape string, commands []string, includeActions bool) *OutputSpec {
	fields := []OutputFieldSpec{
		{Path: "id", Type: "integer", Description: "Warehouse document ID", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Dokumenty magazynowe - przykłady wywołania"},
		{Path: "kind", Type: "string", Description: "Warehouse document kind", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Dodawanie dokumentu MM", EnumValues: []string{"mm", "pz", "wz"}},
		{Path: "number", Type: "string", Description: "Warehouse document number", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie dokumentu magazynowego po ID"},
		{Path: "warehouse_id", Type: "integer", Description: "Primary warehouse ID", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Dodawanie dokumentu MM"},
		{Path: "issue_date", Type: "string", Description: "Issue date", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Dodawanie dokumentu MM"},
		{Path: "department_name", Type: "string", Description: "Department name", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Dokumenty magazynowe - przykłady wywołania"},
		{Path: "department_id", Type: "integer", Description: "Department ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Dokumenty magazynowe - przykłady wywołania"},
		{Path: "client_name", Type: "string", Description: "Client name", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "conditional", SourceSection: "Dokumenty magazynowe - przykłady wywołania"},
		{Path: "client_id", Type: "integer", Description: "Client ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Dokumenty magazynowe - przykłady wywołania"},
	}
	if includeActions {
		fields = append(fields, OutputFieldSpec{Path: "warehouse_actions[]", Type: "array<object>", Description: "Warehouse action line items", Projectable: true, Selectable: false, Commands: commands, Presence: "conditional", SourceSection: "Dodawanie dokumentu MM"})
		fields = append(fields, warehouseActionLeafOutputFields("warehouse_actions[]", commands)...)
	}

	return &OutputSpec{
		Shape:      shape,
		OpenEnded:  true,
		PathSyntax: "dot_bracket",
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		DefaultColumns: []string{"id", "kind", "number", "issue_date", "warehouse_id", "client_name"},
		Notes: []string{
			"known_fields is curated from the upstream README and is not exhaustive",
			"unknown upstream fields may still appear in data and can still be selected when the path syntax is valid",
		},
		KnownFields: fields,
	}
}

func warehouseDocumentRequestFields() []RequestFieldSpec {
	fields := []RequestFieldSpec{
		{Path: "kind", Type: "string", Description: "Warehouse document kind", SourceSection: "Dodawanie dokumentu MM", EnumValues: []string{"mm", "pz", "wz"}},
		{Path: "warehouse_id", Type: "integer", Description: "Primary warehouse ID", SourceSection: "Dodawanie dokumentu MM"},
		{Path: "number", Type: "string", Description: "Warehouse document number", SourceSection: "Aktualizacja dokumentu magazynowego"},
		{Path: "issue_date", Type: "string", Description: "Issue date", SourceSection: "Dodawanie dokumentu MM"},
		{Path: "department_name", Type: "string", Description: "Department name", SourceSection: "Dodawanie dokumentu MM"},
		{Path: "department_id", Type: "integer", Description: "Department ID", SourceSection: "Aktualizacja dokumentu magazynowego"},
		{Path: "client_name", Type: "string", Description: "Client name", SourceSection: "Dodawanie dokumentu WZ"},
		{Path: "client_id", Type: "integer", Description: "Client ID", SourceSection: "Aktualizacja dokumentu magazynowego"},
		{Path: "invoice_ids[]", Type: "integer", Description: "Linked invoice IDs", SourceSection: "Dodawanie numerów powiązanych faktur do dokumentu magazynowego"},
		{Path: "warehouse_actions[]", Type: "array<object>", Description: "Warehouse action line items", SourceSection: "Dodawanie dokumentu MM"},
	}
	fields = append(fields, warehouseActionLeafRequestFields("warehouse_actions[]")...)
	return fields
}
