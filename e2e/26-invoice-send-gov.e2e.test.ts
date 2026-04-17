import { expect, test } from "bun:test";

import { expectSuccessEnvelope } from "./support/assertions";
import { runCli, runJson } from "./support/cli";
import { laneName } from "./support/context";
import { cleanupInvoiceFixture, createInvoiceFixture } from "./support/invoice-fixtures";
import { getResource } from "./support/resources";

test("invoice send-gov supports dry-run, raw, and read-back verification", async () => {
  const lane = laneName("invoice-send-gov");
  const fixture = await createInvoiceFixture(lane);

  try {
    const dryRun = await runJson<Record<string, unknown>>([
      "invoice",
      "send-gov",
      "--id",
      fixture.invoiceId,
      "--dry-run",
    ], {
      lane,
      label: "invoice-send-gov-dry-run",
    });
    expectSuccessEnvelope(dryRun.json, "invoice send-gov");
    expect(dryRun.json.data).toMatchObject({
      method: "GET",
      path: `/invoices/${fixture.invoiceId}.json`,
      body: null,
      query: {
        send_to_ksef: ["yes"],
      },
    });

    const sent = await runJson<Record<string, unknown>>([
      "invoice",
      "send-gov",
      "--id",
      fixture.invoiceId,
    ], {
      lane,
      label: "invoice-send-gov",
    });
    expectSuccessEnvelope(sent.json, "invoice send-gov");
    expect(sent.json.data).toHaveProperty("gov_status");

    const raw = await runCli([
      "invoice",
      "send-gov",
      "--id",
      fixture.invoiceId,
      "--raw",
    ], {
      lane,
      label: "invoice-send-gov-raw",
    });
    expect(raw.exitCode).toBe(0);
    expect(JSON.parse(raw.stdout)).toHaveProperty("gov_status");

    const fetched = await getResource("invoice", fixture.invoiceId, [
      "--additional-field",
      "gov_status",
      "--additional-field",
      "gov_id",
      "--additional-field",
      "gov_send_date",
      "--additional-field",
      "gov_error_messages",
    ], lane);
    const invoice = fetched.json.data as Record<string, unknown>;
    expect(String(invoice.id)).toBe(fixture.invoiceId);
    expect(invoice).toHaveProperty("gov_status");
    expect(invoice).toHaveProperty("gov_id");
    expect(invoice).toHaveProperty("gov_send_date");
    expect(invoice).toHaveProperty("gov_error_messages");
  } finally {
    await cleanupInvoiceFixture(fixture);
  }
}, 30000);
