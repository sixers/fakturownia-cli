package spec

import "path/filepath"

type RecipeSpec struct {
	Key           string
	Name          string
	Title         string
	Description   string
	AreaKey       string
	RelatedSkills []string
	CommandRefs   []SkillCommandRef
	Markdown      string
}

func Recipes() []RecipeSpec {
	return []RecipeSpec{
		{
			Key:           "invoice-minimal",
			Name:          "fakturownia-invoice-minimal",
			Title:         "Minimal Invoice From Existing IDs",
			Description:   "Create a minimal invoice when you already know the client and product IDs.",
			AreaKey:       "invoices",
			RelatedSkills: []string{"shared", "invoices", "products", "clients", "schema"},
			CommandRefs:   []SkillCommandRef{{Noun: "invoice", Verb: "create"}},
			Markdown:      "Use this when you already know the `client_id`, `product_id`, and optionally the seller department/recipient IDs.\n\n## Command\n\n```bash\nfakturownia invoice create --input '{\"payment_to_kind\":5,\"client_id\":1,\"positions\":[{\"product_id\":1,\"quantity\":2}]}' --json\n```\n\n## Notes\n\n- This mirrors the README's minimal invoice example.\n- The upstream API will default the issue date to today and the payment term to the `payment_to_kind` day count.\n- Inspect `fakturownia schema invoice create --json` before building automated payloads.\n",
		},
		{
			Key:           "invoice-copy",
			Name:          "fakturownia-invoice-copy",
			Title:         "Similar Invoice From Another Document",
			Description:   "Create a new invoice by copying another invoice, order, or proforma with `copy_invoice_from`.",
			AreaKey:       "invoices",
			RelatedSkills: []string{"shared", "invoices", "schema"},
			CommandRefs:   []SkillCommandRef{{Noun: "invoice", Verb: "create"}},
			Markdown:      "Use this when the new document should inherit positions and core fields from an existing source document.\n\n## Command\n\n```bash\nfakturownia invoice create --input '{\"copy_invoice_from\":42,\"kind\":\"vat\"}' --json\n```\n\n## Variants\n\n- Advance invoice from an order: set `kind` to `advance`, plus `advance_creation_mode`, `advance_value`, and `position_name`.\n- Final invoice from an order and advances: set `kind` to `final` and include `invoice_ids`.\n",
		},
		{
			Key:           "invoice-correction",
			Name:          "fakturownia-invoice-correction",
			Title:         "Correction Invoice",
			Description:   "Create and inspect correction invoices, including before/after correction fields.",
			AreaKey:       "invoices",
			RelatedSkills: []string{"shared", "invoices", "schema"},
			CommandRefs:   []SkillCommandRef{{Noun: "invoice", Verb: "get"}, {Noun: "invoice", Verb: "create"}},
			Markdown:      "Use this when you need a correction invoice or when you need the extra correction fields from an existing correction.\n\n## Inspect an Existing Correction\n\n```bash\nfakturownia invoice get --id 2432393 \\\n  --additional-field corrected_content_before \\\n  --additional-field corrected_content_after \\\n  --correction-positions full \\\n  --json\n```\n\n## Create a Correction\n\n```bash\nfakturownia invoice create --input '{\n  \"kind\":\"correction\",\n  \"correction_reason\":\"Zła ilość\",\n  \"invoice_id\":2432393,\n  \"from_invoice_id\":2432393,\n  \"client_id\":1,\n  \"positions\":[{\n    \"name\":\"Product A1\",\n    \"quantity\":-1,\n    \"total_price_gross\":\"-10\",\n    \"tax\":\"23\",\n    \"kind\":\"correction\",\n    \"correction_before_attributes\":{\"name\":\"Product A1\",\"quantity\":\"2\",\"total_price_gross\":\"20\",\"tax\":\"23\",\"kind\":\"correction_before\"},\n    \"correction_after_attributes\":{\"name\":\"Product A2\",\"quantity\":\"1\",\"total_price_gross\":\"10\",\"tax\":\"23\",\"kind\":\"correction_after\"}\n  }]\n}' --json\n```\n",
		},
		{
			Key:           "invoice-oss",
			Name:          "fakturownia-invoice-oss",
			Title:         "OSS Invoice With Validation",
			Description:   "Create an OSS invoice and ask the API to validate the OSS conditions before marking it.",
			AreaKey:       "invoices",
			RelatedSkills: []string{"shared", "invoices", "schema"},
			CommandRefs:   []SkillCommandRef{{Noun: "invoice", Verb: "create"}},
			Markdown:      "Use this when an invoice should be marked as OSS and you want the server to validate the country rules before doing so.\n\n## Command\n\n```bash\nfakturownia invoice create \\\n  --identify-oss \\\n  --input '{\n    \"kind\":\"vat\",\n    \"seller_name\":\"Wystawca Sp. z o.o.\",\n    \"seller_country\":\"PL\",\n    \"buyer_name\":\"Klient1 Sp. z o.o.\",\n    \"buyer_country\":\"FR\",\n    \"use_oss\":true,\n    \"positions\":[{\"name\":\"Produkt A1\",\"tax\":20,\"total_price_gross\":50,\"quantity\":3}]\n  }' \\\n  --json\n```\n",
		},
		{
			Key:           "invoice-send-email",
			Name:          "fakturownia-invoice-send-email",
			Title:         "Send Invoice By Email",
			Description:   "Send an invoice email, optionally overriding recipients and attaching the PDF.",
			AreaKey:       "invoices",
			RelatedSkills: []string{"shared", "invoices"},
			CommandRefs:   []SkillCommandRef{{Noun: "invoice", Verb: "send-email"}},
			Markdown:      "Use this when the invoice already exists and you want Fakturownia to send the email.\n\n## Commands\n\n```bash\nfakturownia invoice send-email --id 100 --json\nfakturownia invoice send-email --id 100 --email-to billing@example.com --email-pdf --json\nfakturownia invoice send-email --id 100 --email-to billing@example.com --update-buyer-email --print-option original --dry-run --json\n```\n",
		},
		{
			Key:           "invoice-cancel",
			Name:          "fakturownia-invoice-cancel",
			Title:         "Cancel Invoice",
			Description:   "Cancel an invoice and optionally store a cancellation reason.",
			AreaKey:       "invoices",
			RelatedSkills: []string{"shared", "invoices"},
			CommandRefs:   []SkillCommandRef{{Noun: "invoice", Verb: "cancel"}, {Noun: "invoice", Verb: "get"}},
			Markdown:      "Use this when the invoice should be cancelled rather than edited.\n\n## Cancel\n\n```bash\nfakturownia invoice cancel --id 111 --yes --reason 'Powód anulowania' --json\n```\n\n## Read Back the Stored Reason\n\n```bash\nfakturownia invoice get --id 111 --additional-field cancel_reason --json\n```\n",
		},
		{
			Key:           "invoice-recipients-issuers",
			Name:          "fakturownia-invoice-recipients-issuers",
			Title:         "Recipients And Issuers",
			Description:   "Create or update invoice recipients and issuers through the generic invoice payload.",
			AreaKey:       "invoices",
			RelatedSkills: []string{"shared", "invoices", "schema"},
			CommandRefs:   []SkillCommandRef{{Noun: "invoice", Verb: "create"}, {Noun: "invoice", Verb: "update"}},
			Markdown:      "Use this when an invoice needs extra recipients or issuers from the README's recipient/issuer model.\n\n## Create With Recipients And Issuers\n\n```bash\nfakturownia invoice create --input '{\n  \"issue_date\":\"2024-08-01\",\n  \"seller_name\":\"Wystawca Sp. z o.o.\",\n  \"buyer_name\":\"Klient1 Sp. z o.o.\",\n  \"positions\":[{\"name\":\"Produkt A1\",\"tax\":23,\"total_price_gross\":10.23,\"quantity\":1}],\n  \"recipients\":[{\"name\":\"Odbiorca1\",\"company\":true,\"email\":\"odbiorca1@email.pl\"}],\n  \"issuers\":[{\"name\":\"Wystawca1\",\"company\":true,\"email\":\"wystawca1@email.pl\"}]\n}' --json\n```\n\n## Update Or Delete a Recipient\n\n```bash\nfakturownia invoice update --id 111 --input '{\"recipients\":[{\"id\":1,\"name\":\"Nowa nazwa\"}]}' --json\nfakturownia invoice update --id 111 --input '{\"recipients\":[{\"id\":1,\"_destroy\":1}]}' --json\n```\n",
		},
		{
			Key:           "invoice-attachment",
			Name:          "fakturownia-invoice-attachment",
			Title:         "Attach A File To An Invoice",
			Description:   "Upload a file through the attachment credentials flow and bind it to an invoice.",
			AreaKey:       "invoices",
			RelatedSkills: []string{"shared", "invoices"},
			CommandRefs:   []SkillCommandRef{{Noun: "invoice", Verb: "add-attachment"}, {Noun: "invoice", Verb: "download-attachments"}, {Noun: "invoice", Verb: "update"}},
			Markdown:      "Use this when an invoice needs a new attachment or when you need to fetch all attachments as a ZIP.\n\n## Attach a Local File\n\n```bash\nfakturownia invoice add-attachment --id 111 --file ./scan.pdf --json\n```\n\n## Attach Bytes From Stdin\n\n```bash\ncat ./scan.pdf | fakturownia invoice add-attachment --id 111 --file - --name scan.pdf --dry-run --json\n```\n\n## Make Attachments Visible To Customers\n\n```bash\nfakturownia invoice update --id 111 --input '{\"show_attachments\":true}' --json\n```\n\n## Download All Attachments\n\n```bash\nfakturownia invoice download-attachments --id 111 --dir ./attachments --json\n```\n",
		},
		{
			Key:           "invoice-fiscal-print",
			Name:          "fakturownia-invoice-fiscal-print",
			Title:         "Fiscal Print Batch",
			Description:   "Submit one or more invoices to the fiscal print endpoint, optionally targeting a printer by name.",
			AreaKey:       "invoices",
			RelatedSkills: []string{"shared", "invoices"},
			CommandRefs:   []SkillCommandRef{{Noun: "invoice", Verb: "fiscal-print"}},
			Markdown:      "Use this when invoices should be sent to the fiscal printer integration.\n\n## Commands\n\n```bash\nfakturownia invoice fiscal-print --invoice-id 111 --invoice-id 112 --json\nfakturownia invoice fiscal-print --invoice-id 111 --printer DRUKARKA --dry-run --json\n```\n",
		},
		{
			Key:           "recurring-definition",
			Name:          "fakturownia-recurring-definition",
			Title:         "Create Or Update A Recurring Definition",
			Description:   "Manage recurring invoice definitions through the dedicated recurring noun.",
			AreaKey:       "recurrings",
			RelatedSkills: []string{"shared", "recurrings", "schema"},
			CommandRefs:   []SkillCommandRef{{Noun: "recurring", Verb: "list"}, {Noun: "recurring", Verb: "create"}, {Noun: "recurring", Verb: "update"}},
			Markdown:      "Use this when the task is about the recurring definition itself rather than a concrete invoice instance.\n\n## Create\n\n```bash\nfakturownia recurring create --input '{\n  \"name\":\"Nazwa cyklicznosci\",\n  \"invoice_id\":1,\n  \"start_date\":\"2016-01-01\",\n  \"every\":\"1m\",\n  \"issue_working_day_only\":false,\n  \"send_email\":true,\n  \"buyer_email\":\"mail1@mail.pl, mail2@mail.pl\",\n  \"end_date\":\"null\"\n}' --json\n```\n\n## Update Next Invoice Date\n\n```bash\nfakturownia recurring update --id 111 --input '{\"next_invoice_date\":\"2016-02-01\"}' --json\n```\n",
		},
		{
			Key:           "invoice-receipt-link",
			Name:          "fakturownia-invoice-receipt-link",
			Title:         "Create Invoice From Receipt Or Link Existing Invoice",
			Description:   "Link a receipt to an existing invoice or create a new invoice from a receipt using the README-backed payload fields.",
			AreaKey:       "invoices",
			RelatedSkills: []string{"shared", "invoices", "schema"},
			CommandRefs:   []SkillCommandRef{{Noun: "invoice", Verb: "update"}, {Noun: "invoice", Verb: "create"}},
			Markdown:      "Use this when invoices and receipts need to be linked explicitly.\n\n## Link an Existing Invoice to a Receipt\n\n```bash\nfakturownia invoice update --id 111 --input '{\n  \"from_invoice_id\":222,\n  \"invoice_id\":222,\n  \"exclude_from_stock_level\":true\n}' --json\n```\n\n## Create a New Invoice From a Receipt\n\n```bash\nfakturownia invoice create --input '{\n  \"from_invoice_id\":222,\n  \"additional_params\":\"for_receipt\",\n  \"exclude_from_stock_level\":true,\n  \"buyer_name\":\"Klient1 Sp. z o.o.\",\n  \"buyer_tax_no\":\"6272616681\",\n  \"positions\":[{\"name\":\"Produkt A1\",\"quantity\":\"1\",\"tax\":\"23\",\"total_price_gross\":\"10,23\"}]\n}' --json\n```\n",
		},
	}
}

func recipePath(recipe RecipeSpec) string {
	return filepath.ToSlash(filepath.Join("skills", "fakturownia", "recipes", recipe.Key, "SKILL.md"))
}

func recipeIndexPath() string {
	return filepath.ToSlash(filepath.Join("skills", "fakturownia", "recipes", "index.md"))
}
