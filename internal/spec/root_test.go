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
	"github.com/sixers/fakturownia-cli/internal/client"
	"github.com/sixers/fakturownia-cli/internal/doctor"
	"github.com/sixers/fakturownia-cli/internal/invoice"
	"github.com/sixers/fakturownia-cli/internal/output"
	"github.com/sixers/fakturownia-cli/internal/pricelist"
	"github.com/sixers/fakturownia-cli/internal/product"
	"github.com/sixers/fakturownia-cli/internal/recurring"
	"github.com/sixers/fakturownia-cli/internal/selfupdate"
	"github.com/sixers/fakturownia-cli/internal/transport"
	"github.com/sixers/fakturownia-cli/internal/warehousedocument"
)

type fakeAuthService struct {
	loginReq  auth.LoginRequest
	statusReq auth.StatusRequest
	logoutReq auth.LogoutRequest
}

func (f *fakeAuthService) Login(_ context.Context, req auth.LoginRequest) (*auth.LoginResult, error) {
	f.loginReq = req
	return &auth.LoginResult{Profile: req.Profile, URL: "https://acme.fakturownia.pl", DefaultProfile: req.Profile, TokenStored: true}, nil
}

func (f *fakeAuthService) Status(_ context.Context, req auth.StatusRequest) (*auth.StatusResult, error) {
	f.statusReq = req
	return &auth.StatusResult{Profile: "work", URL: "https://acme.fakturownia.pl", TokenPresent: true}, nil
}

func (f *fakeAuthService) Logout(_ context.Context, req auth.LogoutRequest) (*auth.LogoutResult, error) {
	f.logoutReq = req
	return &auth.LogoutResult{Profile: req.Profile, Removed: true}, nil
}

type fakeInvoiceService struct {
	getReq                 invoice.GetRequest
	downloadReq            invoice.DownloadRequest
	createReq              invoice.CreateRequest
	updateReq              invoice.UpdateRequest
	deleteReq              invoice.DeleteRequest
	sendEmailReq           invoice.SendEmailRequest
	changeStatusReq        invoice.ChangeStatusRequest
	cancelReq              invoice.CancelRequest
	publicLinkReq          invoice.PublicLinkRequest
	addAttachmentReq       invoice.AddAttachmentRequest
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
	if req.DryRun {
		plan := transport.PlanJSONRequest("PUT", "/invoices/"+req.ID+".json", nil, payload)
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
	clientSvc := &fakeClientService{}
	invoiceSvc := &fakeInvoiceService{}
	productSvc := &fakeProductService{}
	priceListSvc := &fakePriceListService{}
	recurringSvc := &fakeRecurringService{}
	warehouseSvc := &fakeWarehouseDocumentService{}
	doctorSvc := &fakeDoctorService{}
	selfSvc := &fakeSelfUpdateService{}

	runWithInput := func(input string, args ...string) (string, string, error) {
		var stdout, stderr bytes.Buffer
		cmd := NewRootCommand(Dependencies{
			Auth:      authSvc,
			Client:    clientSvc,
			Invoice:   invoiceSvc,
			Product:   productSvc,
			PriceList: priceListSvc,
			Recurring: recurringSvc,
			Warehouse: warehouseSvc,
			Doctor:    doctorSvc,
			Self:      selfSvc,
			Stdout:    &stdout,
			Stderr:    &stderr,
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

	stdout, _, err = run("invoice", "create", "--input", `{"kind":"vat","client_id":1,"positions":[{"product_id":1,"quantity":2}]}`, "--json")
	if err != nil {
		t.Fatalf("invoice create error = %v", err)
	}
	if invoiceSvc.createReq.Input["kind"] != "vat" {
		t.Fatalf("expected invoice create input to be parsed, got %#v", invoiceSvc.createReq.Input)
	}
	if !jsonContains(stdout, `"id": 31`) {
		t.Fatalf("unexpected invoice create output: %s", stdout)
	}

	_, _, err = run("invoice", "update", "--id", "31", "--input", `{"buyer_name":"Nowa nazwa"}`, "--json")
	if err != nil {
		t.Fatalf("invoice update error = %v", err)
	}
	if invoiceSvc.updateReq.Input["buyer_name"] != "Nowa nazwa" {
		t.Fatalf("expected invoice update input to be parsed, got %#v", invoiceSvc.updateReq.Input)
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
		{name: "client-list-help", args: []string{"client", "list", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "client-list-help.txt")},
		{name: "schema-client-list-json", args: []string{"schema", "client", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-client-list.json")},
		{name: "schema-client-get-json", args: []string{"schema", "client", "get", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-client-get.json")},
		{name: "schema-client-create-json", args: []string{"schema", "client", "create", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-client-create.json")},
		{name: "schema-client-update-json", args: []string{"schema", "client", "update", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-client-update.json")},
		{name: "schema-client-delete-json", args: []string{"schema", "client", "delete", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-client-delete.json")},
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
		{name: "invoice-change-status-help", args: []string{"invoice", "change-status", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-change-status-help.txt")},
		{name: "invoice-cancel-help", args: []string{"invoice", "cancel", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-cancel-help.txt")},
		{name: "invoice-public-link-help", args: []string{"invoice", "public-link", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-public-link-help.txt")},
		{name: "invoice-add-attachment-help", args: []string{"invoice", "add-attachment", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "invoice-add-attachment-help.txt")},
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
		{name: "schema-invoice-change-status-json", args: []string{"schema", "invoice", "change-status", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-change-status.json")},
		{name: "schema-invoice-cancel-json", args: []string{"schema", "invoice", "cancel", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-cancel.json")},
		{name: "schema-invoice-public-link-json", args: []string{"schema", "invoice", "public-link", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-public-link.json")},
		{name: "schema-invoice-add-attachment-json", args: []string{"schema", "invoice", "add-attachment", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-add-attachment.json")},
		{name: "schema-invoice-download-attachments-json", args: []string{"schema", "invoice", "download-attachments", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-download-attachments.json")},
		{name: "schema-invoice-fiscal-print-json", args: []string{"schema", "invoice", "fiscal-print", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-invoice-fiscal-print.json")},
		{name: "recurring-list-help", args: []string{"recurring", "list", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "recurring-list-help.txt")},
		{name: "recurring-create-help", args: []string{"recurring", "create", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "recurring-create-help.txt")},
		{name: "recurring-update-help", args: []string{"recurring", "update", "--help"}, file: filepath.Join("..", "..", "testdata", "golden", "recurring-update-help.txt")},
		{name: "schema-recurring-list-json", args: []string{"schema", "recurring", "list", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-recurring-list.json")},
		{name: "schema-recurring-create-json", args: []string{"schema", "recurring", "create", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-recurring-create.json")},
		{name: "schema-recurring-update-json", args: []string{"schema", "recurring", "update", "--json"}, file: filepath.Join("..", "..", "testdata", "golden", "schema-recurring-update.json")},
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

	if !jsonContains(stdout.String(), `"path": "positions[].name"`) {
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
	if !jsonContains(body, `"wrapper_key": "invoice"`) || !jsonContains(body, `"identify-oss"`) || !jsonContains(body, `"path": "positions[].product_id"`) || !jsonContains(body, `"path": "settlement_positions[].reason"`) {
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
