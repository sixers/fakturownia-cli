package spec

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sixers/fakturownia-cli/internal/auth"
	"github.com/sixers/fakturownia-cli/internal/category"
	"github.com/sixers/fakturownia-cli/internal/client"
	"github.com/sixers/fakturownia-cli/internal/doctor"
	"github.com/sixers/fakturownia-cli/internal/invoice"
	"github.com/sixers/fakturownia-cli/internal/output"
	"github.com/sixers/fakturownia-cli/internal/payment"
	"github.com/sixers/fakturownia-cli/internal/pricelist"
	"github.com/sixers/fakturownia-cli/internal/product"
	"github.com/sixers/fakturownia-cli/internal/recurring"
	"github.com/sixers/fakturownia-cli/internal/selfupdate"
	"github.com/sixers/fakturownia-cli/internal/transport"
	"github.com/sixers/fakturownia-cli/internal/warehouse"
	"github.com/sixers/fakturownia-cli/internal/warehouseaction"
	"github.com/sixers/fakturownia-cli/internal/warehousedocument"
)

type fakeAuthService struct {
	loginReq    auth.LoginRequest
	exchangeReq auth.ExchangeRequest
	statusReq   auth.StatusRequest
	logoutReq   auth.LogoutRequest
}

func (f *fakeAuthService) Login(_ context.Context, req auth.LoginRequest) (*auth.LoginResult, error) {
	f.loginReq = req
	return &auth.LoginResult{Profile: req.Profile, URL: "https://acme.fakturownia.pl", DefaultProfile: req.Profile, TokenStored: true}, nil
}

func (f *fakeAuthService) Exchange(_ context.Context, req auth.ExchangeRequest) (*auth.ExchangeResult, error) {
	f.exchangeReq = req
	return &auth.ExchangeResult{
		Login:           req.Login,
		Email:           req.Login,
		Prefix:          "acme",
		URL:             "https://acme.fakturownia.pl",
		FirstName:       "Ada",
		LastName:        "Lovelace",
		APITokenPresent: true,
		SavedProfile:    "acme",
		TokenStored:     true,
		ConfigPath:      "/tmp/config.toml",
		RawBody:         []byte(`{"login":"user@example.com","prefix":"acme","url":"https://acme.fakturownia.pl","api_token":"secret"}`),
	}, nil
}

func (f *fakeAuthService) Status(_ context.Context, req auth.StatusRequest) (*auth.StatusResult, error) {
	f.statusReq = req
	return &auth.StatusResult{Profile: "work", URL: "https://acme.fakturownia.pl", TokenPresent: true}, nil
}

func (f *fakeAuthService) Logout(_ context.Context, req auth.LogoutRequest) (*auth.LogoutResult, error) {
	f.logoutReq = req
	return &auth.LogoutResult{Profile: req.Profile, Removed: true}, nil
}

type fakeCategoryService struct {
	listReq   category.ListRequest
	getReq    category.GetRequest
	createReq category.CreateRequest
	updateReq category.UpdateRequest
	deleteReq category.DeleteRequest
}

func (f *fakeCategoryService) List(_ context.Context, req category.ListRequest) (*category.ListResponse, error) {
	f.listReq = req
	return &category.ListResponse{
		Categories: []map[string]any{
			{
				"id":          100,
				"name":        "my_category",
				"description": "new_description",
			},
		},
		RawBody:    []byte(`[{"id":100,"name":"my_category"}]`),
		Profile:    req.Profile,
		RequestID:  "req-category-list",
		Pagination: output.Pagination{Page: req.Page, PerPage: req.PerPage, Returned: 1, HasNext: false},
	}, nil
}

func (f *fakeCategoryService) Get(_ context.Context, req category.GetRequest) (*category.GetResponse, error) {
	f.getReq = req
	return &category.GetResponse{
		Category: map[string]any{
			"id":          100,
			"name":        "my_category",
			"description": "new_description",
		},
		RawBody:   []byte(`{"id":100,"name":"my_category","description":"new_description"}`),
		Profile:   req.Profile,
		RequestID: "req-category-get",
	}, nil
}

func (f *fakeCategoryService) Create(_ context.Context, req category.CreateRequest) (*category.CreateResponse, error) {
	f.createReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("POST", "/categories.json", nil, map[string]any{"category": req.Input})
		return &category.CreateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &category.CreateResponse{
		Category:  map[string]any{"id": 100, "name": req.Input["name"], "description": req.Input["description"]},
		RawBody:   []byte(`{"id":100,"name":"my_category"}`),
		Profile:   req.Profile,
		RequestID: "req-category-create",
	}, nil
}

func (f *fakeCategoryService) Update(_ context.Context, req category.UpdateRequest) (*category.UpdateResponse, error) {
	f.updateReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("PUT", "/categories/"+req.ID+".json", nil, map[string]any{"category": req.Input})
		return &category.UpdateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &category.UpdateResponse{
		Category:  map[string]any{"id": req.ID, "description": req.Input["description"]},
		RawBody:   []byte(`{"id":100,"description":"new_description"}`),
		Profile:   req.Profile,
		RequestID: "req-category-update",
	}, nil
}

func (f *fakeCategoryService) Delete(_ context.Context, req category.DeleteRequest) (*category.DeleteResponse, error) {
	f.deleteReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("DELETE", "/categories/"+req.ID+".json", nil, nil)
		return &category.DeleteResponse{ID: req.ID, Profile: req.Profile, DryRun: &plan}, nil
	}
	return &category.DeleteResponse{ID: req.ID, Deleted: true, Profile: req.Profile, RequestID: "req-category-delete"}, nil
}

type fakeInvoiceService struct {
	getReq                 invoice.GetRequest
	downloadReq            invoice.DownloadRequest
	createReq              invoice.CreateRequest
	updateReq              invoice.UpdateRequest
	deleteReq              invoice.DeleteRequest
	sendEmailReq           invoice.SendEmailRequest
	sendGovReq             invoice.SendGovRequest
	changeStatusReq        invoice.ChangeStatusRequest
	cancelReq              invoice.CancelRequest
	publicLinkReq          invoice.PublicLinkRequest
	addAttachmentReq       invoice.AddAttachmentRequest
	downloadAttachmentReq  invoice.DownloadAttachmentRequest
	downloadAttachmentsReq invoice.DownloadAttachmentsRequest
	fiscalPrintReq         invoice.FiscalPrintRequest
}

func (f *fakeInvoiceService) List(_ context.Context, req invoice.ListRequest) (*invoice.ListResponse, error) {
	return &invoice.ListResponse{
		Invoices: []map[string]any{
			{
				"id":          1,
				"number":      "FV/1",
				"buyer_name":  "Acme",
				"price_gross": 100,
				"status":      "issued",
				"issue_date":  "2026-04-01",
				"positions": []any{
					map[string]any{"name": "Produkt A", "tax": "23"},
					map[string]any{"name": "Produkt B", "tax": "8"},
				},
			},
		},
		RawBody:    []byte(`[{"id":1}]`),
		Profile:    req.Profile,
		RequestID:  "req-1",
		Pagination: output.Pagination{Page: 1, PerPage: 25, Returned: 1, HasNext: false},
	}, nil
}

func (f *fakeInvoiceService) Get(_ context.Context, req invoice.GetRequest) (*invoice.GetResponse, error) {
	f.getReq = req
	return &invoice.GetResponse{
		Invoice: map[string]any{
			"id":     1,
			"number": "FV/1",
			"status": "issued",
			"token":  "TOKEN-1",
			"positions": []any{
				map[string]any{"name": "Produkt A", "tax": "23"},
				map[string]any{"name": "Produkt B", "tax": "8"},
			},
			"descriptions": []any{
				map[string]any{"content": "Treść uwagi"},
			},
			"settlement_positions": []any{
				map[string]any{"kind": "charge", "amount": "100.00", "reason": "Koszty transportu"},
			},
		},
		RawBody:   []byte(`{"id":1,"number":"FV/1","status":"issued","token":"TOKEN-1","positions":[{"name":"Produkt A","tax":"23"},{"name":"Produkt B","tax":"8"}],"descriptions":[{"content":"Treść uwagi"}],"settlement_positions":[{"kind":"charge","amount":"100.00","reason":"Koszty transportu"}]}`),
		Profile:   req.Profile,
		RequestID: "req-2",
	}, nil
}

func (f *fakeInvoiceService) Download(_ context.Context, req invoice.DownloadRequest) (*invoice.DownloadResponse, error) {
	f.downloadReq = req
	return &invoice.DownloadResponse{ID: req.ID, Path: filepath.Join(".", "invoice-"+req.ID+".pdf"), Bytes: 12, Profile: req.Profile}, nil
}

func (f *fakeInvoiceService) Create(_ context.Context, req invoice.CreateRequest) (*invoice.CreateResponse, error) {
	f.createReq = req
	payload := map[string]any{"invoice": req.Input}
	if req.IdentifyOSS {
		payload["identify_oss"] = "1"
	}
	if req.FillDefaultDescriptions {
		payload["fill_default_descriptions"] = true
	}
	var query map[string][]string
	if req.CorrectionPositions != "" {
		query = map[string][]string{"correction_positions": {req.CorrectionPositions}}
	}
	if req.DryRun {
		plan := transport.RequestPlan{Method: "POST", Path: "/invoices.json", Query: query, Body: map[string]any{"invoice": req.Input, "api_token": "[redacted]"}}
		if req.IdentifyOSS {
			plan.Body.(map[string]any)["identify_oss"] = "1"
		}
		if req.FillDefaultDescriptions {
			plan.Body.(map[string]any)["fill_default_descriptions"] = true
		}
		if req.GovSaveAndSend {
			plan.Body.(map[string]any)["gov_save_and_send"] = true
		}
		return &invoice.CreateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &invoice.CreateResponse{
		Invoice:   map[string]any{"id": 31, "kind": req.Input["kind"], "client_id": req.Input["client_id"]},
		RawBody:   []byte(`{"id":31,"kind":"vat"}`),
		Profile:   req.Profile,
		RequestID: "req-invoice-create",
	}, nil
}

func (f *fakeInvoiceService) Update(_ context.Context, req invoice.UpdateRequest) (*invoice.UpdateResponse, error) {
	f.updateReq = req
	payload := map[string]any{"invoice": req.Input}
	if req.IdentifyOSS {
		payload["identify_oss"] = "1"
	}
	if req.FillDefaultDescriptions {
		payload["fill_default_descriptions"] = true
	}
	if req.GovSaveAndSend {
		payload["gov_save_and_send"] = true
	}
	if req.DryRun {
		var query map[string][]string
		if req.CorrectionPositions != "" {
			query = map[string][]string{"correction_positions": {req.CorrectionPositions}}
		}
		plan := transport.RequestPlan{Method: "PUT", Path: "/invoices/" + req.ID + ".json", Query: query, Body: map[string]any{"invoice": req.Input, "api_token": "[redacted]"}}
		if req.IdentifyOSS {
			plan.Body.(map[string]any)["identify_oss"] = "1"
		}
		if req.FillDefaultDescriptions {
			plan.Body.(map[string]any)["fill_default_descriptions"] = true
		}
		if req.GovSaveAndSend {
			plan.Body.(map[string]any)["gov_save_and_send"] = true
		}
		return &invoice.UpdateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &invoice.UpdateResponse{
		Invoice:   map[string]any{"id": req.ID, "buyer_name": req.Input["buyer_name"], "show_attachments": req.Input["show_attachments"]},
		RawBody:   []byte(`{"id":31}`),
		Profile:   req.Profile,
		RequestID: "req-invoice-update",
	}, nil
}

func (f *fakeInvoiceService) Delete(_ context.Context, req invoice.DeleteRequest) (*invoice.DeleteResponse, error) {
	f.deleteReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("DELETE", "/invoices/"+req.ID+".json", nil, nil)
		return &invoice.DeleteResponse{ID: req.ID, Profile: req.Profile, DryRun: &plan}, nil
	}
	return &invoice.DeleteResponse{ID: req.ID, Deleted: true, Profile: req.Profile, RequestID: "req-invoice-delete"}, nil
}

func (f *fakeInvoiceService) SendEmail(_ context.Context, req invoice.SendEmailRequest) (*invoice.SendEmailResponse, error) {
	f.sendEmailReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("POST", "/invoices/"+req.ID+"/send_by_email.json", nil, nil)
		return &invoice.SendEmailResponse{ID: req.ID, Profile: req.Profile, DryRun: &plan}, nil
	}
	return &invoice.SendEmailResponse{ID: req.ID, Sent: true, Profile: req.Profile, RequestID: "req-invoice-send"}, nil
}

func (f *fakeInvoiceService) SendGov(_ context.Context, req invoice.SendGovRequest) (*invoice.SendGovResponse, error) {
	f.sendGovReq = req
	if req.DryRun {
		plan := transport.RequestPlan{Method: "GET", Path: "/invoices/" + req.ID + ".json", Query: map[string][]string{"send_to_ksef": {"yes"}}}
		return &invoice.SendGovResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &invoice.SendGovResponse{
		Invoice:   map[string]any{"id": req.ID, "gov_status": "processing", "gov_id": nil},
		RawBody:   []byte(`{"id":31,"gov_status":"processing","gov_id":null}`),
		Profile:   req.Profile,
		RequestID: "req-invoice-send-gov",
	}, nil
}

func (f *fakeInvoiceService) ChangeStatus(_ context.Context, req invoice.ChangeStatusRequest) (*invoice.ChangeStatusResponse, error) {
	f.changeStatusReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("POST", "/invoices/"+req.ID+"/change_status.json", nil, nil)
		return &invoice.ChangeStatusResponse{ID: req.ID, Status: req.Status, Profile: req.Profile, DryRun: &plan}, nil
	}
	return &invoice.ChangeStatusResponse{ID: req.ID, Status: req.Status, Changed: true, Profile: req.Profile, RequestID: "req-invoice-status"}, nil
}

