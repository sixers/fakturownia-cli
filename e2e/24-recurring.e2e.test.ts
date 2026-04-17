import { expect, test } from "bun:test";

import { expectErrorEnvelope, expectSuccessEnvelope, toId } from "./support/assertions";
import { runJson } from "./support/cli";
import { invoiceRequiredFields, laneName, laneRecipient, supportsCommand } from "./support/context";
import { deleteResource, listResource, updateResource } from "./support/resources";

test("recurring definitions can be created, listed, updated, and deleted", async () => {
  const lane = laneName("recurring");
  const invoiceCreate = await runJson<Record<string, unknown>>([
    "invoice",
    "create",
    "--input",
    JSON.stringify({
      kind: "vat",
      ...invoiceRequiredFields({ buyer_name: "Recurring Test Buyer" }),
      buyer_name: "Recurring Test Buyer",
      positions: [
        { name: "Recurring test item", quantity: 1, total_price_gross: 100, tax: 23 },
      ],
    }),
  ], {
    lane,
    label: "invoice-create",
  });
  expectSuccessEnvelope(invoiceCreate.json, "invoice create");
  const invoiceId = Number(toId(invoiceCreate.json.data));

  const recurringCreate = await runJson<Record<string, unknown>>([
    "recurring",
    "create",
    "--input",
    JSON.stringify({
      name: "Recurring E2E",
      invoice_id: invoiceId,
      start_date: "2026-04-17",
      every: "1m",
      send_email: false,
      buyer_email: laneRecipient(lane, "recurring"),
    }),
  ], {
    lane,
    label: "recurring-create",
    allowFailure: true,
  });
  if (recurringCreate.exitCode !== 0) {
    expectErrorEnvelope(recurringCreate.json, "recurring create");
    expect((recurringCreate.json.errors[0]?.message ?? "")).toContain("decode upstream JSON response");
    console.warn("recurring create soft skip: upstream returned a non-JSON login page for this account");
    await deleteResource("invoice", invoiceId, lane);
    return;
  }
  expectSuccessEnvelope(recurringCreate.json, "recurring create");
  const recurringId = toId(recurringCreate.json.data);

  const listed = await listResource("recurring", [], lane);
  expect((listed.json.data as Array<Record<string, unknown>>).some((item) => String(item.id) === recurringId)).toBe(true);

  await updateResource("recurring", recurringId, { next_invoice_date: "2026-05-01" }, lane);
  const relisted = await listResource("recurring", [], lane);
  const recurring = (relisted.json.data as Array<Record<string, unknown>>).find((item) => String(item.id) === recurringId);
  expect(recurring?.next_invoice_date).toBe("2026-05-01");

  await deleteResource("recurring", recurringId, lane, { required: false });
  await deleteResource("invoice", invoiceId, lane);
  if (!supportsCommand("recurring", "delete")) {
    expect(recurringId).toBeTruthy();
  }
});
