import { expect, test } from "bun:test";

import { expectErrorEnvelope, expectSuccessEnvelope, toId } from "./support/assertions";
import { runJson } from "./support/cli";
import { artifactPath, bankAccountNumber, invoiceRequiredFields, laneName, resourceName, supportsCommand } from "./support/context";
import { createResource, deleteResource, getResource, listResource, updateResource } from "./support/resources";

test("invoice core flows work end-to-end", async () => {
  const lane = laneName("invoice-core");
  const client = await createResource("client", {
    name: resourceName(lane, "client"),
    email: `${resourceName(lane, "billing")}@example.com`,
    tax_no: "9876543210",
    street: "ul. Kliencka 12",
    post_code: "30-001",
    city: "Krakow",
    country: "PL",
  }, lane);
  const product = await createResource("product", {
    name: resourceName(lane, "product"),
    code: resourceName(lane, "product-code"),
    price_gross: 123,
    tax: "23",
  }, lane);
  const bankAccount = await createResource("bank-account", {
    name: resourceName(lane, "bank"),
    bank_name: "Santander Bank Polska",
    bank_account_number: bankAccountNumber(lane),
    bank_currency: "PLN",
  }, lane);

  const invoiceCreate = await runJson<Record<string, unknown>>([
    "invoice",
    "create",
    "--input",
    JSON.stringify({
      kind: "vat",
      client_id: Number(toId(client)),
      ...invoiceRequiredFields(),
      positions: [
        {
          product_id: Number(toId(product)),
          quantity: 2,
        },
      ],
    }),
  ], {
    lane,
    label: "invoice-create",
  });
  expectSuccessEnvelope(invoiceCreate.json, "invoice create");
  const invoiceId = toId(invoiceCreate.json.data);

  const fetched = await getResource("invoice", invoiceId, [], lane);
  expect((fetched.json.data as Record<string, unknown>).id).toBeTruthy();

  const listed = await listResource("invoice", [], lane);
  expect((listed.json.data as Array<Record<string, unknown>>).some((item) => String(item.id) === invoiceId)).toBe(true);

  await updateResource("invoice", invoiceId, { description: "updated by invoice core e2e" }, lane);
  const refetched = await getResource("invoice", invoiceId, [], lane);
  expect((refetched.json.data as Record<string, unknown>).description).toBe("updated by invoice core e2e");

  const links = await runJson<Record<string, unknown>>(["invoice", "public-link", "--id", invoiceId], {
    lane,
    label: "invoice-public-link",
  });
  expectSuccessEnvelope(links.json, "invoice public-link");

  const download = await runJson<Record<string, unknown>>([
    "invoice",
    "download",
    "--id",
    invoiceId,
    "--path",
    artifactPath(lane, "invoice.pdf"),
  ], {
    lane,
    label: "invoice-download",
  });
  expectSuccessEnvelope(download.json, "invoice download");
  expect((download.json.data as Record<string, unknown>).bytes).toBeTruthy();

  const changed = await runJson<Record<string, unknown>>([
    "invoice",
    "change-status",
    "--id",
    invoiceId,
    "--status",
    "sent",
  ], {
    lane,
    label: "invoice-change-status",
  });
  expectSuccessEnvelope(changed.json, "invoice change-status");

  await deleteResource("invoice", invoiceId, lane);
  const missing = await runJson(["invoice", "get", "--id", invoiceId], { lane, label: "invoice-get-missing", allowFailure: true });
  expect(missing.exitCode).toBe(3);
  expectErrorEnvelope(missing.json, "invoice get");

  await deleteResource("bank-account", toId(bankAccount), lane);
  await deleteResource("product", toId(product), lane, { required: false });
  await deleteResource("client", toId(client), lane);
  if (!supportsCommand("product", "delete")) {
    expect(toId(product)).toBeTruthy();
  }
}, 30000);
