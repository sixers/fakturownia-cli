import { expect, test } from "bun:test";

import { expectErrorEnvelope, expectSuccessEnvelope, toId } from "./support/assertions";
import { runJson } from "./support/cli";
import { invoiceRequiredFields, laneName } from "./support/context";
import { deleteResource, getResource, listResource, updateResource } from "./support/resources";

test("payment CRUD works end-to-end", async () => {
  const lane = laneName("payment");
  const invoiceCreate = await runJson<Record<string, unknown>>([
    "invoice",
    "create",
    "--input",
    JSON.stringify({
      kind: "vat",
      ...invoiceRequiredFields({ buyer_name: "Payment Test Buyer" }),
      buyer_name: "Payment Test Buyer",
      positions: [
        { name: "Payment test item", quantity: 1, total_price_gross: 200, tax: 23 },
      ],
    }),
  ], {
    lane,
    label: "invoice-create",
  });
  expectSuccessEnvelope(invoiceCreate.json, "invoice create");
  const invoiceId = toId(invoiceCreate.json.data);

  const paymentCreate = await runJson<Record<string, unknown>>([
    "payment",
    "create",
    "--input",
    JSON.stringify({
      name: "Payment 001",
      price: 200,
      invoice_id: Number(invoiceId),
      paid: true,
      kind: "api",
    }),
  ], {
    lane,
    label: "payment-create",
  });
  expectSuccessEnvelope(paymentCreate.json, "payment create");
  const paymentId = toId(paymentCreate.json.data);

  const fetched = await getResource("payment", paymentId, [], lane);
  expect((fetched.json.data as Record<string, unknown>).invoice_id).toBe(Number(invoiceId));

  const listed = await listResource("payment", [], lane);
  expect((listed.json.data as Array<Record<string, unknown>>).some((item) => String(item.id) === paymentId)).toBe(true);

  await updateResource("payment", paymentId, { name: "Payment 001 Updated" }, lane);
  const refetched = await getResource("payment", paymentId, [], lane);
  expect((refetched.json.data as Record<string, unknown>).name).toBe("Payment 001 Updated");

  await deleteResource("payment", paymentId, lane);
  const missing = await runJson(["payment", "get", "--id", paymentId], { lane, label: "payment-get-missing", allowFailure: true });
  expect(missing.exitCode).toBe(3);
  expectErrorEnvelope(missing.json, "payment get");

  await deleteResource("invoice", invoiceId, lane);
});
