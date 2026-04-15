package spec

func priceListListOutputSpec() *OutputSpec {
	return priceListBaseOutputSpec("array", []string{"price-list list"}, false)
}

func priceListGetOutputSpec(commands ...string) *OutputSpec {
	if len(commands) == 0 {
		commands = []string{"price-list get", "price-list create", "price-list update"}
	}
	return priceListBaseOutputSpec("object", commands, true)
}

func priceListRequestBodySpec() *RequestBodySpec {
	return &RequestBodySpec{
		InputFlag:  "input",
		InputModes: []string{"inline_json", "@file", "stdin"},
		WrapperKey: "price_list",
		OpenEnded:  true,
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		PathSyntax:  "dot_bracket",
		KnownFields: priceListRequestFields(),
		Notes: []string{
			"the CLI accepts the inner price list object, then wraps it in the upstream {\"price_list\": ...} envelope",
			"known_fields is curated from the upstream README and verified API behavior and is not exhaustive",
			"price_list_positions_attributes is an open object keyed by upstream string indexes like \"0\" and \"1\"",
		},
	}
}

func priceListBaseOutputSpec(shape string, commands []string, includePositions bool) *OutputSpec {
	fields := []OutputFieldSpec{
		{Path: "id", Type: "integer", Description: "Price list ID", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pobranie listy cenników"},
		{Path: "name", Type: "string", Description: "Price list name", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Dodanie cennika"},
		{Path: "description", Type: "string", Description: "Price list description", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "conditional", SourceSection: "Dodanie cennika"},
		{Path: "currency", Type: "string", Description: "Price list currency", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Dodanie cennika"},
		{Path: "deleted", Type: "boolean", Description: "Deleted flag", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie cennika po ID"},
		{Path: "account_id", Type: "integer", Description: "Owning account ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie cennika po ID"},
		{Path: "created_at", Type: "string", Description: "Creation timestamp", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie cennika po ID"},
		{Path: "updated_at", Type: "string", Description: "Last update timestamp", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie cennika po ID"},
	}
	if includePositions {
		fields = append(fields,
			OutputFieldSpec{Path: "price_list_positions[]", Type: "array<object>", Description: "Price list position entries", Projectable: true, Selectable: false, Commands: commands, Presence: "conditional", SourceSection: "Pobranie cennika po ID"},
			OutputFieldSpec{Path: "price_list_positions[].id", Type: "integer", Description: "Price list position ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie cennika po ID"},
			OutputFieldSpec{Path: "price_list_positions[].priceable_id", Type: "integer", Description: "Referenced priceable ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie cennika po ID"},
			OutputFieldSpec{Path: "price_list_positions[].priceable_type", Type: "string", Description: "Referenced priceable type", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie cennika po ID"},
			OutputFieldSpec{Path: "price_list_positions[].price_list_id", Type: "integer", Description: "Owning price list ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie cennika po ID"},
			OutputFieldSpec{Path: "price_list_positions[].use_percentage", Type: "boolean", Description: "Use percentage pricing flag", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie cennika po ID"},
			OutputFieldSpec{Path: "price_list_positions[].percentage", Type: "number|null", Description: "Percentage adjustment", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie cennika po ID"},
			OutputFieldSpec{Path: "price_list_positions[].price_net", Type: "number", Description: "Net price", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie cennika po ID"},
			OutputFieldSpec{Path: "price_list_positions[].price_gross", Type: "number", Description: "Gross price", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie cennika po ID"},
			OutputFieldSpec{Path: "price_list_positions[].tax", Type: "string", Description: "Tax rate", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie cennika po ID"},
			OutputFieldSpec{Path: "price_list_positions[].deleted", Type: "boolean", Description: "Deleted flag for the position", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie cennika po ID"},
			OutputFieldSpec{Path: "price_list_positions[].account_id", Type: "integer", Description: "Owning account ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie cennika po ID"},
			OutputFieldSpec{Path: "price_list_positions[].created_at", Type: "string", Description: "Creation timestamp", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie cennika po ID"},
			OutputFieldSpec{Path: "price_list_positions[].updated_at", Type: "string", Description: "Last update timestamp", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Pobranie cennika po ID"},
		)
	}

	notes := []string{
		"known_fields is curated from the upstream README and verified API behavior and is not exhaustive",
		"unknown upstream fields may still appear in data and can still be selected when the path syntax is valid",
	}
	if includePositions {
		notes = append(notes, "price-list get is included because /price_lists/:id.json has been verified against live API behavior even though the current README price_lists section does not document it explicitly")
	}

	return &OutputSpec{
		Shape:      shape,
		OpenEnded:  true,
		PathSyntax: "dot_bracket",
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		DefaultColumns: []string{"id", "name", "currency", "description"},
		Notes:          notes,
		KnownFields:    fields,
	}
}

func priceListRequestFields() []RequestFieldSpec {
	return []RequestFieldSpec{
		{Path: "name", Type: "string", Description: "Price list name", SourceSection: "Dodanie cennika"},
		{Path: "description", Type: "string", Description: "Price list description", SourceSection: "Dodanie cennika"},
		{Path: "currency", Type: "string", Description: "Price list currency", SourceSection: "Dodanie cennika"},
		{
			Path:          "price_list_positions_attributes",
			Type:          "object",
			Description:   "Open object keyed by upstream string indexes like \"0\" and \"1\", where each value describes one price list position entry",
			SourceSection: "Dodanie cennika",
			SchemaOverride: map[string]any{
				"type":                 "object",
				"additionalProperties": priceListPositionAttributeSchema(),
			},
		},
	}
}

func priceListPositionAttributeSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": true,
		"properties": map[string]any{
			"id":             map[string]any{"type": "integer", "description": "Existing price list position ID"},
			"priceable_id":   map[string]any{"type": "integer", "description": "Referenced priceable ID"},
			"priceable_name": map[string]any{"type": "string", "description": "Referenced priceable name"},
			"priceable_type": map[string]any{"type": "string", "description": "Referenced priceable type"},
			"use_percentage": map[string]any{"type": "boolean", "description": "Use percentage pricing flag"},
			"percentage":     map[string]any{"type": "number", "description": "Percentage adjustment"},
			"price_net":      map[string]any{"type": "number", "description": "Net price"},
			"price_gross":    map[string]any{"type": "number", "description": "Gross price"},
			"use_tax":        map[string]any{"type": "boolean", "description": "Use explicit tax flag"},
			"tax":            map[string]any{"type": "string", "description": "Tax rate"},
		},
	}
}
