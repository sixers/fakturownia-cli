package spec

import (
	"fmt"

	"github.com/sixers/fakturownia-cli/internal/output"
)

const fakturowniaReadmeURL = "https://github.com/fakturownia/API/blob/master/README.md"

type CatalogBasis struct {
	Source string `json:"source"`
	URL    string `json:"url"`
}

type OutputFieldSpec struct {
	Path          string   `json:"path"`
	Type          string   `json:"type"`
	Description   string   `json:"description"`
	Projectable   bool     `json:"projectable"`
	Selectable    bool     `json:"selectable"`
	DefaultColumn bool     `json:"default_column,omitempty"`
	Commands      []string `json:"commands,omitempty"`
	Presence      string   `json:"presence,omitempty"`
	Requires      []string `json:"requires,omitempty"`
	SourceSection string   `json:"source_section,omitempty"`
	EnumValues    []string `json:"enum_values,omitempty"`
}

type OutputSpec struct {
	Shape          string            `json:"shape"`
	OpenEnded      bool              `json:"open_ended"`
	CatalogBasis   *CatalogBasis     `json:"catalog_basis,omitempty"`
	PathSyntax     string            `json:"path_syntax,omitempty"`
	KnownFields    []OutputFieldSpec `json:"known_fields,omitempty"`
	DefaultColumns []string          `json:"default_columns,omitempty"`
	Notes          []string          `json:"notes,omitempty"`
}

func invoiceListOutputSpec() *OutputSpec {
	spec := invoiceBaseOutputSpec("array", []string{"invoice list", "invoice get"})
	spec.KnownFields = append([]OutputFieldSpec{}, spec.KnownFields...)
	return spec
}

func invoiceGetOutputSpec(commands ...string) *OutputSpec {
	if len(commands) == 0 {
		commands = []string{"invoice get"}
	}
	return invoiceBaseOutputSpec("object", commands)
}