func (f *fakeInvoiceService) Cancel(_ context.Context, req invoice.CancelRequest) (*invoice.CancelResponse, error) {
	f.cancelReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("POST", "/invoices/cancel.json", nil, map[string]any{"cancel_invoice_id": req.ID, "cancel_reason": req.Reason})
		return &invoice.CancelResponse{ID: req.ID, Reason: req.Reason, Profile: req.Profile, DryRun: &plan}, nil
	}
	return &invoice.CancelResponse{ID: req.ID, Cancelled: true, Reason: req.Reason, Profile: req.Profile, RequestID: "req-invoice-cancel"}, nil
}

func (f *fakeInvoiceService) PublicLink(_ context.Context, req invoice.PublicLinkRequest) (*invoice.PublicLinkResponse, error) {
	f.publicLinkReq = req
	return &invoice.PublicLinkResponse{
		ID:           req.ID,
		Token:        "TOKEN-1",
		ViewURL:      "https://acme.fakturownia.pl/invoice/TOKEN-1",
		PDFURL:       "https://acme.fakturownia.pl/invoice/TOKEN-1.pdf",
		PDFInlineURL: "https://acme.fakturownia.pl/invoice/TOKEN-1.pdf?inline=yes",
		Profile:      req.Profile,
		RequestID:    "req-invoice-link",
	}, nil
}

func (f *fakeInvoiceService) AddAttachment(_ context.Context, req invoice.AddAttachmentRequest) (*invoice.AddAttachmentResponse, error) {
	f.addAttachmentReq = req
	if req.DryRun {
		return &invoice.AddAttachmentResponse{
			ID:      req.ID,
			Name:    req.Name,
			Bytes:   len(req.Content),
			Profile: req.Profile,
			DryRun: &invoice.AddAttachmentPlan{
				Steps: []invoice.AttachmentStepPlan{
					{Name: "get_credentials", Request: transport.PlanJSONRequest("GET", "/invoices/"+req.ID+"/get_new_attachment_credentials.json", nil, nil)},
				},
			},
		}, nil
	}
	return &invoice.AddAttachmentResponse{ID: req.ID, Name: req.Name, Bytes: len(req.Content), Attached: true, Profile: req.Profile, RequestID: "req-invoice-attach"}, nil
}

func (f *fakeInvoiceService) DownloadAttachments(_ context.Context, req invoice.DownloadAttachmentsRequest) (*invoice.DownloadAttachmentsResponse, error) {
	f.downloadAttachmentsReq = req
	return &invoice.DownloadAttachmentsResponse{ID: req.ID, Path: filepath.Join(".", "invoice-"+req.ID+"-attachments.zip"), Bytes: 42, Profile: req.Profile, RequestID: "req-invoice-zip"}, nil
}

func (f *fakeInvoiceService) DownloadAttachment(_ context.Context, req invoice.DownloadAttachmentRequest) (*invoice.DownloadAttachmentResponse, error) {
	f.downloadAttachmentReq = req
	return &invoice.DownloadAttachmentResponse{ID: req.ID, Kind: req.Kind, Path: filepath.Join(".", "invoice-"+req.ID+"-"+req.Kind+".xml"), FileName: "invoice-" + req.ID + "-" + req.Kind + ".xml", Bytes: 64, Profile: req.Profile, RequestID: "req-invoice-attachment"}, nil
}

func (f *fakeInvoiceService) FiscalPrint(_ context.Context, req invoice.FiscalPrintRequest) (*invoice.FiscalPrintResponse, error) {
	f.fiscalPrintReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("GET", "/invoices/fiscal_print", nil, nil)
		return &invoice.FiscalPrintResponse{InvoiceIDs: req.InvoiceIDs, Printer: req.Printer, Profile: req.Profile, DryRun: &plan}, nil
	}
	return &invoice.FiscalPrintResponse{InvoiceIDs: req.InvoiceIDs, Printer: req.Printer, Submitted: true, Profile: req.Profile, RequestID: "req-invoice-fiscal"}, nil
}

type fakeClientService struct {
	listReq   client.ListRequest
	getReq    client.GetRequest
	createReq client.CreateRequest
	updateReq client.UpdateRequest
	deleteReq client.DeleteRequest
}

type fakePaymentService struct {
	listReq   payment.ListRequest
	getReq    payment.GetRequest
	createReq payment.CreateRequest
	updateReq payment.UpdateRequest
	deleteReq payment.DeleteRequest
}

func (f *fakePaymentService) List(_ context.Context, req payment.ListRequest) (*payment.ListResponse, error) {
	f.listReq = req
	return &payment.ListResponse{
		Payments: []map[string]any{
			{
				"id":         555,
				"name":       "Payment 001",
				"price":      100.05,
				"paid":       true,
				"kind":       "api",
				"invoice_id": nil,
				"invoices": []any{
					map[string]any{"id": 31, "number": "FV/1"},
				},
			},
		},
		RawBody:     []byte(`[{"id":555,"name":"Payment 001"}]`),
		Profile:     req.Profile,
		RequestID:   "req-payment-list",
		IncludeUsed: req.Include,
		Pagination:  output.Pagination{Page: req.Page, PerPage: req.PerPage, Returned: 1, HasNext: false},
	}, nil
}

func (f *fakePaymentService) Get(_ context.Context, req payment.GetRequest) (*payment.GetResponse, error) {
	f.getReq = req
	return &payment.GetResponse{
		Payment: map[string]any{
			"id":         555,
			"name":       "Payment 001",
			"price":      100.05,
			"paid":       true,
			"kind":       "api",
			"invoice_id": nil,
		},
		RawBody:   []byte(`{"id":555,"name":"Payment 001","price":100.05,"paid":true,"kind":"api","invoice_id":null}`),
		Profile:   req.Profile,
		RequestID: "req-payment-get",
	}, nil
}

func (f *fakePaymentService) Create(_ context.Context, req payment.CreateRequest) (*payment.CreateResponse, error) {
	f.createReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("POST", "/banking/payments.json", nil, map[string]any{"banking_payment": req.Input})
		return &payment.CreateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &payment.CreateResponse{
		Payment:   map[string]any{"id": 555, "name": req.Input["name"], "price": req.Input["price"], "paid": req.Input["paid"], "kind": req.Input["kind"]},
		RawBody:   []byte(`{"id":555,"name":"Payment 001"}`),
		Profile:   req.Profile,
		RequestID: "req-payment-create",
	}, nil
}

func (f *fakePaymentService) Update(_ context.Context, req payment.UpdateRequest) (*payment.UpdateResponse, error) {
	f.updateReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("PATCH", "/banking/payments/"+req.ID+".json", nil, map[string]any{"banking_payment": req.Input})
		return &payment.UpdateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &payment.UpdateResponse{
		Payment:   map[string]any{"id": req.ID, "name": req.Input["name"], "price": req.Input["price"]},
		RawBody:   []byte(`{"id":555,"name":"New payment name","price":100}`),
		Profile:   req.Profile,
		RequestID: "req-payment-update",
	}, nil
}

func (f *fakePaymentService) Delete(_ context.Context, req payment.DeleteRequest) (*payment.DeleteResponse, error) {
	f.deleteReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("DELETE", "/banking/payments/"+req.ID+".json", nil, nil)
		return &payment.DeleteResponse{ID: req.ID, Profile: req.Profile, DryRun: &plan}, nil
	}
	return &payment.DeleteResponse{ID: req.ID, Deleted: true, Profile: req.Profile, RequestID: "req-payment-delete"}, nil
}

func (f *fakeClientService) List(_ context.Context, req client.ListRequest) (*client.ListResponse, error) {
	f.listReq = req
	return &client.ListResponse{
		Clients: []map[string]any{
			{
				"id":       11,
				"name":     "Acme Sp. z o.o.",
				"tax_no":   "1234567890",
				"email":    "billing@acme.test",
				"city":     "Warsaw",
				"country":  "PL",
				"tag_list": []any{"vip", "b2b"},
			},
		},
		RawBody:    []byte(`[{"id":11,"name":"Acme Sp. z o.o."}]`),
		Profile:    req.Profile,
		RequestID:  "req-client-list",
		Pagination: output.Pagination{Page: req.Page, PerPage: req.PerPage, Returned: 1, HasNext: false},
	}, nil
}

func (f *fakeClientService) Get(_ context.Context, req client.GetRequest) (*client.GetResponse, error) {
	f.getReq = req
	value := req.ID
	if value == "" {
		value = req.ExternalID
	}
	return &client.GetResponse{
		Client: map[string]any{
			"id":          11,
			"name":        "Acme Sp. z o.o.",
			"email":       "billing@acme.test",
			"external_id": value,
			"tag_list":    []any{"vip", "b2b"},
		},
		RawBody:   []byte(`{"id":11,"name":"Acme Sp. z o.o.","email":"billing@acme.test"}`),
		Profile:   req.Profile,
		RequestID: "req-client-get",
	}, nil
}

func (f *fakeClientService) Create(_ context.Context, req client.CreateRequest) (*client.CreateResponse, error) {
	f.createReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("POST", "/clients.json", nil, map[string]any{"client": req.Input})
		return &client.CreateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &client.CreateResponse{
		Client: map[string]any{
			"id":    12,
			"name":  req.Input["name"],
			"email": req.Input["email"],
		},
		RawBody:   []byte(`{"id":12,"name":"New Client"}`),
		Profile:   req.Profile,
		RequestID: "req-client-create",
	}, nil
}

func (f *fakeClientService) Update(_ context.Context, req client.UpdateRequest) (*client.UpdateResponse, error) {
	f.updateReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("PUT", "/clients/"+req.ID+".json", nil, map[string]any{"client": req.Input})
		return &client.UpdateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &client.UpdateResponse{
		Client: map[string]any{
			"id":    req.ID,
			"email": req.Input["email"],
		},
		RawBody:   []byte(`{"id":12,"email":"updated@example.com"}`),
		Profile:   req.Profile,
		RequestID: "req-client-update",
	}, nil
}

func (f *fakeClientService) Delete(_ context.Context, req client.DeleteRequest) (*client.DeleteResponse, error) {
	f.deleteReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("DELETE", "/clients/"+req.ID+".json", nil, nil)
		return &client.DeleteResponse{ID: req.ID, Profile: req.Profile, DryRun: &plan}, nil
	}
	return &client.DeleteResponse{
		ID:        req.ID,
		Deleted:   true,
		Profile:   req.Profile,
		RequestID: "req-client-delete",
	}, nil
}

type fakeDoctorService struct {
	runReq doctor.RunRequest
}

func (f *fakeDoctorService) Run(_ context.Context, req doctor.RunRequest) (*doctor.RunResult, error) {
	f.runReq = req
	return &doctor.RunResult{
		Profile: "work",
		Report: doctor.Report{
			Version: req.Version,
			Status:  "ok",
			Checks: []doctor.Check{
				{Name: "config-path", Status: "ok", Message: "using config path"},
			},
		},
	}, nil
}

type fakeProductService struct {
	listReq   product.ListRequest
	getReq    product.GetRequest
	createReq product.CreateRequest
	updateReq product.UpdateRequest
}

func (f *fakeProductService) List(_ context.Context, req product.ListRequest) (*product.ListResponse, error) {
	f.listReq = req
	return &product.ListResponse{
		Products: []map[string]any{
			{
				"id":          21,
				"name":        "Widget",
				"code":        "W-001",
				"price_gross": "123.00",
				"tax":         "23",
				"stock_level": "9.0",
				"tag_list":    []any{"core", "retail"},
				"gtu_codes":   []any{"GTU_01"},
			},
		},
		RawBody:    []byte(`[{"id":21,"name":"Widget"}]`),
		Profile:    req.Profile,
		RequestID:  "req-product-list",
		Pagination: output.Pagination{Page: req.Page, PerPage: req.PerPage, Returned: 1, HasNext: false},
	}, nil
}

func (f *fakeProductService) Get(_ context.Context, req product.GetRequest) (*product.GetResponse, error) {
	f.getReq = req
	return &product.GetResponse{
		Product: map[string]any{
			"id":           21,
			"name":         "Widget",
			"warehouse_id": req.WarehouseID,
			"stock_level":  "9.0",
			"gtu_codes":    []any{"GTU_01"},
		},
		RawBody:   []byte(`{"id":21,"name":"Widget","stock_level":"9.0"}`),
		Profile:   req.Profile,
		RequestID: "req-product-get",
	}, nil
}

func (f *fakeProductService) Create(_ context.Context, req product.CreateRequest) (*product.CreateResponse, error) {
	f.createReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("POST", "/products.json", nil, map[string]any{"product": req.Input})
		return &product.CreateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &product.CreateResponse{
		Product: map[string]any{
			"id":   22,
			"name": req.Input["name"],
			"code": req.Input["code"],
			"tax":  req.Input["tax"],
		},
		RawBody:   []byte(`{"id":22,"name":"New Product"}`),
		Profile:   req.Profile,
		RequestID: "req-product-create",
	}, nil
}

