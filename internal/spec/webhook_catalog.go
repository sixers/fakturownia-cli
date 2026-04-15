package spec

var webhookKindValues = []string{
	"client:create",
	"client:update",
	"client:destroy",
	"invoice:create",
	"invoice:update",
	"invoice:destroy",
	"product:create",
	"product:update",
	"product:destroy",
}

func webhookListOutputSpec() *OutputSpec {
	return webhookBaseOutputSpec("array", []string{"webhook list"})
}

func webhookGetOutputSpec(commands ...string) *OutputSpec {
	if len(commands) == 0 {
		commands = []string{"webhook get", "webhook create", "webhook update"}
	}
	return webhookBaseOutputSpec("object", commands)
}

func webhookRequestBodySpec() *RequestBodySpec {
	return &RequestBodySpec{
		InputFlag:  "input",
		InputModes: []string{"inline_json", "@file", "stdin"},
		OpenEnded:  true,
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		PathSyntax: "dot_bracket",
		KnownFields: []RequestFieldSpec{
			{Path: "kind", Type: "string", Description: "Webhook event kind", EnumValues: webhookKindValues, SourceSection: "Webhooki"},
			{Path: "url", Type: "string", Description: "Webhook target URL", SourceSection: "Webhooki"},
			{Path: "api_token", Type: "string", Description: "Webhook token shared with the receiver", SourceSection: "Webhooki"},
			{Path: "active", Type: "boolean", Description: "Whether the webhook is active", SourceSection: "Webhooki"},
		},
		Notes: []string{
			"webhook create and update accept the full top-level request object because the upstream README documents endpoints but not a wrapper key",
			"account_id is not required in the CLI request schema",
			"known_fields is curated from the upstream README and user-verified behavior and is not exhaustive",
		},
	}
}

func webhookBaseOutputSpec(shape string, commands []string) *OutputSpec {
	return &OutputSpec{
		Shape:      shape,
		OpenEnded:  true,
		PathSyntax: "dot_bracket",
		CatalogBasis: &CatalogBasis{
			Source: "readme",
			URL:    fakturowniaReadmeURL,
		},
		DefaultColumns: []string{"id", "kind", "url", "active"},
		Notes: []string{
			"known_fields is curated from the upstream README and user-verified behavior and is not exhaustive",
			"unknown upstream fields may still appear in data and can still be selected when the path syntax is valid",
		},
		KnownFields: []OutputFieldSpec{
			{Path: "id", Type: "integer", Description: "Webhook ID", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Webhooki"},
			{Path: "kind", Type: "string", Description: "Webhook event kind", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", EnumValues: webhookKindValues, SourceSection: "Webhooki"},
			{Path: "account_id", Type: "integer", Description: "Owning account ID", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Webhooki"},
			{Path: "url", Type: "string", Description: "Webhook target URL", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "common", SourceSection: "Webhooki"},
			{Path: "api_token", Type: "string", Description: "Webhook token shared with the receiver", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Webhooki"},
			{Path: "created_at", Type: "string", Description: "Creation timestamp", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Webhooki"},
			{Path: "updated_at", Type: "string", Description: "Last update timestamp", Projectable: true, Selectable: true, Commands: commands, Presence: "conditional", SourceSection: "Webhooki"},
			{Path: "active", Type: "boolean", Description: "Whether the webhook is active", Projectable: true, Selectable: true, DefaultColumn: true, Commands: commands, Presence: "conditional", SourceSection: "Webhooki"},
		},
	}
}