func invoiceBaseOutputSpec(shape string, commands []string) *OutputSpec {
	return &OutputSpec{
		Shape:      shape,
		OpenEnded:  true,
		PathSyntax: "dot_bracket",
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		DefaultColumns: []string{"id", "number", "issue_date", "buyer_name", "price_gross", "status"},
		Notes: []string{
			"known_fields is curated from the upstream README and is not exhaustive",
			"unknown upstream fields may still appear in data and can still be selected when the path syntax is valid",
		},
		KnownFields: []OutputFieldSpec{
			{Path: "id", Type: "integer", Description: "Invoice ID", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Faktury - przykłady wywołania"},
			{Path: "number", Type: "string", Description: "Invoice number", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Faktury - przykłady wywołania"},
			{Path: "token", Type: "string", Description: "Public invoice token used to derive view and PDF links", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Link do podglądu faktury i pobieranie do PDF"},
			{Path: "kind", Type: "string", Description: "Invoice kind", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU", EnumValues: []string{"vat", "proforma", "bill", "receipt", "advance", "final", "correction", "invoice_other", "vat_margin", "kp", "kw", "estimate", "vat_mp", "vat_rr", "correction_note", "accounting_note"}},
			{Path: "status", Type: "string", Description: "Invoice status", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU", EnumValues: []string{"issued", "sent", "paid", "partial", "rejected"}},
			{Path: "issue_date", Type: "string", Description: "Invoice issue date", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pobranie listy faktur z aktualnego miesiąca"},
			{Path: "sale_date", Type: "string", Description: "Invoice sale date", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Pobranie faktury po ID"},
			{Path: "payment_to", Type: "string", Description: "Payment due date", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU"},
			{Path: "payment_type", Type: "string", Description: "Payment type", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU", EnumValues: []string{"transfer", "card", "cash", "barter", "cheque", "bill_of_exchange", "cash_on_delivery", "compensation", "letter_of_credit", "payu", "paypal", "off"}},
			{Path: "currency", Type: "string", Description: "Invoice currency", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU"},
			{Path: "price_net", Type: "number", Description: "Net invoice total", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Pobranie faktury po ID"},
			{Path: "price_gross", Type: "number", Description: "Gross invoice total", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Pobranie listy faktur z aktualnego miesiąca"},
			{Path: "income", Type: "string", Description: "Income selector", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU", EnumValues: []string{"1", "0"}},
			{Path: "buyer_name", Type: "string", Description: "Buyer display name", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU"},
			{Path: "buyer_tax_no", Type: "string", Description: "Buyer tax number", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU"},
			{Path: "buyer_email", Type: "string", Description: "Buyer email address", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU"},
			{Path: "seller_name", Type: "string", Description: "Seller display name", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU"},
			{Path: "seller_tax_no", Type: "string", Description: "Seller tax number", Projectable: true, Selectable: true, Commands: commands, Presence: "common", SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU"},
			{Path: "invoice_id", Type: "integer", Description: "Linked source invoice ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Dodanie nowej faktury korygującej"},
			{Path: "from_invoice_id", Type: "integer", Description: "Source invoice or receipt ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Dodawanie faktury do istniejącego paragonu"},
			{Path: "correction_reason", Type: "string", Description: "Correction or cancellation reason", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Dodanie nowej faktury korygującej"},
			{Path: "cancel_reason", Type: "string", Description: "Cancellation reason returned when requested through additional_fields", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice get --additional-field cancel_reason"}, SourceSection: "Anulowanie faktury"},
			{Path: "corrected_content_before", Type: "string", Description: "Correction content before change", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice get --additional-field corrected_content_before"}, SourceSection: "Faktura korekta"},
			{Path: "corrected_content_after", Type: "string", Description: "Correction content after change", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice get --additional-field corrected_content_after"}, SourceSection: "Faktura korekta"},
			{Path: "connected_payments[]", Type: "array<object>", Description: "Payments connected to the invoice", Projectable: true, Selectable: false, Commands: commands, Presence: "conditional", Requires: []string{"invoice get --additional-field connected_payments"}, SourceSection: "Pobranie faktury razem z połączonymi płatnościami"},
			{Path: "description", Type: "string", Description: "First invoice note content", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Uwagi na fakturze (descriptions)"},
			{Path: "positions[]", Type: "array<object>", Description: "Invoice positions array", Projectable: true, Selectable: false, Commands: commands, Presence: "conditional", Requires: []string{"invoice list --include-positions"}, SourceSection: "Pobranie listy faktur wraz z ich pozycjami"},
			{Path: "positions[].id", Type: "integer", Description: "Invoice position ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes positions"}, SourceSection: "Aktualizacja pozycji na fakturze"},
			{Path: "positions[].product_id", Type: "integer", Description: "Referenced product ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes positions"}, SourceSection: "Dodanie nowej faktury - minimalna wersja"},
			{Path: "positions[].name", Type: "string", Description: "Invoice position name", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice list --include-positions"}, SourceSection: "Pobranie listy faktur wraz z ich pozycjami"},
			{Path: "positions[].quantity", Type: "number", Description: "Invoice position quantity", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice list --include-positions"}, SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU"},
			{Path: "positions[].tax", Type: "string", Description: "Invoice position tax rate", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice list --include-positions"}, SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU"},
			{Path: "positions[].price_net", Type: "number", Description: "Invoice position net price", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes positions"}, SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU"},
			{Path: "positions[].price_gross", Type: "number", Description: "Invoice position gross price", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes positions"}, SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU"},
			{Path: "positions[].total_price_net", Type: "number", Description: "Invoice position net total", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes positions"}, SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU"},
			{Path: "positions[].total_price_gross", Type: "number", Description: "Invoice position gross total", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice list --include-positions"}, SourceSection: "Pobranie listy faktur wraz z ich pozycjami"},
			{Path: "positions[].description", Type: "string", Description: "Invoice position description", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes positions"}, SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU"},
			{Path: "positions[].code", Type: "string", Description: "Invoice position product code", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes positions"}, SourceSection: "Faktury - specyfikacja, rodzaje pól, kody GTU"},
			{Path: "positions[].gtu_code", Type: "string", Description: "Invoice position GTU code", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes positions"}, SourceSection: "Faktury - kody GTU"},
			{Path: "positions[].correction_before_attributes", Type: "object", Description: "Correction values before the change", Projectable: true, Selectable: false, Commands: commands, Presence: "conditional", Requires: []string{"invoice correction positions"}, SourceSection: "Dodanie nowej faktury korygującej"},
			{Path: "positions[].correction_before_attributes.name", Type: "string", Description: "Corrected item name before change", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice correction positions"}, SourceSection: "Dodanie nowej faktury korygującej"},
			{Path: "positions[].correction_before_attributes.quantity", Type: "number", Description: "Corrected item quantity before change", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice correction positions"}, SourceSection: "Dodanie nowej faktury korygującej"},
			{Path: "positions[].correction_after_attributes", Type: "object", Description: "Correction values after the change", Projectable: true, Selectable: false, Commands: commands, Presence: "conditional", Requires: []string{"invoice correction positions"}, SourceSection: "Dodanie nowej faktury korygującej"},
			{Path: "positions[].correction_after_attributes.name", Type: "string", Description: "Corrected item name after change", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice correction positions"}, SourceSection: "Dodanie nowej faktury korygującej"},
			{Path: "positions[].correction_after_attributes.quantity", Type: "number", Description: "Corrected item quantity after change", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice correction positions"}, SourceSection: "Dodanie nowej faktury korygującej"},
			{Path: "descriptions[]", Type: "array<object>", Description: "Structured invoice notes", Projectable: true, Selectable: false, Commands: commands, Presence: "conditional", Requires: []string{"invoice get --include descriptions"}, SourceSection: "Uwagi na fakturze (descriptions)"},
			{Path: "descriptions[].id", Type: "integer", Description: "Structured note ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice get --include descriptions"}, SourceSection: "Uwagi na fakturze (descriptions)"},
			{Path: "descriptions[].kind", Type: "string", Description: "Structured note heading", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice get --include descriptions"}, SourceSection: "Uwagi na fakturze (descriptions)"},
			{Path: "descriptions[].content", Type: "string", Description: "Structured note content", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice get --include descriptions"}, SourceSection: "Uwagi na fakturze (descriptions)"},
			{Path: "descriptions[].position_index", Type: "integer|null", Description: "Structured note position index", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice get --include descriptions"}, SourceSection: "Uwagi na fakturze (descriptions)"},
			{Path: "descriptions[].row_number", Type: "integer", Description: "Structured note row number", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice get --include descriptions"}, SourceSection: "Uwagi na fakturze (descriptions)"},
			{Path: "recipients[]", Type: "array<object>", Description: "Invoice recipients array", Projectable: true, Selectable: false, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes recipients"}, SourceSection: "Odbiorcy/Wystawcy na fakturze"},
			{Path: "recipients[].id", Type: "integer", Description: "Recipient ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes recipients"}, SourceSection: "Odbiorcy/Wystawcy na fakturze"},
			{Path: "recipients[].name", Type: "string", Description: "Recipient name", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes recipients"}, SourceSection: "Odbiorcy/Wystawcy na fakturze"},
			{Path: "recipients[].first_name", Type: "string", Description: "Recipient first name", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes recipients"}, SourceSection: "Odbiorcy/Wystawcy na fakturze"},
			{Path: "recipients[].last_name", Type: "string", Description: "Recipient last name", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes recipients"}, SourceSection: "Odbiorcy/Wystawcy na fakturze"},
			{Path: "recipients[].company", Type: "boolean", Description: "Recipient company flag", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes recipients"}, SourceSection: "Odbiorcy/Wystawcy na fakturze"},
			{Path: "recipients[].country", Type: "string", Description: "Recipient country", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes recipients"}, SourceSection: "Odbiorcy/Wystawcy na fakturze"},
			{Path: "recipients[].city", Type: "string", Description: "Recipient city", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes recipients"}, SourceSection: "Odbiorcy/Wystawcy na fakturze"},
			{Path: "recipients[].post_code", Type: "string", Description: "Recipient postal code", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes recipients"}, SourceSection: "Odbiorcy/Wystawcy na fakturze"},
			{Path: "recipients[].street", Type: "string", Description: "Recipient street", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes recipients"}, SourceSection: "Odbiorcy/Wystawcy na fakturze"},
			{Path: "recipients[].phone", Type: "string", Description: "Recipient phone number", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes recipients"}, SourceSection: "Odbiorcy/Wystawcy na fakturze"},
			{Path: "recipients[].email", Type: "string", Description: "Recipient email address", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes recipients"}, SourceSection: "Odbiorcy/Wystawcy na fakturze"},
			{Path: "recipients[].note", Type: "string", Description: "Recipient note", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes recipients"}, SourceSection: "Odbiorcy/Wystawcy na fakturze"},
			{Path: "recipients[].role", Type: "string", Description: "Recipient role", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes recipients"}, SourceSection: "Odbiorcy/Wystawcy na fakturze"},
			{Path: "issuers[]", Type: "array<object>", Description: "Invoice issuers array", Projectable: true, Selectable: false, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes issuers"}, SourceSection: "Odbiorcy/Wystawcy na fakturze"},
			{Path: "issuers[].id", Type: "integer", Description: "Issuer ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes issuers"}, SourceSection: "Odbiorcy/Wystawcy na fakturze"},
			{Path: "issuers[].name", Type: "string", Description: "Issuer name", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes issuers"}, SourceSection: "Odbiorcy/Wystawcy na fakturze"},
			{Path: "issuers[].email", Type: "string", Description: "Issuer email address", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes issuers"}, SourceSection: "Odbiorcy/Wystawcy na fakturze"},
			{Path: "settlement_positions[]", Type: "array<object>", Description: "Invoice settlement positions", Projectable: true, Selectable: false, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes settlement_positions"}, SourceSection: "Rozliczenia na fakturze (settlement_positions)"},
			{Path: "settlement_positions[].id", Type: "integer", Description: "Settlement position ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes settlement_positions"}, SourceSection: "Rozliczenia na fakturze (settlement_positions)"},
			{Path: "settlement_positions[].kind", Type: "string", Description: "Settlement position kind", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes settlement_positions"}, SourceSection: "Rozliczenia na fakturze (settlement_positions)"},
			{Path: "settlement_positions[].amount", Type: "number", Description: "Settlement position amount", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes settlement_positions"}, SourceSection: "Rozliczenia na fakturze (settlement_positions)"},
			{Path: "settlement_positions[].reason", Type: "string", Description: "Settlement position reason", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", Requires: []string{"invoice includes settlement_positions"}, SourceSection: "Rozliczenia na fakturze (settlement_positions)"},
		},
	}
}

func validateOutputSelection(spec CommandSpec, fields, columns []string) ([]output.WarningDetail, *output.AppError) {
	if len(fields) > 0 {
		if _, err := output.ParsePaths(fields); err != nil {
			return nil, output.Usage("invalid_fields", fmt.Sprintf("invalid --fields path: %v", err), "use dot/bracket paths like number,status or positions[].name")
		}
	}
	if len(columns) > 0 {
		if _, err := output.ParsePaths(columns); err != nil {
			return nil, output.Usage("invalid_columns", fmt.Sprintf("invalid --columns path: %v", err), "use dot/bracket paths like number or positions[].name")
		}
	}
	if spec.Output == nil {
		return nil, nil
	}

	known := make(map[string]OutputFieldSpec, len(spec.Output.KnownFields))
	for _, field := range spec.Output.KnownFields {
		known[field.Path] = field
	}

	warnings := make([]output.WarningDetail, 0)
	for _, fieldPath := range fields {
		field, ok := known[fieldPath]
		if !ok {
			warnings = append(warnings, output.WarningDetail{
				Code:    "undocumented_field_path",
				Message: fmt.Sprintf("field path %q is not in the README-backed known_fields catalog for %s %s", fieldPath, spec.Noun, spec.Verb),
			})
			continue
		}
		if !field.Projectable {
			return nil, output.Usage("unprojectable_field", fmt.Sprintf("field path %q is not projectable", fieldPath), "inspect `fakturownia schema "+spec.Noun+" "+spec.Verb+" --json` for projectable known_fields")
		}
	}
	for _, columnPath := range columns {
		field, ok := known[columnPath]
		if !ok {
			warnings = append(warnings, output.WarningDetail{
				Code:    "undocumented_column_path",
				Message: fmt.Sprintf("column path %q is not in the README-backed known_fields catalog for %s %s", columnPath, spec.Noun, spec.Verb),
			})
			continue
		}
		if !field.Selectable {
			return nil, output.Usage("unselectable_column", fmt.Sprintf("column path %q is not selectable for table output", columnPath), "choose a scalar path such as number or positions[].name")
		}
	}

	return warnings, nil
}

func buildOutputDataSchema(spec *OutputSpec) (map[string]any, error) {
	if spec == nil {
		return nil, nil
	}
	objectSchema := newOpenObjectSchema()
	for _, field := range spec.KnownFields {
		path, err := output.ParsePath(field.Path)
		if err != nil {
			return nil, err
		}
		applyFieldSchema(objectSchema, path.Segments, field)
	}
	switch spec.Shape {
	case "object":
		return objectSchema, nil
	case "array":
		return map[string]any{
			"type":  "array",
			"items": objectSchema,
		}, nil
	default:
		return nil, nil
	}
}

func applyFieldSchema(root map[string]any, segments []output.PathSegment, field OutputFieldSpec) {
	if len(segments) == 0 {
		return
	}

	segment := segments[0]
	if len(segments) == 1 {
		setPropertySchema(root, segment, field)
		return
	}

	var child map[string]any
	if segment.Array {
		child = ensureArrayObjectProperty(root, segment.Name)
	} else {
		child = ensureObjectProperty(root, segment.Name)
	}
	applyFieldSchema(child, segments[1:], field)
}

func setPropertySchema(root map[string]any, segment output.PathSegment, field OutputFieldSpec) {
	properties := ensureProperties(root)
	schema := schemaForField(field)
	if segment.Array && schemaType(schema) != "array" {
		schema = map[string]any{
			"type":  "array",
			"items": schema,
		}
	}
	if existing, ok := properties[segment.Name].(map[string]any); ok {
		properties[segment.Name] = mergeSchemaMaps(existing, schema)
		return
	}
	properties[segment.Name] = schema
}

func ensureObjectProperty(root map[string]any, name string) map[string]any {
	properties := ensureProperties(root)
	if existing, ok := properties[name].(map[string]any); ok {
		if schemaType(existing) == "object" {
			existing["additionalProperties"] = true
			ensureProperties(existing)
			return existing
		}
	}
	child := newOpenObjectSchema()
	properties[name] = child
	return child
}

func ensureArrayObjectProperty(root map[string]any, name string) map[string]any {
	properties := ensureProperties(root)
	if existing, ok := properties[name].(map[string]any); ok && schemaType(existing) == "array" {
		if items, ok := existing["items"].(map[string]any); ok {
			if schemaType(items) == "object" {
				items["additionalProperties"] = true
				ensureProperties(items)
				return items
			}
		}
	}

	items := newOpenObjectSchema()
	properties[name] = map[string]any{
		"type":  "array",
		"items": items,
	}
	return items
}

func ensureProperties(schema map[string]any) map[string]any {
	if properties, ok := schema["properties"].(map[string]any); ok {
		return properties
	}
	properties := map[string]any{}
	schema["properties"] = properties
	return properties
}

func newOpenObjectSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": true,
		"properties":           map[string]any{},
	}
}

func schemaForField(field OutputFieldSpec) map[string]any {
	var schema map[string]any
	switch field.Type {
	case "string":
		schema = map[string]any{"type": "string"}
	case "integer":
		schema = map[string]any{"type": "integer"}
	case "number":
		schema = map[string]any{"type": "number"}
	case "boolean":
		schema = map[string]any{"type": "boolean"}
	case "object":
		schema = newOpenObjectSchema()
	case "array<object>":
		schema = map[string]any{
			"type":  "array",
			"items": newOpenObjectSchema(),
		}
	case "integer|null":
		schema = map[string]any{"type": []any{"integer", "null"}}
	default:
		schema = map[string]any{"type": "string"}
	}
	if len(field.EnumValues) > 0 {
		schema["enum"] = append([]string{}, field.EnumValues...)
	}
	return schema
}

func mergeSchemaMaps(left, right map[string]any) map[string]any {
	merged := cloneSchemaMap(left)
	for key, value := range right {
		existing, ok := merged[key]
		if !ok {
			merged[key] = cloneSchemaValue(value)
			continue
		}
		existingMap, existingOK := existing.(map[string]any)
		valueMap, valueOK := value.(map[string]any)
		if existingOK && valueOK {
			merged[key] = mergeSchemaMaps(existingMap, valueMap)
			continue
		}
		merged[key] = cloneSchemaValue(value)
	}
	return merged
}

func cloneSchemaMap(value map[string]any) map[string]any {
	out := make(map[string]any, len(value))
	for key, child := range value {
		out[key] = cloneSchemaValue(child)
	}
	return out
}

func cloneSchemaValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneSchemaMap(typed)
	case []any:
		out := make([]any, len(typed))
		for idx, child := range typed {
			out[idx] = cloneSchemaValue(child)
		}
		return out
	case []string:
		out := make([]string, len(typed))
		copy(out, typed)
		return out
	default:
		return typed
	}
}

func schemaType(schema map[string]any) string {
	typed, _ := schema["type"].(string)
	return typed
}