func (f *fakeProductService) Update(_ context.Context, req product.UpdateRequest) (*product.UpdateResponse, error) {
	f.updateReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("PUT", "/products/"+req.ID+".json", nil, map[string]any{"product": req.Input})
		return &product.UpdateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &product.UpdateResponse{
		Product: map[string]any{
			"id":          req.ID,
			"price_gross": req.Input["price_gross"],
		},
		RawBody:   []byte(`{"id":22,"price_gross":"102"}`),
		Profile:   req.Profile,
		RequestID: "req-product-update",
	}, nil
}

type fakePriceListService struct {
	listReq   pricelist.ListRequest
	getReq    pricelist.GetRequest
	createReq pricelist.CreateRequest
	updateReq pricelist.UpdateRequest
	deleteReq pricelist.DeleteRequest
}

func (f *fakePriceListService) List(_ context.Context, req pricelist.ListRequest) (*pricelist.ListResponse, error) {
	f.listReq = req
	return &pricelist.ListResponse{
		PriceLists: []map[string]any{
			{
				"id":          8523,
				"name":        "Dropshipper",
				"description": "test",
				"currency":    "PLN",
			},
		},
		RawBody:    []byte(`[{"id":8523,"name":"Dropshipper"}]`),
		Profile:    req.Profile,
		RequestID:  "req-price-list-list",
		Pagination: output.Pagination{Page: req.Page, PerPage: req.PerPage, Returned: 1, HasNext: false},
	}, nil
}

func (f *fakePriceListService) Get(_ context.Context, req pricelist.GetRequest) (*pricelist.GetResponse, error) {
	f.getReq = req
	return &pricelist.GetResponse{
		PriceList: map[string]any{
			"id":          8523,
			"name":        "Dropshipper",
			"description": "test",
			"currency":    "PLN",
			"price_list_positions": []any{
				map[string]any{"id": 556438, "priceable_id": 97149307, "price_gross": "33.16"},
			},
		},
		RawBody:   []byte(`{"id":8523,"name":"Dropshipper","price_list_positions":[{"id":556438,"priceable_id":97149307,"price_gross":"33.16"}]}`),
		Profile:   req.Profile,
		RequestID: "req-price-list-get",
	}, nil
}

func (f *fakePriceListService) Create(_ context.Context, req pricelist.CreateRequest) (*pricelist.CreateResponse, error) {
	f.createReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("POST", "/price_lists.json", nil, map[string]any{"price_list": req.Input})
		return &pricelist.CreateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &pricelist.CreateResponse{
		PriceList: map[string]any{"id": 8523, "name": req.Input["name"], "currency": req.Input["currency"]},
		RawBody:   []byte(`{"id":8523,"name":"Dropshipper"}`),
		Profile:   req.Profile,
		RequestID: "req-price-list-create",
	}, nil
}

func (f *fakePriceListService) Update(_ context.Context, req pricelist.UpdateRequest) (*pricelist.UpdateResponse, error) {
	f.updateReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("PUT", "/price_lists/"+req.ID+".json", nil, map[string]any{"price_list": req.Input})
		return &pricelist.UpdateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &pricelist.UpdateResponse{
		PriceList: map[string]any{"id": req.ID, "description": req.Input["description"], "currency": req.Input["currency"]},
		RawBody:   []byte(`{"id":8523,"description":"updated"}`),
		Profile:   req.Profile,
		RequestID: "req-price-list-update",
	}, nil
}

func (f *fakePriceListService) Delete(_ context.Context, req pricelist.DeleteRequest) (*pricelist.DeleteResponse, error) {
	f.deleteReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("DELETE", "/price_lists/"+req.ID+".json", nil, nil)
		return &pricelist.DeleteResponse{ID: req.ID, Profile: req.Profile, DryRun: &plan}, nil
	}
	return &pricelist.DeleteResponse{ID: req.ID, Deleted: true, Profile: req.Profile, RequestID: "req-price-list-delete"}, nil
}

type fakeSelfUpdateService struct {
	updateReq selfupdate.UpdateRequest
}

type fakeRecurringService struct {
	listReq   recurring.ListRequest
	createReq recurring.CreateRequest
	updateReq recurring.UpdateRequest
}

func (f *fakeRecurringService) List(_ context.Context, req recurring.ListRequest) (*recurring.ListResponse, error) {
	f.listReq = req
	return &recurring.ListResponse{
		Recurrings: []map[string]any{{"id": 41, "name": "Miesięczna", "invoice_id": 1, "every": "1m", "next_invoice_date": "2016-02-01", "send_email": true}},
		RawBody:    []byte(`[{"id":41,"name":"Miesięczna"}]`),
		Profile:    req.Profile,
		RequestID:  "req-recurring-list",
	}, nil
}

func (f *fakeRecurringService) Create(_ context.Context, req recurring.CreateRequest) (*recurring.CreateResponse, error) {
	f.createReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("POST", "/recurrings.json", nil, map[string]any{"recurring": req.Input})
		return &recurring.CreateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &recurring.CreateResponse{
		Recurring: map[string]any{"id": 41, "name": req.Input["name"], "every": req.Input["every"]},
		RawBody:   []byte(`{"id":41,"name":"Miesięczna"}`),
		Profile:   req.Profile,
		RequestID: "req-recurring-create",
	}, nil
}

func (f *fakeRecurringService) Update(_ context.Context, req recurring.UpdateRequest) (*recurring.UpdateResponse, error) {
	f.updateReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("PUT", "/recurrings/"+req.ID+".json", nil, map[string]any{"recurring": req.Input})
		return &recurring.UpdateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &recurring.UpdateResponse{
		Recurring: map[string]any{"id": req.ID, "next_invoice_date": req.Input["next_invoice_date"]},
		RawBody:   []byte(`{"id":41,"next_invoice_date":"2016-02-01"}`),
		Profile:   req.Profile,
		RequestID: "req-recurring-update",
	}, nil
}

type fakeWarehouseService struct {
	listReq   warehouse.ListRequest
	getReq    warehouse.GetRequest
	createReq warehouse.CreateRequest
	updateReq warehouse.UpdateRequest
	deleteReq warehouse.DeleteRequest
}

func (f *fakeWarehouseService) List(_ context.Context, req warehouse.ListRequest) (*warehouse.ListResponse, error) {
	f.listReq = req
	return &warehouse.ListResponse{
		Warehouses: []map[string]any{
			{
				"id":          1,
				"name":        "my_warehouse",
				"kind":        nil,
				"description": "new_description",
			},
		},
		RawBody:    []byte(`[{"id":1,"name":"my_warehouse"}]`),
		Profile:    req.Profile,
		RequestID:  "req-warehouse-entity-list",
		Pagination: output.Pagination{Page: req.Page, PerPage: req.PerPage, Returned: 1, HasNext: false},
	}, nil
}

func (f *fakeWarehouseService) Get(_ context.Context, req warehouse.GetRequest) (*warehouse.GetResponse, error) {
	f.getReq = req
	return &warehouse.GetResponse{
		Warehouse: map[string]any{
			"id":          1,
			"name":        "my_warehouse",
			"kind":        nil,
			"description": "new_description",
		},
		RawBody:   []byte(`{"id":1,"name":"my_warehouse","description":"new_description"}`),
		Profile:   req.Profile,
		RequestID: "req-warehouse-entity-get",
	}, nil
}

func (f *fakeWarehouseService) Create(_ context.Context, req warehouse.CreateRequest) (*warehouse.CreateResponse, error) {
	f.createReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("POST", "/warehouses.json", nil, map[string]any{"warehouse": req.Input})
		return &warehouse.CreateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &warehouse.CreateResponse{
		Warehouse: map[string]any{"id": 1, "name": req.Input["name"], "kind": req.Input["kind"], "description": req.Input["description"]},
		RawBody:   []byte(`{"id":1,"name":"my_warehouse"}`),
		Profile:   req.Profile,
		RequestID: "req-warehouse-entity-create",
	}, nil
}

func (f *fakeWarehouseService) Update(_ context.Context, req warehouse.UpdateRequest) (*warehouse.UpdateResponse, error) {
	f.updateReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("PUT", "/warehouses/"+req.ID+".json", nil, map[string]any{"warehouse": req.Input})
		return &warehouse.UpdateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &warehouse.UpdateResponse{
		Warehouse: map[string]any{"id": req.ID, "description": req.Input["description"], "name": req.Input["name"]},
		RawBody:   []byte(`{"id":1,"description":"new_description"}`),
		Profile:   req.Profile,
		RequestID: "req-warehouse-entity-update",
	}, nil
}

func (f *fakeWarehouseService) Delete(_ context.Context, req warehouse.DeleteRequest) (*warehouse.DeleteResponse, error) {
	f.deleteReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("DELETE", "/warehouses/"+req.ID+".json", nil, nil)
		return &warehouse.DeleteResponse{ID: req.ID, Profile: req.Profile, DryRun: &plan}, nil
	}
	return &warehouse.DeleteResponse{ID: req.ID, Deleted: true, Profile: req.Profile, RequestID: "req-warehouse-entity-delete"}, nil
}

type fakeWarehouseActionService struct {
	listReq warehouseaction.ListRequest
}

func (f *fakeWarehouseActionService) List(_ context.Context, req warehouseaction.ListRequest) (*warehouseaction.ListResponse, error) {
	f.listReq = req
	return &warehouseaction.ListResponse{
		WarehouseActions: []map[string]any{
			{
				"id":                    77,
				"kind":                  "mm",
				"product_id":            7,
				"quantity":              2,
				"warehouse_id":          1,
				"warehouse_document_id": 15,
				"warehouse2_id":         3,
			},
		},
		RawBody:    []byte(`[{"id":77,"kind":"mm","product_id":7,"quantity":2}]`),
		Profile:    req.Profile,
		RequestID:  "req-warehouse-action-list",
		Pagination: output.Pagination{Page: req.Page, PerPage: req.PerPage, Returned: 1, HasNext: false},
	}, nil
}

type fakeWarehouseDocumentService struct {
	listReq   warehousedocument.ListRequest
	getReq    warehousedocument.GetRequest
	createReq warehousedocument.CreateRequest
	updateReq warehousedocument.UpdateRequest
	deleteReq warehousedocument.DeleteRequest
}

func (f *fakeWarehouseDocumentService) List(_ context.Context, req warehousedocument.ListRequest) (*warehousedocument.ListResponse, error) {
	f.listReq = req
	return &warehousedocument.ListResponse{
		WarehouseDocuments: []map[string]any{
			{
				"id":           15,
				"kind":         "mm",
				"number":       "MM/1/2026",
				"issue_date":   "2026-04-01",
				"warehouse_id": 1,
				"client_name":  "Acme",
			},
		},
		RawBody:    []byte(`[{"id":15,"kind":"mm"}]`),
		Profile:    req.Profile,
		RequestID:  "req-warehouse-list",
		Pagination: output.Pagination{Page: req.Page, PerPage: req.PerPage, Returned: 1, HasNext: false},
	}, nil
}

func (f *fakeWarehouseDocumentService) Get(_ context.Context, req warehousedocument.GetRequest) (*warehousedocument.GetResponse, error) {
	f.getReq = req
	return &warehousedocument.GetResponse{
		WarehouseDocument: map[string]any{
			"id":           15,
			"kind":         "wz",
			"number":       "WZ/1/2026",
			"client_name":  "Acme",
			"warehouse_id": 1,
			"warehouse_actions": []any{
				map[string]any{"product_id": 7, "quantity": 2, "warehouse2_id": 3},
			},
		},
		RawBody:   []byte(`{"id":15,"kind":"wz","warehouse_actions":[{"product_id":7,"quantity":2,"warehouse2_id":3}]}`),
		Profile:   req.Profile,
		RequestID: "req-warehouse-get",
	}, nil
}

func (f *fakeWarehouseDocumentService) Create(_ context.Context, req warehousedocument.CreateRequest) (*warehousedocument.CreateResponse, error) {
	f.createReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("POST", "/warehouse_documents.json", nil, map[string]any{"warehouse_document": req.Input})
		return &warehousedocument.CreateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &warehousedocument.CreateResponse{
		WarehouseDocument: map[string]any{"id": 15, "kind": req.Input["kind"], "warehouse_actions": req.Input["warehouse_actions"]},
		RawBody:           []byte(`{"id":15,"kind":"mm"}`),
		Profile:           req.Profile,
		RequestID:         "req-warehouse-create",
	}, nil
}

func (f *fakeWarehouseDocumentService) Update(_ context.Context, req warehousedocument.UpdateRequest) (*warehousedocument.UpdateResponse, error) {
	f.updateReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("PUT", "/warehouse_documents/"+req.ID+".json", nil, map[string]any{"warehouse_document": req.Input})
		return &warehousedocument.UpdateResponse{Profile: req.Profile, DryRun: &plan}, nil
	}
	return &warehousedocument.UpdateResponse{
		WarehouseDocument: map[string]any{"id": req.ID, "invoice_ids": req.Input["invoice_ids"]},
		RawBody:           []byte(`{"id":15,"invoice_ids":[100,111]}`),
		Profile:           req.Profile,
		RequestID:         "req-warehouse-update",
	}, nil
}

