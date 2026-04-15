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
			Key:           "invoice-ksef-create-send",
			Name:          "fakturownia-invoice-ksef-create-send",
			Title:         "Create And Immediately Send To KSeF",
			Description:   "Create an invoice with the API-native `gov` companion flag that queues the document for KSeF submission right away.",
			AreaKey:       "invoices",
			RelatedSkills: []string{"shared", "invoices", "schema"},
			CommandRefs:   []SkillCommandRef{{Noun: "invoice", Verb: "create"}, {Noun: "invoice", Verb: "update"}},
			Markdown:      "Use this when the invoice should go to KSeF as part of the same create or update call.\n\n`gov` is the upstream API name for the KSeF integration surface.\n\n## Create And Queue For KSeF\n\n```bash\nfakturownia invoice create \\\n  --gov-save-and-send \\\n  --input '{\n    \"kind\":\"vat\",\n    \"buyer_company\":true,\n    \"seller_tax_no\":\"5252445767\",\n    \"seller_street\":\"ul. Przykładowa 10\",\n    \"seller_post_code\":\"00-001\",\n    \"seller_city\":\"Warszawa\",\n    \"buyer_name\":\"Klient ABC Sp. z o.o.\",\n    \"buyer_tax_no\":\"9876543210\",\n    \"positions\":[{\"name\":\"Usługa\",\"quantity\":1,\"total_price_gross\":1230,\"tax\":23}]\n  }' \\\n  --json\n```\n\n## Update And Requeue For KSeF\n\n```bash\nfakturownia invoice update --id 111 --gov-save-and-send --input '{\"buyer_tax_no_kind\":\"nip_ue\",\"buyer_tax_no\":\"DE123456789\"}' --json\n```\n\n## Notes\n\n- `--gov-save-and-send` is a top-level companion flag outside the inner invoice object.\n- Inspect `fakturownia schema invoice create --json` before automating KSeF payload generation.\n",
		},
		{
			Key:           "invoice-ksef-send-status",
			Name:          "fakturownia-invoice-ksef-send-status",
			Title:         "Send Existing Invoice To KSeF And Inspect Status",
			Description:   "Queue an already-created invoice for KSeF submission and read the `gov_*` status fields that the API returns.",
			AreaKey:       "invoices",
			RelatedSkills: []string{"shared", "invoices", "schema"},
			CommandRefs:   []SkillCommandRef{{Noun: "invoice", Verb: "send-gov"}, {Noun: "invoice", Verb: "get"}},
			Markdown:      "Use this when an invoice already exists and you want to send it to KSeF separately from create or update.\n\n`send-gov` is the API-native CLI name; it means “send the invoice to KSeF”.\n\n## Queue The Invoice For KSeF\n\n```bash\nfakturownia invoice send-gov --id 111 --json\n```\n\n## Inspect KSeF Status Fields\n\n```bash\nfakturownia invoice get --id 111 --fields id,number,gov_status,gov_id,gov_send_date,gov_error_messages[] --json\n```\n\n## Notes\n\n- `gov_status` and related fields are the KSeF integration fields exposed by the upstream API.\n- If the API reports send or validation errors, inspect `gov_error_messages[]` and retry after fixing the payload.\n",
		},
		{
			Key:           "invoice-ksef-download-documents",
			Name:          "fakturownia-invoice-ksef-download-documents",
			Title:         "Download KSeF XML And UPO",
			Description:   "Download KSeF XML documents through the generic invoice attachment endpoint using API-native `gov` kinds.",
			AreaKey:       "invoices",
			RelatedSkills: []string{"shared", "invoices"},
			CommandRefs:   []SkillCommandRef{{Noun: "invoice", Verb: "download-attachment"}},
			Markdown:      "Use this when you need the KSeF XML or UPO XML that the API exposes as invoice attachments.\n\n## Download The KSeF Invoice XML\n\n```bash\nfakturownia invoice download-attachment --id 111 --kind gov --dir ./attachments --json\n```\n\n## Download The KSeF UPO XML\n\n```bash\nfakturownia invoice download-attachment --id 111 --kind gov_upo --dir ./attachments --json\n```\n\n## Notes\n\n- `kind=gov` means the KSeF invoice XML.\n- `kind=gov_upo` means the KSeF UPO XML.\n- The CLI prefers the upstream filename from response headers when one is provided.\n",
		},
		{
			Key:           "invoice-ksef-tax-id-kinds",
			Name:          "fakturownia-invoice-ksef-tax-id-kinds",
			Title:         "Foreign Buyer And tax_no_kind",
			Description:   "Prepare buyer and seller tax identifier fields for KSeF-aware invoice payloads, including foreign buyers.",
			AreaKey:       "invoices",
			RelatedSkills: []string{"shared", "invoices", "schema"},
			CommandRefs:   []SkillCommandRef{{Noun: "invoice", Verb: "create"}, {Noun: "invoice", Verb: "update"}},
			Markdown:      "Use this when KSeF requires explicit tax identifier semantics for the buyer or seller.\n\n## Example\n\n```bash\nfakturownia invoice create --input '{\n  \"kind\":\"vat\",\n  \"seller_name\":\"Example Sp. z o.o.\",\n  \"seller_tax_no\":\"5252445767\",\n  \"seller_tax_no_kind\":\"\",\n  \"seller_street\":\"ul. Przykładowa 10\",\n  \"seller_post_code\":\"00-001\",\n  \"seller_city\":\"Warszawa\",\n  \"buyer_name\":\"DE Buyer GmbH\",\n  \"buyer_company\":true,\n  \"buyer_tax_no\":\"DE123456789\",\n  \"buyer_tax_no_kind\":\"nip_ue\",\n  \"buyer_country\":\"DE\",\n  \"positions\":[{\"name\":\"Usługa\",\"quantity\":1,\"total_price_gross\":1230,\"tax\":23}]\n}' --json\n```\n\n## Notes\n\n- The API-native field names are `buyer_tax_no_kind` and `seller_tax_no_kind`.\n- Inspect `fakturownia schema invoice create --json` for the curated enum values and related KSeF fields.\n",
		},
		{
			Key:           "invoice-ksef-recipients-issuers",
			Name:          "fakturownia-invoice-ksef-recipients-issuers",
			Title:         "KSeF Recipients And Issuers",
			Description:   "Build recipient and issuer payloads with KSeF-specific roles, tax identifier kinds, and participation fields.",
			AreaKey:       "invoices",
			RelatedSkills: []string{"shared", "invoices", "schema"},
			CommandRefs:   []SkillCommandRef{{Noun: "invoice", Verb: "create"}, {Noun: "invoice", Verb: "update"}},
			Markdown:      "Use this when KSeF invoice payloads need explicit recipient or issuer roles.\n\n## Example\n\n```bash\nfakturownia invoice create --input '{\n  \"kind\":\"vat\",\n  \"seller_name\":\"Wystawca Sp. z o.o.\",\n  \"buyer_name\":\"Klient1 Sp. z o.o.\",\n  \"positions\":[{\"name\":\"Produkt A1\",\"tax\":23,\"total_price_gross\":10.23,\"quantity\":1}],\n  \"recipients\":[{\n    \"name\":\"Odbiorca1\",\n    \"company\":true,\n    \"tax_no\":\"1234567890\",\n    \"tax_no_kind\":\"nip_with_id\",\n    \"role\":\"Dokonujący płatności\",\n    \"participation\":100\n  }],\n  \"issuers\":[{\n    \"name\":\"Wystawca1\",\n    \"company\":true,\n    \"tax_no\":\"1234567890\",\n    \"tax_no_kind\":\"\",\n    \"country\":\"PL\",\n    \"role\":\"Wystawca faktury\"\n  }]\n}' --json\n```\n\n## Notes\n\n- The CLI uses API-native field names, but the recipe language uses KSeF as the business concept.\n- Inspect `request_body_schema` for the curated role and `tax_no_kind` enum values before automating payload generation.\n",
		},
		{
			Key:           "invoice-ksef-correction-to-zero",
			Name:          "fakturownia-invoice-ksef-correction-to-zero",
			Title:         "KSeF Correction And Correction To Zero",
			Description:   "Model KSeF correction workflows, including `gov_corrected_invoice_number` and the practical correction-to-zero pattern.",
			AreaKey:       "invoices",
			RelatedSkills: []string{"shared", "invoices", "schema"},
			CommandRefs:   []SkillCommandRef{{Noun: "invoice", Verb: "create"}, {Noun: "invoice", Verb: "get"}},
			Markdown:      "Use this when a KSeF invoice needs correction and the upstream workflow requires the KSeF number of the corrected invoice.\n\n## Inspect The Existing Invoice\n\n```bash\nfakturownia invoice get --id 2432393 --fields id,number,gov_id,gov_status --json\n```\n\n## Create A KSeF Correction\n\n```bash\nfakturownia invoice create --input '{\n  \"kind\":\"correction\",\n  \"invoice_id\":2432393,\n  \"from_invoice_id\":2432393,\n  \"gov_corrected_invoice_number\":\"KSEF-EXAMPLE-NUMBER\",\n  \"correction_reason\":\"Korekta do zera\",\n  \"positions\":[{\n    \"name\":\"Usługa\",\n    \"quantity\":-1,\n    \"total_price_gross\":\"-123.00\",\n    \"tax\":\"23\",\n    \"kind\":\"correction\"\n  }]\n}' --json\n```\n\n## Notes\n\n- The API-native field name is `gov_corrected_invoice_number`; it is the KSeF number of the corrected invoice.\n- Keep correction-to-zero logic in payloads and operator guidance rather than expecting a dedicated CLI verb.\n",
		},
		{
			Key:           "invoice-send-email",
			Name:          "fakturownia-invoice-send-email",
			Title:         "Send Invoice By Email",
			Description:   "Send an invoice email, optionally overriding recipients and attaching the PDF.",
			AreaKey:       "invoices",
			RelatedSkills: []string{"shared", "invoices"},
			CommandRefs:   []SkillCommandRef{{Noun: "invoice", Verb: "send-email"}},
			Markdown:      "Use this when the invoice already exists and you want Fakturownia to send the email.\n\n## Commands\n\n```bash\nfakturownia invoice send-email --id 100 --json\nfakturownia invoice send-email --id 100 --email-to billing@example.com --email-pdf --json\nfakturownia invoice send-email --id 100 --email-to billing@example.com --update-buyer-email --print-option original --dry-run --json\n```\n\n## KSeF Note\n\n- If the upstream API returns the documented “brak numeru KSeF” response, the CLI surfaces it as an error instead of reporting a successful email send.\n- In API naming, `gov_id` is the KSeF number that must exist before email delivery can succeed for those flows.\n",
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
			CommandRefs:   []SkillCommandRef{{Noun: "invoice", Verb: "add-attachment"}, {Noun: "invoice", Verb: "download-attachment"}, {Noun: "invoice", Verb: "download-attachments"}, {Noun: "invoice", Verb: "update"}},
			Markdown:      "Use this when an invoice needs a new attachment or when you need to fetch either one attachment by kind or all attachments as a ZIP.\n\n## Attach a Local File\n\n```bash\nfakturownia invoice add-attachment --id 111 --file ./scan.pdf --json\n```\n\n## Attach Bytes From Stdin\n\n```bash\ncat ./scan.pdf | fakturownia invoice add-attachment --id 111 --file - --name scan.pdf --dry-run --json\n```\n\n## Make Attachments Visible To Customers\n\n```bash\nfakturownia invoice update --id 111 --input '{\"show_attachments\":true}' --json\n```\n\n## Download A Single Attachment By Kind\n\n```bash\nfakturownia invoice download-attachment --id 111 --kind gov --dir ./attachments --json\n```\n\n## Download All Attachments\n\n```bash\nfakturownia invoice download-attachments --id 111 --dir ./attachments --json\n```\n",
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
