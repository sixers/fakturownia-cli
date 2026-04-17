import { expect, test } from "bun:test";

import { expectSuccessEnvelope, toId } from "./support/assertions";
import { runJson } from "./support/cli";
import { laneName, resourceName } from "./support/context";
import { cleanupInvoiceFixture, createInvoiceFixture } from "./support/invoice-fixtures";
import { getResource } from "./support/resources";

test("invoice cancel stores the cancellation reason on an older invoice", async () => {
  const lane = laneName("invoice-cancel");
  const fixture = await createInvoiceFixture(lane, {
    invoiceInput: {
      description: resourceName(lane, "cancel-target"),
    },
  });

  let newerInvoiceId: string | undefined;
  try {
    const newerInvoice = await runJson<Record<string, unknown>>([
      "invoice",
      "create",
      "--input",
      JSON.stringify({
        kind: "vat",
        client_id: Number(fixture.clientId),
        description: resourceName(lane, "newer"),
        seller_name: "E2E Seller Sp. z o.o.",
        seller_tax_no: "5252445767",
        seller_street: "ul. Przykładowa 10",
        seller_post_code: "00-001",
        seller_city: "Warszawa",
        buyer_company: true,
        buyer_tax_no: "9876543210",
        buyer_country: "PL",
        positions: [
          {
            product_id: Number(fixture.productId),
            quantity: 1,
          },
        ],
      }),
    ], {
      lane,
      label: "invoice-create-newer",
    });
    expectSuccessEnvelope(newerInvoice.json, "invoice create");
    newerInvoiceId = toId(newerInvoice.json.data);

    const reason = resourceName(lane, "reason");
    const dryRun = await runJson<Record<string, unknown>>([
      "invoice",
      "cancel",
      "--id",
      fixture.invoiceId,
      "--yes",
      "--reason",
      reason,
      "--dry-run",
    ], {
      lane,
      label: "invoice-cancel-dry-run",
    });
    expectSuccessEnvelope(dryRun.json, "invoice cancel");
    expect(dryRun.json.data).toMatchObject({
      method: "POST",
      path: "/invoices/cancel.json",
      body: {
        cancel_invoice_id: fixture.invoiceId,
        cancel_reason: reason,
      },
    });

    const cancelled = await runJson<Record<string, unknown>>([
      "invoice",
      "cancel",
      "--id",
      fixture.invoiceId,
      "--yes",
      "--reason",
      reason,
    ], {
      lane,
      label: "invoice-cancel",
    });
    expectSuccessEnvelope(cancelled.json, "invoice cancel");
    expect(cancelled.json.data).toMatchObject({
      code: "success",
      message: expect.any(String),
    });

    const fetched = await getResource("invoice", fixture.invoiceId, ["--additional-field", "cancel_reason"], lane);
    expect((fetched.json.data as Record<string, unknown>).kind).toBe("canceled");
    expect((fetched.json.data as Record<string, unknown>).cancel_reason).toBe(reason);
  } finally {
    await cleanupInvoiceFixture(fixture, [newerInvoiceId]);
  }
}, 30000);