func (f *fakeWarehouseDocumentService) Delete(_ context.Context, req warehousedocument.DeleteRequest) (*warehousedocument.DeleteResponse, error) {
	f.deleteReq = req
	if req.DryRun {
		plan := transport.PlanJSONRequest("DELETE", "/warehouse_documents/"+req.ID+".json", nil, nil)
		return &warehousedocument.DeleteResponse{ID: req.ID, Profile: req.Profile, DryRun: &plan}, nil
	}
	return &warehousedocument.DeleteResponse{ID: req.ID, Deleted: true, Profile: req.Profile, RequestID: "req-warehouse-delete"}, nil
}

func (f *fakeSelfUpdateService) Update(_ context.Context, req selfupdate.UpdateRequest) (*selfupdate.UpdateResult, error) {
	f.updateReq = req
	return &selfupdate.UpdateResult{
		RequestedVersion: normalizeRequested(req.TargetVersion),
		CurrentVersion:   req.CurrentVersion,
		TargetVersion:    "v9.9.9",
		ExecutablePath:   "/tmp/fakturownia",
		OS:               "darwin",
		Arch:             "arm64",
		ReleaseURL:       "https://github.com/sixers/fakturownia-cli/releases/tag/v9.9.9",
		AssetName:        "fakturownia_9.9.9_darwin_arm64.tar.gz",
		DownloadURL:      "https://example.test/fakturownia_9.9.9_darwin_arm64.tar.gz",
		ChecksumURL:      "https://example.test/checksums.txt",
		Updated:          !req.DryRun,
		DryRun:           req.DryRun,
		ChecksumVerified: !req.DryRun,
	}, nil
}

func normalizeRequested(value string) string {
	if strings.TrimSpace(value) == "" {
		return "latest"
	}
	return value
}

func TestCommandIntegration(t *testing.T) {
	authSvc := &fakeAuthService{}
	categorySvc := &fakeCategoryService{}
	clientSvc := &fakeClientService{}
	invoiceSvc := &fakeInvoiceService{}
	paymentSvc := &fakePaymentService{}
	productSvc := &fakeProductService{}
	priceListSvc := &fakePriceListService{}
	recurringSvc := &fakeRecurringService{}
	warehouseEntitySvc := &fakeWarehouseService{}
	warehouseActionSvc := &fakeWarehouseActionService{}
	warehouseSvc := &fakeWarehouseDocumentService{}
	doctorSvc := &fakeDoctorService{}
	selfSvc := &fakeSelfUpdateService{}

	runWithInput := func(input string, args ...string) (string, string, error) {
		var stdout, stderr bytes.Buffer
		cmd := NewRootCommand(Dependencies{
			Auth:            authSvc,
			Category:        categorySvc,
			Client:          clientSvc,
			Invoice:         invoiceSvc,
			Payment:         paymentSvc,
			Product:         productSvc,
			PriceList:       priceListSvc,
			Recurring:       recurringSvc,
			Warehouses:      warehouseEntitySvc,
			WarehouseAction: warehouseActionSvc,
			Warehouse:       warehouseSvc,
			Doctor:          doctorSvc,
			Self:            selfSvc,
			Stdout:          &stdout,
			Stderr:          &stderr,
		})
		if input != "" {
			cmd.SetIn(strings.NewReader(input))
		}
		cmd.SetArgs(args)
		err := cmd.Execute()
		return stdout.String(), stderr.String(), err
	}
	run := func(args ...string) (string, string, error) {
		return runWithInput("", args...)
	}

	_, _, err := run("auth", "login", "--profile", "work", "--prefix", "acme", "--api-token", "token", "--json")
	if err != nil {
		t.Fatalf("auth login error = %v", err)
	}
	if authSvc.loginReq.Profile != "work" {
		t.Fatalf("expected login profile to come from --profile, got %q", authSvc.loginReq.Profile)
	}

	stdout, _, err := run("invoice", "list", "--json")
	if err != nil {
		t.Fatalf("invoice list error = %v", err)
	}
	if !jsonContains(stdout, `"status": "success"`) {
		t.Fatalf("unexpected invoice list output: %s", stdout)
	}

	stdout, _, err = run("client", "list", "--json")
	if err != nil {
		t.Fatalf("client list error = %v", err)
	}
	if !jsonContains(stdout, `"name": "Acme Sp. z o.o."`) {
		t.Fatalf("unexpected client list output: %s", stdout)
	}

	stdout, _, err = run("category", "list", "--json")
	if err != nil {
		t.Fatalf("category list error = %v", err)
	}
	if !jsonContains(stdout, `"name": "my_category"`) {
		t.Fatalf("unexpected category list output: %s", stdout)
	}

	stdout, _, err = run("category", "get", "--id", "100", "--json")
	if err != nil {
		t.Fatalf("category get error = %v", err)
	}
	if categorySvc.getReq.ID != "100" {
		t.Fatalf("expected category get ID to be forwarded, got %q", categorySvc.getReq.ID)
	}
	if !jsonContains(stdout, `"description": "new_description"`) {
		t.Fatalf("unexpected category get output: %s", stdout)
	}

	stdout, _, err = run("category", "create", "--input", `{"name":"my_category","description":null}`, "--json")
	if err != nil {
		t.Fatalf("category create error = %v", err)
	}
	if categorySvc.createReq.Input["name"] != "my_category" {
		t.Fatalf("expected category create input to be parsed, got %#v", categorySvc.createReq.Input)
	}
	if !jsonContains(stdout, `"id": 100`) {
		t.Fatalf("unexpected category create output: %s", stdout)
	}

	stdout, _, err = runWithInput(`{"description":"new_description"}`, "category", "update", "--id", "100", "--input", "-", "--json")
	if err != nil {
		t.Fatalf("category update error = %v", err)
	}
	if categorySvc.updateReq.Input["description"] != "new_description" {
		t.Fatalf("expected category update stdin input, got %#v", categorySvc.updateReq.Input)
	}
	if !jsonContains(stdout, `"description": "new_description"`) {
		t.Fatalf("unexpected category update output: %s", stdout)
	}

	stdout, _, err = run("category", "delete", "--id", "100", "--yes", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("category delete dry-run error = %v", err)
	}
	if !categorySvc.deleteReq.DryRun {
		t.Fatal("expected category delete dry-run flag to be forwarded")
	}
	if !jsonContains(stdout, `"method": "DELETE"`) {
		t.Fatalf("unexpected category delete dry-run output: %s", stdout)
	}

	stdout, _, err = run("client", "get", "--external-id", "ext-123", "--json")
	if err != nil {
		t.Fatalf("client get by external-id error = %v", err)
	}
	if clientSvc.getReq.ExternalID != "ext-123" {
		t.Fatalf("expected client get to receive external ID, got %q", clientSvc.getReq.ExternalID)
	}
	if !jsonContains(stdout, `"external_id": "ext-123"`) {
		t.Fatalf("unexpected client get output: %s", stdout)
	}

	stdout, _, err = run("client", "create", "--input", `{"name":"New Client","email":"new@example.com"}`, "--json")
	if err != nil {
		t.Fatalf("client create error = %v", err)
	}
	if clientSvc.createReq.Input["name"] != "New Client" {
		t.Fatalf("expected client create input to be parsed, got %#v", clientSvc.createReq.Input)
	}
	if !jsonContains(stdout, `"id": 12`) {
		t.Fatalf("unexpected client create output: %s", stdout)
	}

	stdout, _, err = runWithInput(`{"email":"stdin@example.com"}`, "client", "update", "--id", "12", "--input", "-", "--json")
	if err != nil {
		t.Fatalf("client update error = %v", err)
	}
	if clientSvc.updateReq.Input["email"] != "stdin@example.com" {
		t.Fatalf("expected client update stdin input, got %#v", clientSvc.updateReq.Input)
	}
	if !jsonContains(stdout, `"email": "stdin@example.com"`) {
		t.Fatalf("unexpected client update output: %s", stdout)
	}

	stdout, _, err = run("client", "delete", "--id", "12", "--yes", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("client delete dry-run error = %v", err)
	}
	if !clientSvc.deleteReq.DryRun {
		t.Fatal("expected client delete dry-run flag to be forwarded")
	}
	if !jsonContains(stdout, `"method": "DELETE"`) || !jsonContains(stdout, `"[redacted]"`) {
		t.Fatalf("unexpected client delete dry-run output: %s", stdout)
	}

	stdout, _, err = run("payment", "list", "--include", "invoices", "--json")
	if err != nil {
		t.Fatalf("payment list error = %v", err)
	}
	if len(paymentSvc.listReq.Include) != 1 || paymentSvc.listReq.Include[0] != "invoices" {
		t.Fatalf("expected payment include to be forwarded, got %#v", paymentSvc.listReq.Include)
	}
	if !jsonContains(stdout, `"name": "Payment 001"`) {
		t.Fatalf("unexpected payment list output: %s", stdout)
	}

	stdout, _, err = run("payment", "get", "--id", "555", "--json")
	if err != nil {
		t.Fatalf("payment get error = %v", err)
	}
	if paymentSvc.getReq.ID != "555" {
		t.Fatalf("expected payment get ID to be forwarded, got %q", paymentSvc.getReq.ID)
	}
	if !jsonContains(stdout, `"price": 100.05`) {
		t.Fatalf("unexpected payment get output: %s", stdout)
	}

	stdout, _, err = run("payment", "create", "--input", `{"name":"Payment 001","price":100.05,"invoice_id":null,"paid":true,"kind":"api"}`, "--json")
	if err != nil {
		t.Fatalf("payment create error = %v", err)
	}
	if paymentSvc.createReq.Input["name"] != "Payment 001" {
		t.Fatalf("expected payment create input to be parsed, got %#v", paymentSvc.createReq.Input)
	}
	if !jsonContains(stdout, `"id": 555`) {
		t.Fatalf("unexpected payment create output: %s", stdout)
	}

	stdout, _, err = runWithInput(`{"name":"New payment name","price":100}`, "payment", "update", "--id", "555", "--input", "-", "--json")
	if err != nil {
		t.Fatalf("payment update error = %v", err)
	}
	if paymentSvc.updateReq.Input["name"] != "New payment name" {
		t.Fatalf("expected payment update stdin input, got %#v", paymentSvc.updateReq.Input)
	}
	if !jsonContains(stdout, `"name": "New payment name"`) {
		t.Fatalf("unexpected payment update output: %s", stdout)
	}

	stdout, _, err = run("payment", "delete", "--id", "555", "--yes", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("payment delete dry-run error = %v", err)
	}
	if !paymentSvc.deleteReq.DryRun {
		t.Fatal("expected payment delete dry-run flag to be forwarded")
	}
	if !jsonContains(stdout, `"method": "DELETE"`) {
		t.Fatalf("unexpected payment delete dry-run output: %s", stdout)
	}

	stdout, _, err = run("product", "list", "--date-from", "2025-11-01", "--warehouse-id", "7", "--json")
	if err != nil {
		t.Fatalf("product list error = %v", err)
	}
	if productSvc.listReq.DateFrom != "2025-11-01" || productSvc.listReq.WarehouseID != "7" {
		t.Fatalf("expected product list filters to be forwarded, got %#v", productSvc.listReq)
	}
	if !jsonContains(stdout, `"name": "Widget"`) {
		t.Fatalf("unexpected product list output: %s", stdout)
	}

	stdout, _, err = run("product", "get", "--id", "21", "--warehouse-id", "3", "--json")
	if err != nil {
		t.Fatalf("product get error = %v", err)
	}
	if productSvc.getReq.WarehouseID != "3" {
		t.Fatalf("expected product get warehouse ID to be forwarded, got %q", productSvc.getReq.WarehouseID)
	}
	if !jsonContains(stdout, `"stock_level": "9.0"`) {
		t.Fatalf("unexpected product get output: %s", stdout)
	}

	stdout, _, err = run("product", "create", "--input", `{"name":"Widget","code":"W-001","tax":"23"}`, "--json")
	if err != nil {
		t.Fatalf("product create error = %v", err)
	}
	if productSvc.createReq.Input["name"] != "Widget" {
		t.Fatalf("expected product create input to be parsed, got %#v", productSvc.createReq.Input)
	}
	if !jsonContains(stdout, `"id": 22`) {
		t.Fatalf("unexpected product create output: %s", stdout)
	}

	stdout, _, err = runWithInput(`{"price_gross":"102"}`, "product", "update", "--id", "22", "--input", "-", "--json")
	if err != nil {
		t.Fatalf("product update error = %v", err)
	}
	if productSvc.updateReq.Input["price_gross"] != "102" {
		t.Fatalf("expected product update stdin input, got %#v", productSvc.updateReq.Input)
	}
	if !jsonContains(stdout, `"price_gross": "102"`) {
		t.Fatalf("unexpected product update output: %s", stdout)
	}

	stdout, _, err = run("price-list", "list", "--json")
	if err != nil {
		t.Fatalf("price-list list error = %v", err)
	}
	if !jsonContains(stdout, `"name": "Dropshipper"`) {
		t.Fatalf("unexpected price-list list output: %s", stdout)
	}

	stdout, _, err = run("price-list", "get", "--id", "8523", "--json")
	if err != nil {
		t.Fatalf("price-list get error = %v", err)
	}
	if priceListSvc.getReq.ID != "8523" {
		t.Fatalf("expected price-list get ID to be forwarded, got %q", priceListSvc.getReq.ID)
	}
	if !jsonContains(stdout, `"price_list_positions": [`) {
		t.Fatalf("unexpected price-list get output: %s", stdout)
	}

	stdout, _, err = run("price-list", "create", "--input", `{"name":"Dropshipper","currency":"PLN"}`, "--json")
	if err != nil {
		t.Fatalf("price-list create error = %v", err)
	}
	if priceListSvc.createReq.Input["name"] != "Dropshipper" {
		t.Fatalf("expected price-list create input to be parsed, got %#v", priceListSvc.createReq.Input)
	}
	if !jsonContains(stdout, `"id": 8523`) {
		t.Fatalf("unexpected price-list create output: %s", stdout)
	}

	stdout, _, err = runWithInput(`{"description":"updated"}`, "price-list", "update", "--id", "8523", "--input", "-", "--json")
	if err != nil {
		t.Fatalf("price-list update error = %v", err)
	}
	if priceListSvc.updateReq.Input["description"] != "updated" {
		t.Fatalf("expected price-list update stdin input, got %#v", priceListSvc.updateReq.Input)
	}
	if !jsonContains(stdout, `"description": "updated"`) {
		t.Fatalf("unexpected price-list update output: %s", stdout)
	}

	stdout, _, err = run("price-list", "delete", "--id", "8523", "--yes", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("price-list delete dry-run error = %v", err)
	}
	if !priceListSvc.deleteReq.DryRun {
		t.Fatal("expected price-list delete dry-run flag to be forwarded")
	}
	if !jsonContains(stdout, `"method": "DELETE"`) {
		t.Fatalf("unexpected price-list delete dry-run output: %s", stdout)
	}

	stdout, _, err = run("warehouse", "list", "--json")
	if err != nil {
		t.Fatalf("warehouse list error = %v", err)
	}
	if !jsonContains(stdout, `"name": "my_warehouse"`) {
		t.Fatalf("unexpected warehouse list output: %s", stdout)
	}

	stdout, _, err = run("warehouse", "get", "--id", "1", "--json")
	if err != nil {
		t.Fatalf("warehouse get error = %v", err)
	}
	if warehouseEntitySvc.getReq.ID != "1" {
		t.Fatalf("expected warehouse get ID to be forwarded, got %q", warehouseEntitySvc.getReq.ID)
	}
	if !jsonContains(stdout, `"description": "new_description"`) {
		t.Fatalf("unexpected warehouse get output: %s", stdout)
	}

	stdout, _, err = run("warehouse", "create", "--input", `{"name":"my_warehouse","kind":null,"description":null}`, "--json")
	if err != nil {
		t.Fatalf("warehouse create error = %v", err)
	}
	if warehouseEntitySvc.createReq.Input["name"] != "my_warehouse" {
		t.Fatalf("expected warehouse create input to be parsed, got %#v", warehouseEntitySvc.createReq.Input)
	}
	if !jsonContains(stdout, `"id": 1`) {
		t.Fatalf("unexpected warehouse create output: %s", stdout)
	}

	stdout, _, err = runWithInput(`{"description":"new_description"}`, "warehouse", "update", "--id", "1", "--input", "-", "--json")
	if err != nil {
		t.Fatalf("warehouse update error = %v", err)
	}
	if warehouseEntitySvc.updateReq.Input["description"] != "new_description" {
		t.Fatalf("expected warehouse update stdin input, got %#v", warehouseEntitySvc.updateReq.Input)
	}
	if !jsonContains(stdout, `"description": "new_description"`) {
		t.Fatalf("unexpected warehouse update output: %s", stdout)
	}

	stdout, _, err = run("warehouse", "delete", "--id", "1", "--yes", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("warehouse delete dry-run error = %v", err)
	}
	if !warehouseEntitySvc.deleteReq.DryRun {
		t.Fatal("expected warehouse delete dry-run flag to be forwarded")
	}
	if !jsonContains(stdout, `"method": "DELETE"`) {
		t.Fatalf("unexpected warehouse delete dry-run output: %s", stdout)
	}

	stdout, _, err = run("warehouse-action", "list",
		"--warehouse-id", "1",
		"--kind", "mm",
		"--product-id", "7",
		"--date-from", "2026-04-01",
		"--date-to", "2026-04-15",
		"--from-warehouse-document", "10",
		"--to-warehouse-document", "11",
		"--warehouse-document-id", "15",
		"--json",
	)
	if err != nil {
		t.Fatalf("warehouse-action list error = %v", err)
	}
	if warehouseActionSvc.listReq.WarehouseID != "1" ||
		warehouseActionSvc.listReq.Kind != "mm" ||
		warehouseActionSvc.listReq.ProductID != "7" ||
		warehouseActionSvc.listReq.DateFrom != "2026-04-01" ||
		warehouseActionSvc.listReq.DateTo != "2026-04-15" ||
		warehouseActionSvc.listReq.FromWarehouseDocument != "10" ||
		warehouseActionSvc.listReq.ToWarehouseDocument != "11" ||
		warehouseActionSvc.listReq.WarehouseDocumentID != "15" {
		t.Fatalf("expected warehouse-action filters to be forwarded, got %#v", warehouseActionSvc.listReq)
	}
	if !jsonContains(stdout, `"warehouse_document_id": 15`) || !jsonContains(stdout, `"quantity": 2`) {
		t.Fatalf("unexpected warehouse-action list output: %s", stdout)
	}

	stdout, _, err = run("warehouse-document", "list", "--json")
	if err != nil {
		t.Fatalf("warehouse-document list error = %v", err)
	}
	if !jsonContains(stdout, `"number": "MM/1/2026"`) {
		t.Fatalf("unexpected warehouse-document list output: %s", stdout)
	}

	stdout, _, err = run("warehouse-document", "get", "--id", "15", "--json")
	if err != nil {
		t.Fatalf("warehouse-document get error = %v", err)
	}
	if warehouseSvc.getReq.ID != "15" {
		t.Fatalf("expected warehouse-document get ID to be forwarded, got %q", warehouseSvc.getReq.ID)
	}
	if !jsonContains(stdout, `"warehouse_actions": [`) {
		t.Fatalf("unexpected warehouse-document get output: %s", stdout)
	}

	stdout, _, err = run("warehouse-document", "create", "--input", `{"kind":"mm","warehouse_actions":[{"product_id":7,"quantity":2,"warehouse2_id":3}]}`, "--json")
	if err != nil {
		t.Fatalf("warehouse-document create error = %v", err)
	}
	if warehouseSvc.createReq.Input["kind"] != "mm" {
		t.Fatalf("expected warehouse-document create input to be parsed, got %#v", warehouseSvc.createReq.Input)
	}
	if !jsonContains(stdout, `"id": 15`) {
		t.Fatalf("unexpected warehouse-document create output: %s", stdout)
	}

	stdout, _, err = runWithInput(`{"invoice_ids":[100,111]}`, "warehouse-document", "update", "--id", "15", "--input", "-", "--json")
	if err != nil {
		t.Fatalf("warehouse-document update error = %v", err)
	}
	if _, ok := warehouseSvc.updateReq.Input["invoice_ids"]; !ok {
		t.Fatalf("expected warehouse-document update stdin input, got %#v", warehouseSvc.updateReq.Input)
	}
	if !jsonContains(stdout, `"invoice_ids": [`) {
		t.Fatalf("unexpected warehouse-document update output: %s", stdout)
	}

	stdout, _, err = run("warehouse-document", "delete", "--id", "15", "--yes", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("warehouse-document delete dry-run error = %v", err)
	}
	if !warehouseSvc.deleteReq.DryRun {
		t.Fatal("expected warehouse-document delete dry-run flag to be forwarded")
	}
	if !jsonContains(stdout, `"method": "DELETE"`) {
		t.Fatalf("unexpected warehouse-document delete dry-run output: %s", stdout)
	}

	stdout, _, err = run("invoice", "get", "--id", "1", "--fields", "id,number", "--json")
	if err != nil {
		t.Fatalf("invoice get error = %v", err)
	}
	if !jsonContains(stdout, `"number": "FV/1"`) || jsonContains(stdout, `"status": "issued"`) {
		t.Fatalf("unexpected invoice get projection output: %s", stdout)
	}

	stdout, _, err = run("invoice", "get", "--id", "1", "--fields", "number,positions[].name", "--json")
	if err != nil {
		t.Fatalf("invoice get nested projection error = %v", err)
	}
	if !jsonContains(stdout, `"positions": [`) || !jsonContains(stdout, `"name": "Produkt A"`) {
		t.Fatalf("unexpected nested projection output: %s", stdout)
	}

	_, _, err = run("invoice", "get", "--id", "1", "--additional-field", "cancel_reason", "--additional-field", "corrected_content_before", "--json")
	if err != nil {
		t.Fatalf("invoice get additional-field error = %v", err)
	}
	if len(invoiceSvc.getReq.AdditionalFields) != 2 || invoiceSvc.getReq.AdditionalFields[0] != "cancel_reason" {
		t.Fatalf("expected invoice additional fields to be forwarded, got %#v", invoiceSvc.getReq.AdditionalFields)
	}
	_, _, err = run("invoice", "get", "--id", "1", "--include", "descriptions", "--correction-positions", "full", "--json")
	if err != nil {
		t.Fatalf("invoice get include/correction-positions error = %v", err)
	}
	if len(invoiceSvc.getReq.Includes) != 1 || invoiceSvc.getReq.Includes[0] != "descriptions" || invoiceSvc.getReq.CorrectionDetails != "full" {
		t.Fatalf("expected invoice include and correction detail flags to be forwarded, got %#v", invoiceSvc.getReq)
	}

	stdout, stderr, err := run("invoice", "list", "--columns", "number,positions[].name")
	if err != nil {
		t.Fatalf("invoice list nested columns error = %v", err)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr for nested columns: %s", stderr)
	}
	if !strings.Contains(stdout, "Produkt A, Produkt B") {
		t.Fatalf("expected joined nested columns in output: %s", stdout)
	}

	stdout, _, err = run("invoice", "get", "--id", "1", "--fields", "number,custom_field", "--json")
	if err != nil {
		t.Fatalf("invoice get undocumented field warning error = %v", err)
	}
	if !jsonContains(stdout, `"code": "undocumented_field_path"`) {
		t.Fatalf("expected undocumented field warning in output: %s", stdout)
	}

	stdout, _, err = run("invoice", "download", "--id", "1", "--json")
	if err != nil {
		t.Fatalf("invoice download error = %v", err)
	}
	if !jsonContains(stdout, `"path": "invoice-1.pdf"`) {
		t.Fatalf("unexpected invoice download output: %s", stdout)
	}

	stdout, _, err = run("invoice", "create", "--gov-save-and-send", "--input", `{"kind":"vat","client_id":1,"positions":[{"product_id":1,"quantity":2}]}`, "--json")
	if err != nil {
		t.Fatalf("invoice create error = %v", err)
	}
	if invoiceSvc.createReq.Input["kind"] != "vat" {
		t.Fatalf("expected invoice create input to be parsed, got %#v", invoiceSvc.createReq.Input)
	}
	if !invoiceSvc.createReq.GovSaveAndSend {
		t.Fatalf("expected invoice create gov-save-and-send to be forwarded, got %#v", invoiceSvc.createReq)
	}
	if !jsonContains(stdout, `"id": 31`) {
		t.Fatalf("unexpected invoice create output: %s", stdout)
	}

	_, _, err = run("invoice", "update", "--id", "31", "--gov-save-and-send", "--input", `{"buyer_name":"Nowa nazwa"}`, "--json")
	if err != nil {
		t.Fatalf("invoice update error = %v", err)
	}
	if invoiceSvc.updateReq.Input["buyer_name"] != "Nowa nazwa" {
		t.Fatalf("expected invoice update input to be parsed, got %#v", invoiceSvc.updateReq.Input)
	}
	if !invoiceSvc.updateReq.GovSaveAndSend {
		t.Fatalf("expected invoice update gov-save-and-send to be forwarded, got %#v", invoiceSvc.updateReq)
	}

	stdout, _, err = run("invoice", "change-status", "--id", "31", "--status", "paid", "--json")
	if err != nil {
		t.Fatalf("invoice change-status error = %v", err)
	}
	if invoiceSvc.changeStatusReq.Status != "paid" {
		t.Fatalf("expected invoice change-status flags to be forwarded, got %#v", invoiceSvc.changeStatusReq)
	}
	if !jsonContains(stdout, `"changed": true`) {
		t.Fatalf("unexpected invoice change-status output: %s", stdout)
	}

	stdout, _, err = run("invoice", "send-email", "--id", "31", "--email-to", "billing@example.com", "--email-pdf", "--json")
	if err != nil {
		t.Fatalf("invoice send-email error = %v", err)
	}
	if len(invoiceSvc.sendEmailReq.EmailTo) == 0 || invoiceSvc.sendEmailReq.EmailTo[0] != "billing@example.com" || !invoiceSvc.sendEmailReq.EmailPDF {
		t.Fatalf("expected invoice send-email flags to be forwarded, got %#v", invoiceSvc.sendEmailReq)
	}
	if !jsonContains(stdout, `"sent": true`) {
		t.Fatalf("unexpected invoice send-email output: %s", stdout)
	}

	stdout, _, err = run("invoice", "send-gov", "--id", "31", "--json")
	if err != nil {
		t.Fatalf("invoice send-gov error = %v", err)
	}
	if invoiceSvc.sendGovReq.ID != "31" {
		t.Fatalf("expected invoice send-gov flags to be forwarded, got %#v", invoiceSvc.sendGovReq)
	}
	if !jsonContains(stdout, `"gov_status": "processing"`) {
		t.Fatalf("unexpected invoice send-gov output: %s", stdout)
	}

	stdout, _, err = run("invoice", "public-link", "--id", "31", "--json")
	if err != nil {
		t.Fatalf("invoice public-link error = %v", err)
	}
	if !jsonContains(stdout, `"pdf_inline_url": "https://acme.fakturownia.pl/invoice/TOKEN-1.pdf?inline=yes"`) {
		t.Fatalf("unexpected invoice public-link output: %s", stdout)
	}

	stdout, _, err = run("invoice", "cancel", "--id", "31", "--yes", "--reason", "Wrong data", "--json")
	if err != nil {
		t.Fatalf("invoice cancel error = %v", err)
	}
	if invoiceSvc.cancelReq.Reason != "Wrong data" || invoiceSvc.cancelReq.ID != "31" {
		t.Fatalf("expected invoice cancel flags to be forwarded, got %#v", invoiceSvc.cancelReq)
	}
	if !jsonContains(stdout, `"cancelled": true`) {
		t.Fatalf("unexpected invoice cancel output: %s", stdout)
	}

	stdout, _, err = runWithInput("attachment-bytes", "invoice", "add-attachment", "--id", "31", "--file", "-", "--name", "scan.pdf", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("invoice add-attachment dry-run error = %v", err)
	}
	if invoiceSvc.addAttachmentReq.Name != "scan.pdf" || string(invoiceSvc.addAttachmentReq.Content) != "attachment-bytes" {
		t.Fatalf("expected invoice attachment input to be forwarded, got %#v", invoiceSvc.addAttachmentReq)
	}
	if !jsonContains(stdout, `"get_credentials"`) {
		t.Fatalf("unexpected invoice add-attachment dry-run output: %s", stdout)
	}

	stdout, _, err = run("invoice", "download-attachments", "--id", "31", "--json")
	if err != nil {
		t.Fatalf("invoice download-attachments error = %v", err)
	}
	if !jsonContains(stdout, `"path": "invoice-31-attachments.zip"`) {
		t.Fatalf("unexpected invoice download-attachments output: %s", stdout)
	}

	stdout, _, err = run("invoice", "download-attachment", "--id", "31", "--kind", "gov", "--json")
	if err != nil {
		t.Fatalf("invoice download-attachment error = %v", err)
	}
	if invoiceSvc.downloadAttachmentReq.ID != "31" || invoiceSvc.downloadAttachmentReq.Kind != "gov" {
		t.Fatalf("expected invoice download-attachment flags to be forwarded, got %#v", invoiceSvc.downloadAttachmentReq)
	}
	if !jsonContains(stdout, `"kind": "gov"`) || !jsonContains(stdout, `"path": "invoice-31-gov.xml"`) {
		t.Fatalf("unexpected invoice download-attachment output: %s", stdout)
	}

	stdout, _, err = run("invoice", "fiscal-print", "--invoice-id", "31", "--invoice-id", "32", "--json")
	if err != nil {
		t.Fatalf("invoice fiscal-print error = %v", err)
	}
	if len(invoiceSvc.fiscalPrintReq.InvoiceIDs) != 2 {
		t.Fatalf("expected fiscal print IDs to be forwarded, got %#v", invoiceSvc.fiscalPrintReq.InvoiceIDs)
	}
	if !jsonContains(stdout, `"submitted": true`) {
		t.Fatalf("unexpected invoice fiscal-print output: %s", stdout)
	}

	stdout, _, err = run("invoice", "delete", "--id", "31", "--yes", "--json")
	if err != nil {
		t.Fatalf("invoice delete error = %v", err)
	}
	if invoiceSvc.deleteReq.ID != "31" {
		t.Fatalf("expected invoice delete confirmation to be forwarded, got %#v", invoiceSvc.deleteReq)
	}
	if !jsonContains(stdout, `"deleted": true`) {
		t.Fatalf("unexpected invoice delete output: %s", stdout)
	}

	stdout, _, err = run("recurring", "list", "--json")
	if err != nil {
		t.Fatalf("recurring list error = %v", err)
	}
	if !jsonContains(stdout, `"name": "Miesięczna"`) {
		t.Fatalf("unexpected recurring list output: %s", stdout)
	}

	stdout, _, err = run("recurring", "create", "--input", `{"name":"Miesięczna","invoice_id":1,"every":"1m"}`, "--json")
	if err != nil {
		t.Fatalf("recurring create error = %v", err)
	}
	if recurringSvc.createReq.Input["name"] != "Miesięczna" {
		t.Fatalf("expected recurring create input to be parsed, got %#v", recurringSvc.createReq.Input)
	}
	if !jsonContains(stdout, `"every": "1m"`) {
		t.Fatalf("unexpected recurring create output: %s", stdout)
	}

	stdout, _, err = run("recurring", "update", "--id", "11", "--input", `{"next_invoice_date":"2026-05-01"}`, "--json")
	if err != nil {
		t.Fatalf("recurring update error = %v", err)
	}
	if recurringSvc.updateReq.Input["next_invoice_date"] != "2026-05-01" {
		t.Fatalf("expected recurring update input to be parsed, got %#v", recurringSvc.updateReq.Input)
	}
	if !jsonContains(stdout, `"next_invoice_date": "2026-05-01"`) {
		t.Fatalf("unexpected recurring update output: %s", stdout)
	}

	stdout, _, err = run("doctor", "run", "--check-release-integrity", "--json")
	if err != nil {
		t.Fatalf("doctor run error = %v", err)
	}
	if !doctorSvc.runReq.CheckReleaseIntegrity {
		t.Fatal("expected --check-release-integrity to be forwarded")
	}
	if !jsonContains(stdout, `"status": "ok"`) {
		t.Fatalf("unexpected doctor output: %s", stdout)
	}

	stdout, _, err = run("self", "update", "--version", "v9.9.9", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("self update dry-run error = %v", err)
	}
	if !selfSvc.updateReq.DryRun || selfSvc.updateReq.TargetVersion != "v9.9.9" {
		t.Fatalf("expected self update dry-run request, got %#v", selfSvc.updateReq)
	}
	if !jsonContains(stdout, `"target_version": "v9.9.9"`) || !jsonContains(stdout, `"dry_run": true`) {
		t.Fatalf("unexpected self update output: %s", stdout)
	}
}

func TestGolden(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		args []string
		file string
	}{
		{name: "category-list-help", args: []string{"category", "list", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "category-list-help.txt")},
		{name: "auth-exchange-help", args: []string{"auth", "exchange", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "auth-exchange-help.txt")},
		{name: "schema-auth-exchange-json", args: []string{"schema", "auth", "exchange", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-auth-exchange.json")},
		{name: "account-create-help", args: []string{"account", "create", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "account-create-help.txt")},
		{name: "account-get-help", args: []string{"account", "get", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "account-get-help.txt")},
		{name: "account-delete-help", args: []string{"account", "delete", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "account-delete-help.txt")},
		{name: "account-unlink-help", args: []string{"account", "unlink", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "account-unlink-help.txt")},
		{name: "schema-account-create-json", args: []string{"schema", "account", "create", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-account-create.json")},
		{name: "schema-account-get-json", args: []string{"schema", "account", "get", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-account-get.json")},
		{name: "schema-account-delete-json", args: []string{"schema", "account", "delete", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-account-delete.json")},
		{name: "schema-account-unlink-json", args: []string{"schema", "account", "unlink", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-account-unlink.json")},
		{name: "department-list-help", args: []string{"department", "list", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "department-list-help.txt")},
		{name: "department-get-help", args: []string{"department", "get", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "department-get-help.txt")},
		{name: "department-create-help", args: []string{"department", "create", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "department-create-help.txt")},
		{name: "department-update-help", args: []string{"department", "update", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "department-update-help.txt")},
		{name: "department-delete-help", args: []string{"department", "delete", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "department-delete-help.txt")},
		{name: "department-set-logo-help", args: []string{"department", "set-logo", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "department-set-logo-help.txt")},
		{name: "schema-department-list-json", args: []string{"schema", "department", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-department-list.json")},
		{name: "schema-department-get-json", args: []string{"schema", "department", "get", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-department-get.json")},
		{name: "schema-department-create-json", args: []string{"schema", "department", "create", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-department-create.json")},
		{name: "schema-department-update-json", args: []string{"schema", "department", "update", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-department-update.json")},
		{name: "schema-department-delete-json", args: []string{"schema", "department", "delete", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-department-delete.json")},
		{name: "schema-department-set-logo-json", args: []string{"schema", "department", "set-logo", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-department-set-logo.json")},
		{name: "issuer-list-help", args: []string{"issuer", "list", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "issuer-list-help.txt")},
		{name: "issuer-get-help", args: []string{"issuer", "get", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "issuer-get-help.txt")},
		{name: "issuer-create-help", args: []string{"issuer", "create", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "issuer-create-help.txt")},
		{name: "issuer-update-help", args: []string{"issuer", "update", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "issuer-update-help.txt")},
		{name: "issuer-delete-help", args: []string{"issuer", "delete", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "issuer-delete-help.txt")},
		{name: "schema-issuer-list-json", args: []string{"schema", "issuer", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-issuer-list.json")},
		{name: "schema-issuer-get-json", args: []string{"schema", "issuer", "get", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-issuer-get.json")},
		{name: "schema-issuer-create-json", args: []string{"schema", "issuer", "create", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-issuer-create.json")},
		{name: "schema-issuer-update-json", args: []string{"schema", "issuer", "update", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-issuer-update.json")},
		{name: "schema-issuer-delete-json", args: []string{"schema", "issuer", "delete", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-issuer-delete.json")},
		{name: "user-create-help", args: []string{"user", "create", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "user-create-help.txt")},
		{name: "schema-user-create-json", args: []string{"schema", "user", "create", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-user-create.json")},
		{name: "webhook-list-help", args: []string{"webhook", "list", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "webhook-list-help.txt")},
		{name: "webhook-get-help", args: []string{"webhook", "get", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "webhook-get-help.txt")},
		{name: "webhook-create-help", args: []string{"webhook", "create", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "webhook-create-help.txt")},
		{name: "webhook-update-help", args: []string{"webhook", "update", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "webhook-update-help.txt")},
		{name: "webhook-delete-help", args: []string{"webhook", "delete", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "webhook-delete-help.txt")},
		{name: "schema-webhook-list-json", args: []string{"schema", "webhook", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-webhook-list.json")},
		{name: "schema-webhook-get-json", args: []string{"schema", "webhook", "get", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-webhook-get.json")},
		{name: "schema-webhook-create-json", args: []string{"schema", "webhook", "create", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-webhook-create.json")},
		{name: "schema-webhook-update-json", args: []string{"schema", "webhook", "update", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-webhook-update.json")},
		{name: "schema-webhook-delete-json", args: []string{"schema", "webhook", "delete", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-webhook-delete.json")},
		{name: "category-get-help", args: []string{"category", "get", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "category-get-help.txt")},
		{name: "category-create-help", args: []string{"category", "create", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "category-create-help.txt")},
		{name: "category-update-help", args: []string{"category", "update", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "category-update-help.txt")},
		{name: "category-delete-help", args: []string{"category", "delete", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "category-delete-help.txt")},
		{name: "schema-category-list-json", args: []string{"schema", "category", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-category-list.json")},
		{name: "schema-category-get-json", args: []string{"schema", "category", "get", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-category-get.json")},
		{name: "schema-category-create-json", args: []string{"schema", "category", "create", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-category-create.json")},
		{name: "schema-category-update-json", args: []string{"schema", "category", "update", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-category-update.json")},
		{name: "schema-category-delete-json", args: []string{"schema", "category", "delete", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-category-delete.json")},
		{name: "client-list-help", args: []string{"client", "list", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "client-list-help.txt")},
		{name: "schema-client-list-json", args: []string{"schema", "client", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-client-list.json")},
		{name: "schema-client-get-json", args: []string{"schema", "client", "get", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-client-get.json")},
		{name: "schema-client-create-json", args: []string{"schema", "client", "create", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-client-create.json")},
		{name: "schema-client-update-json", args: []string{"schema", "client", "update", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-client-update.json")},
		{name: "schema-client-delete-json", args: []string{"schema", "client", "delete", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-client-delete.json")},
		{name: "payment-list-help", args: []string{"payment", "list", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "payment-list-help.txt")},
		{name: "payment-get-help", args: []string{"payment", "get", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "payment-get-help.txt")},
		{name: "payment-create-help", args: []string{"payment", "create", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "payment-create-help.txt")},
		{name: "payment-update-help", args: []string{"payment", "update", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "payment-update-help.txt")},
		{name: "payment-delete-help", args: []string{"payment", "delete", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "payment-delete-help.txt")},
		{name: "schema-payment-list-json", args: []string{"schema", "payment", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-payment-list.json")},
		{name: "schema-payment-get-json", args: []string{"schema", "payment", "get", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-payment-get.json")},
		{name: "schema-payment-create-json", args: []string{"schema", "payment", "create", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-payment-create.json")},
		{name: "schema-payment-update-json", args: []string{"schema", "payment", "update", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-payment-update.json")},
		{name: "schema-payment-delete-json", args: []string{"schema", "payment", "delete", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-payment-delete.json")},
		{name: "bank-account-list-help", args: []string{"bank-account", "list", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "bank-account-list-help.txt")},
		{name: "bank-account-get-help", args: []string{"bank-account", "get", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "bank-account-get-help.txt")},
		{name: "bank-account-create-help", args: []string{"bank-account", "create", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "bank-account-create-help.txt")},
		{name: "bank-account-update-help", args: []string{"bank-account", "update", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "bank-account-update-help.txt")},
		{name: "bank-account-delete-help", args: []string{"bank-account", "delete", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "bank-account-delete-help.txt")},
		{name: "schema-bank-account-list-json", args: []string{"schema", "bank-account", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-bank-account-list.json")},
		{name: "schema-bank-account-get-json", args: []string{"schema", "bank-account", "get", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-bank-account-get.json")},
		{name: "schema-bank-account-create-json", args: []string{"schema", "bank-account", "create", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-bank-account-create.json")},
		{name: "schema-bank-account-update-json", args: []string{"schema", "bank-account", "update", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-bank-account-update.json")},
		{name: "schema-bank-account-delete-json", args: []string{"schema", "bank-account", "delete", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-bank-account-delete.json")},
		{name: "product-list-help", args: []string{"product", "list", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "product-list-help.txt")},
		{name: "schema-product-list-json", args: []string{"schema", "product", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-product-list.json")},
		{name: "schema-product-get-json", args: []string{"schema", "product", "get", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-product-get.json")},
		{name: "schema-product-create-json", args: []string{"schema", "product", "create", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-product-create.json")},
		{name: "schema-product-update-json", args: []string{"schema", "product", "update", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-product-update.json")},
		{name: "price-list-list-help", args: []string{"price-list", "list", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "price-list-list-help.txt")},
		{name: "price-list-get-help", args: []string{"price-list", "get", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "price-list-get-help.txt")},
		{name: "price-list-create-help", args: []string{"price-list", "create", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "price-list-create-help.txt")},
		{name: "price-list-update-help", args: []string{"price-list", "update", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "price-list-update-help.txt")},
		{name: "price-list-delete-help", args: []string{"price-list", "delete", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "price-list-delete-help.txt")},
		{name: "schema-price-list-list-json", args: []string{"schema", "price-list", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-price-list-list.json")},
		{name: "schema-price-list-get-json", args: []string{"schema", "price-list", "get", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-price-list-get.json")},
		{name: "schema-price-list-create-json", args: []string{"schema", "price-list", "create", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-price-list-create.json")},
		{name: "schema-price-list-update-json", args: []string{"schema", "price-list", "update", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-price-list-update.json")},
		{name: "schema-price-list-delete-json", args: []string{"schema", "price-list", "delete", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-price-list-delete.json")},
		{name: "self-update-help", args: []string{"self", "update", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "self-update-help.txt")},
		{name: "schema-self-update-json", args: []string{"schema", "self", "update", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-self-update.json")},
		{name: "invoice-list-help", args: []string{"invoice", "list", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-list-help.txt")},
		{name: "invoice-get-help", args: []string{"invoice", "get", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-get-help.txt")},
		{name: "invoice-download-help", args: []string{"invoice", "download", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-download-help.txt")},
		{name: "invoice-create-help", args: []string{"invoice", "create", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-create-help.txt")},
		{name: "invoice-update-help", args: []string{"invoice", "update", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-update-help.txt")},
		{name: "invoice-delete-help", args: []string{"invoice", "delete", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-delete-help.txt")},
		{name: "invoice-send-email-help", args: []string{"invoice", "send-email", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-send-email-help.txt")},
		{name: "invoice-send-gov-help", args: []string{"invoice", "send-gov", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-send-gov-help.txt")},
		{name: "invoice-change-status-help", args: []string{"invoice", "change-status", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-change-status-help.txt")},
		{name: "invoice-cancel-help", args: []string{"invoice", "cancel", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-cancel-help.txt")},
		{name: "invoice-public-link-help", args: []string{"invoice", "public-link", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-public-link-help.txt")},
		{name: "invoice-add-attachment-help", args: []string{"invoice", "add-attachment", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-add-attachment-help.txt")},
		{name: "invoice-download-attachment-help", args: []string{"invoice", "download-attachment", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-download-attachment-help.txt")},
		{name: "invoice-download-attachments-help", args: []string{"invoice", "download-attachments", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-download-attachments-help.txt")},
		{name: "invoice-fiscal-print-help", args: []string{"invoice", "fiscal-print", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-fiscal-print-help.txt")},
		{name: "schema-list-json", args: []string{"schema", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-list.json")},
		{name: "schema-invoice-list-json", args: []string{"schema", "invoice", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-list.json")},
		{name: "schema-invoice-get-json", args: []string{"schema", "invoice", "get", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-get.json")},
		{name: "schema-invoice-download-json", args: []string{"schema", "invoice", "download", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-download.json")},
		{name: "schema-invoice-create-json", args: []string{"schema", "invoice", "create", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-create.json")},
		{name: "schema-invoice-update-json", args: []string{"schema", "invoice", "update", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-update.json")},
		{name: "schema-invoice-delete-json", args: []string{"schema", "invoice", "delete", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-delete.json")},
		{name: "schema-invoice-send-email-json", args: []string{"schema", "invoice", "send-email", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-send-email.json")},
		{name: "schema-invoice-send-gov-json", args: []string{"schema", "invoice", "send-gov", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-send-gov.json")},
		{name: "schema-invoice-change-status-json", args: []string{"schema", "invoice", "change-status", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-change-status.json")},
		{name: "schema-invoice-cancel-json", args: []string{"schema", "invoice", "cancel", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-cancel.json")},
		{name: "schema-invoice-public-link-json", args: []string{"schema", "invoice", "public-link", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-public-link.json")},
		{name: "schema-invoice-add-attachment-json", args: []string{"schema", "invoice", "add-attachment", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-add-attachment.json")},
		{name: "schema-invoice-download-attachment-json", args: []string{"schema", "invoice", "download-attachment", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-download-attachment.json")},
		{name: "schema-invoice-download-attachments-json", args: []string{"schema", "invoice", "download-attachments", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-download-attachments.json")},
		{name: "schema-invoice-fiscal-print-json", args: []string{"schema", "invoice", "fiscal-print", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-fiscal-print.json")},
		{name: "recurring-list-help", args: []string{"recurring", "list", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "recurring-list-help.txt")},
		{name: "recurring-create-help", args: []string{"recurring", "create", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "recurring-create-help.txt")},
		{name: "recurring-update-help", args: []string{"recurring", "update", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "recurring-update-help.txt")},
		{name: "schema-recurring-list-json", args: []string{"schema", "recurring", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-recurring-list.json")},
		{name: "schema-recurring-create-json", args: []string{"schema", "recurring", "create", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-recurring-create.json")},
		{name: "schema-recurring-update-json", args: []string{"schema", "recurring", "update", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-recurring-update.json")},
		{name: "warehouse-list-help", args: []string{"warehouse", "list", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "warehouse-list-help.txt")},
		{name: "warehouse-get-help", args: []string{"warehouse", "get", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "warehouse-get-help.txt")},
		{name: "warehouse-create-help", args: []string{"warehouse", "create", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "warehouse-create-help.txt")},
		{name: "warehouse-update-help", args: []string{"warehouse", "update", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "warehouse-update-help.txt")},
		{name: "warehouse-delete-help", args: []string{"warehouse", "delete", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "warehouse-delete-help.txt")},
		{name: "schema-warehouse-list-json", args: []string{"schema", "warehouse", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-warehouse-list.json")},
		{name: "schema-warehouse-get-json", args: []string{"schema", "warehouse", "get", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-warehouse-get.json")},
		{name: "schema-warehouse-create-json", args: []string{"schema", "warehouse", "create", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-warehouse-create.json")},
		{name: "schema-warehouse-update-json", args: []string{"schema", "warehouse", "update", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-warehouse-update.json")},
		{name: "schema-warehouse-delete-json", args: []string{"schema", "warehouse", "delete", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-warehouse-delete.json")},
		{name: "warehouse-action-list-help", args: []string{"warehouse-action", "list", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "warehouse-action-list-help.txt")},
		{name: "schema-warehouse-action-list-json", args: []string{"schema", "warehouse-action", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-warehouse-action-list.json")},
		{name: "warehouse-document-list-help", args: []string{"warehouse-document", "list", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "warehouse-document-list-help.txt")},
		{name: "warehouse-document-get-help", args: []string{"warehouse-document", "get", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "warehouse-document-get-help.txt")},
		{name: "warehouse-document-create-help", args: []string{"warehouse-document", "create", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "warehouse-document-create-help.txt")},
		{name: "warehouse-document-update-help", args: []string{"warehouse-document", "update", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "warehouse-document-update-help.txt")},
		{name: "warehouse-document-delete-help", args: []string{"warehouse-document", "delete", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "warehouse-document-delete-help.txt")},
		{name: "schema-warehouse-document-list-json", args: []string{"schema", "warehouse-document", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-warehouse-document-list.json")},
		{name: "schema-warehouse-document-get-json", args: []string{"schema", "warehouse-document", "get", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-warehouse-document-get.json")},
		{name: "schema-warehouse-document-create-json", args: []string{"schema", "warehouse-document", "create", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-warehouse-document-create.json")},
		{name: "schema-warehouse-document-update-json", args: []string{"schema", "warehouse-document", "update", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-warehouse-document-update.json")},
		{name: "schema-warehouse-document-delete-json", args: []string{"schema", "warehouse-document", "delete", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-warehouse-document-delete.json")},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			cmd := NewRootCommand(Dependencies{
				Auth:      &fakeAuthService{},
				Client:    &fakeClientService{},
				Invoice:   &fakeInvoiceService{},
				Product:   &fakeProductService{},
				PriceList: &fakePriceListService{},
				Recurring: &fakeRecurringService{},
				Warehouse: &fakeWarehouseDocumentService{},
				Doctor:    &fakeDoctorService{},
				Self:      &fakeSelfUpdateService{},
				Stdout:    &stdout,
				Stderr:    &stderr,
			})
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Execute() error = %v, stderr = %s", err, stderr.String())
			}

			got := stdout.String()
			if got == "" {
				got = stderr.String()
			}
			assertGolden(t, tc.file, got)
		})
	}
}

func assertGolden(t *testing.T, path, got string) {
	t.Helper()

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	normalizedWant := normalizeGoldenText(string(want))
	normalizedGot := normalizeGoldenText(got)
	if normalizedWant != normalizedGot {
		t.Fatalf("golden mismatch for %s\nwant:\n%s\ngot:\n%s", path, normalizedWant, normalizedGot)
	}
}

func jsonContains(body, needle string) bool {
	return bytes.Contains([]byte(body), []byte(needle))
}

func normalizeGoldenText(value string) string {
	return strings.ReplaceAll(value, "\r\n", "\n")
}

func TestSchemaListUsesRows(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Auth:      &fakeAuthService{},
		Client:    &fakeClientService{},
		Invoice:   &fakeInvoiceService{},
		Product:   &fakeProductService{},
		PriceList: &fakePriceListService{},
		Recurring: &fakeRecurringService{},
		Warehouse: &fakeWarehouseDocumentService{},
		Doctor:    &fakeDoctorService{},
		Self:      &fakeSelfUpdateService{},
		Stdout:    &stdout,
		Stderr:    &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"schema", "list", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var envelope struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(envelope.Data) == 0 {
		t.Fatal("expected schema list data")
	}
	if _, ok := envelope.Data[0]["noun"]; !ok {
		t.Fatalf("expected noun field in schema list row")
	}
}

func TestSchemaInvoiceListExposesKnownFields(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Auth:      &fakeAuthService{},
		Client:    &fakeClientService{},
		Invoice:   &fakeInvoiceService{},
		Product:   &fakeProductService{},
		PriceList: &fakePriceListService{},
		Recurring: &fakeRecurringService{},
		Warehouse: &fakeWarehouseDocumentService{},
		Doctor:    &fakeDoctorService{},
		Self:      &fakeSelfUpdateService{},
		Stdout:    &stdout,
		Stderr:    &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"schema", "invoice", "list", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !jsonContains(stdout.String(), `"path": "positions[].name"`) || !jsonContains(stdout.String(), `"path": "bank_accounts[].bank_account_number"`) {
		t.Fatalf("expected schema invoice list to advertise nested known fields: %s", stdout.String())
	}
}

func TestSchemaProductCreateExposesRequestBodySchema(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Auth:      &fakeAuthService{},
		Client:    &fakeClientService{},
		Invoice:   &fakeInvoiceService{},
		Product:   &fakeProductService{},
		PriceList: &fakePriceListService{},
		Recurring: &fakeRecurringService{},
		Warehouse: &fakeWarehouseDocumentService{},
		Doctor:    &fakeDoctorService{},
		Self:      &fakeSelfUpdateService{},
		Stdout:    &stdout,
		Stderr:    &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"schema", "product", "create", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body := stdout.String()
	if !jsonContains(body, `"path": "gtu_codes[]"`) || !jsonContains(body, `"package_products_details"`) {
		t.Fatalf("expected schema product create to advertise request-body fields: %s", body)
	}
}

func TestSchemaProductListExposesKnownFields(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Auth:      &fakeAuthService{},
		Client:    &fakeClientService{},
		Invoice:   &fakeInvoiceService{},
		Product:   &fakeProductService{},
		PriceList: &fakePriceListService{},
		Recurring: &fakeRecurringService{},
		Warehouse: &fakeWarehouseDocumentService{},
		Doctor:    &fakeDoctorService{},
		Self:      &fakeSelfUpdateService{},
		Stdout:    &stdout,
		Stderr:    &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"schema", "product", "list", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body := stdout.String()
	if !jsonContains(body, `"path": "tag_list[]"`) || !jsonContains(body, `"path": "gtu_codes[]"`) {
		t.Fatalf("expected schema product list to advertise array known fields: %s", body)
	}
}

func TestSchemaPaymentListExposesKnownFields(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Auth:      &fakeAuthService{},
		Client:    &fakeClientService{},
		Invoice:   &fakeInvoiceService{},
		Product:   &fakeProductService{},
		PriceList: &fakePriceListService{},
		Recurring: &fakeRecurringService{},
		Warehouse: &fakeWarehouseDocumentService{},
		Doctor:    &fakeDoctorService{},
		Self:      &fakeSelfUpdateService{},
		Stdout:    &stdout,
		Stderr:    &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"schema", "payment", "list", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body := stdout.String()
	if !jsonContains(body, `"path": "invoices[]"`) || !jsonContains(body, `"requires": [`) {
		t.Fatalf("expected schema payment list to advertise include-backed fields: %s", body)
	}
}

func TestSchemaPaymentCreateExposesRequestBodySchema(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Auth:      &fakeAuthService{},
		Client:    &fakeClientService{},
		Invoice:   &fakeInvoiceService{},
		Product:   &fakeProductService{},
		PriceList: &fakePriceListService{},
		Recurring: &fakeRecurringService{},
		Warehouse: &fakeWarehouseDocumentService{},
		Doctor:    &fakeDoctorService{},
		Self:      &fakeSelfUpdateService{},
		Stdout:    &stdout,
		Stderr:    &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"schema", "payment", "create", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body := stdout.String()
	if !jsonContains(body, `"wrapper_key": "banking_payment"`) || !jsonContains(body, `"path": "invoice_ids[]"`) {
		t.Fatalf("expected schema payment create to advertise banking payment request fields: %s", body)
	}
}

func TestSchemaBankAccountGetExposesKnownFields(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Auth:      &fakeAuthService{},
		Client:    &fakeClientService{},
		Invoice:   &fakeInvoiceService{},
		Product:   &fakeProductService{},
		PriceList: &fakePriceListService{},
		Recurring: &fakeRecurringService{},
		Warehouse: &fakeWarehouseDocumentService{},
		Doctor:    &fakeDoctorService{},
		Self:      &fakeSelfUpdateService{},
		Stdout:    &stdout,
		Stderr:    &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"schema", "bank-account", "get", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body := stdout.String()
	if !jsonContains(body, `"path": "bank_account_number"`) || !jsonContains(body, `"path": "bank_account_version_departments[].show_on_invoice"`) {
		t.Fatalf("expected schema bank-account get to advertise known fields: %s", body)
	}
}

func TestSchemaBankAccountCreateExposesRequestBodySchema(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Auth:      &fakeAuthService{},
		Client:    &fakeClientService{},
		Invoice:   &fakeInvoiceService{},
		Product:   &fakeProductService{},
		PriceList: &fakePriceListService{},
		Recurring: &fakeRecurringService{},
		Warehouse: &fakeWarehouseDocumentService{},
		Doctor:    &fakeDoctorService{},
		Self:      &fakeSelfUpdateService{},
		Stdout:    &stdout,
		Stderr:    &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"schema", "bank-account", "create", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body := stdout.String()
	if !jsonContains(body, `"wrapper_key": "bank_account"`) || !jsonContains(body, `"path": "bank_account_number"`) || !jsonContains(body, `"path": "bank_account_version_departments[].remove"`) {
		t.Fatalf("expected schema bank-account create to advertise request-body fields: %s", body)
	}
}

func TestSchemaCategoryCreateExposesRequestBodySchema(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Auth:      &fakeAuthService{},
		Client:    &fakeClientService{},
		Invoice:   &fakeInvoiceService{},
		Product:   &fakeProductService{},
		PriceList: &fakePriceListService{},
		Recurring: &fakeRecurringService{},
		Warehouse: &fakeWarehouseDocumentService{},
		Doctor:    &fakeDoctorService{},
		Self:      &fakeSelfUpdateService{},
		Stdout:    &stdout,
		Stderr:    &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"schema", "category", "create", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body := stdout.String()
	if !jsonContains(body, `"wrapper_key": "category"`) || !jsonContains(body, `"path": "description"`) {
		t.Fatalf("expected schema category create to advertise category request fields: %s", body)
	}
}

func TestSchemaPriceListGetExposesKnownFields(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Auth:      &fakeAuthService{},
		Client:    &fakeClientService{},
		Invoice:   &fakeInvoiceService{},
		Product:   &fakeProductService{},
		PriceList: &fakePriceListService{},
		Recurring: &fakeRecurringService{},
		Warehouse: &fakeWarehouseDocumentService{},
		Doctor:    &fakeDoctorService{},
		Self:      &fakeSelfUpdateService{},
		Stdout:    &stdout,
		Stderr:    &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"schema", "price-list", "get", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body := stdout.String()
	if !jsonContains(body, `"path": "price_list_positions[]"`) || !strings.Contains(body, "live API behavior") {
		t.Fatalf("expected schema price-list get to advertise positions and live-behavior note: %s", body)
	}
}

func TestSchemaPriceListCreateExposesRequestBodySchema(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Auth:      &fakeAuthService{},
		Client:    &fakeClientService{},
		Invoice:   &fakeInvoiceService{},
		Product:   &fakeProductService{},
		PriceList: &fakePriceListService{},
		Recurring: &fakeRecurringService{},
		Warehouse: &fakeWarehouseDocumentService{},
		Doctor:    &fakeDoctorService{},
		Self:      &fakeSelfUpdateService{},
		Stdout:    &stdout,
		Stderr:    &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"schema", "price-list", "create", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body := stdout.String()
	if !jsonContains(body, `"wrapper_key": "price_list"`) || !jsonContains(body, `"price_list_positions_attributes"`) || !jsonContains(body, `"priceable_id"`) {
		t.Fatalf("expected schema price-list create to advertise upstream request-body fields: %s", body)
	}
}

func TestSchemaWarehouseListExposesKnownFields(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Auth:      &fakeAuthService{},
		Client:    &fakeClientService{},
		Invoice:   &fakeInvoiceService{},
		Product:   &fakeProductService{},
		PriceList: &fakePriceListService{},
		Recurring: &fakeRecurringService{},
		Warehouse: &fakeWarehouseDocumentService{},
		Doctor:    &fakeDoctorService{},
		Self:      &fakeSelfUpdateService{},
		Stdout:    &stdout,
		Stderr:    &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"schema", "warehouse", "list", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body := stdout.String()
	if !jsonContains(body, `"path": "name"`) || !jsonContains(body, `"path": "description"`) {
		t.Fatalf("expected schema warehouse list to advertise known fields: %s", body)
	}
}

func TestSchemaWarehouseCreateExposesRequestBodySchema(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Auth:      &fakeAuthService{},
		Client:    &fakeClientService{},
		Invoice:   &fakeInvoiceService{},
		Product:   &fakeProductService{},
		PriceList: &fakePriceListService{},
		Recurring: &fakeRecurringService{},
		Warehouse: &fakeWarehouseDocumentService{},
		Doctor:    &fakeDoctorService{},
		Self:      &fakeSelfUpdateService{},
		Stdout:    &stdout,
		Stderr:    &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"schema", "warehouse", "create", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body := stdout.String()
	if !jsonContains(body, `"wrapper_key": "warehouse"`) || !jsonContains(body, `"path": "description"`) {
		t.Fatalf("expected schema warehouse create to advertise request-body fields: %s", body)
	}
}

func TestSchemaWarehouseActionListExposesKnownFields(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Auth:      &fakeAuthService{},
		Client:    &fakeClientService{},
		Invoice:   &fakeInvoiceService{},
		Product:   &fakeProductService{},
		PriceList: &fakePriceListService{},
		Recurring: &fakeRecurringService{},
		Warehouse: &fakeWarehouseDocumentService{},
		Doctor:    &fakeDoctorService{},
		Self:      &fakeSelfUpdateService{},
		Stdout:    &stdout,
		Stderr:    &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"schema", "warehouse-action", "list", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body := stdout.String()
	if !jsonContains(body, `"path": "warehouse_document_id"`) || !jsonContains(body, `"path": "warehouse2_id"`) {
		t.Fatalf("expected schema warehouse-action list to advertise known fields: %s", body)
	}
}

func TestSchemaInvoiceCreateExposesRequestBodySchema(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Auth:      &fakeAuthService{},
		Client:    &fakeClientService{},
		Invoice:   &fakeInvoiceService{},
		Product:   &fakeProductService{},
		PriceList: &fakePriceListService{},
		Recurring: &fakeRecurringService{},
		Warehouse: &fakeWarehouseDocumentService{},
		Doctor:    &fakeDoctorService{},
		Self:      &fakeSelfUpdateService{},
		Stdout:    &stdout,
		Stderr:    &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"schema", "invoice", "create", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body := stdout.String()
	if !jsonContains(body, `"wrapper_key": "invoice"`) || !jsonContains(body, `"identify-oss"`) || !jsonContains(body, `"gov-save-and-send"`) || !jsonContains(body, `"additional_catalog_bases"`) || !jsonContains(body, `"path": "gov_corrected_invoice_number"`) || !jsonContains(body, `"path": "settlement_positions[].reason"`) || !jsonContains(body, `"path": "buyer_mass_payment_code"`) || !jsonContains(body, `"path": "bank_accounts[].bank_account_number"`) {
		t.Fatalf("expected schema invoice create to advertise companion options and nested request fields: %s", body)
	}
}

func TestSchemaRecurringCreateExposesRequestBodySchema(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Auth:      &fakeAuthService{},
		Client:    &fakeClientService{},
		Invoice:   &fakeInvoiceService{},
		Product:   &fakeProductService{},
		PriceList: &fakePriceListService{},
		Recurring: &fakeRecurringService{},
		Warehouse: &fakeWarehouseDocumentService{},
		Doctor:    &fakeDoctorService{},
		Self:      &fakeSelfUpdateService{},
		Stdout:    &stdout,
		Stderr:    &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"schema", "recurring", "create", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body := stdout.String()
	if !jsonContains(body, `"wrapper_key": "recurring"`) || !jsonContains(body, `"path": "next_invoice_date"`) || !jsonContains(body, `"path": "buyer_email"`) {
		t.Fatalf("expected schema recurring create to advertise request-body fields: %s", body)
	}
}

func TestSchemaWarehouseDocumentCreateExposesRequestBodySchema(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Auth:      &fakeAuthService{},
		Client:    &fakeClientService{},
		Invoice:   &fakeInvoiceService{},
		Product:   &fakeProductService{},
		PriceList: &fakePriceListService{},
		Recurring: &fakeRecurringService{},
		Warehouse: &fakeWarehouseDocumentService{},
		Doctor:    &fakeDoctorService{},
		Self:      &fakeSelfUpdateService{},
		Stdout:    &stdout,
		Stderr:    &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"schema", "warehouse-document", "create", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body := stdout.String()
	if !jsonContains(body, `"wrapper_key": "warehouse_document"`) || !jsonContains(body, `"path": "warehouse_actions[].product_id"`) || !jsonContains(body, `"path": "invoice_ids[]"`) {
		t.Fatalf("expected schema warehouse-document create to advertise nested request-body fields: %s", body)
	}
}

func TestConfigFlagPassThrough(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	authSvc := &fakeAuthService{}
	cmd := NewRootCommand(Dependencies{
		Auth:      authSvc,
		Client:    &fakeClientService{},
		Invoice:   &fakeInvoiceService{},
		Product:   &fakeProductService{},
		PriceList: &fakePriceListService{},
		Recurring: &fakeRecurringService{},
		Warehouse: &fakeWarehouseDocumentService{},
		Doctor:    &fakeDoctorService{},
		Self:      &fakeSelfUpdateService{},
		Stdout:    &stdout,
		Stderr:    &bytes.Buffer{},
	})
	cmd.SetArgs([]string{"auth", "status", "--config", filepath.Join(t.TempDir(), "config.json"), "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if authSvc.statusReq.ConfigPath == "" {
		t.Fatal("expected config path to be forwarded")
	}
}

func TestGlobalEnvContractDocumented(t *testing.T) {
	t.Parallel()

	spec, ok := FindCommand("invoice", "list")
	if !ok {
		t.Fatal("missing command spec")
	}
	names := make([]string, 0, len(spec.EnvVars))
	for _, env := range spec.EnvVars {
		names = append(names, env.Name)
	}
	expected := []string{"FAKTUROWNIA_PROFILE", "FAKTUROWNIA_URL", "FAKTUROWNIA_API_TOKEN"}
	for _, name := range expected {
		found := false
		for _, got := range names {
			if got == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected env var %s in spec", name)
		}
	}
}
