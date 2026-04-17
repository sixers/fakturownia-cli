import { type CliRunResult, runCli, runJson } from "./cli";
import { registerDelete } from "./cleanup";
import { invoiceRequiredFields, resourceName, supportsCommand } from "./context";
import { expectSuccessEnvelope, toId } from "./assertions";
import { createResource } from "./resources";

export interface InvoiceFixture {
  lane: string;
  client: Record<string, unknown>;
  clientId: string;
  product: Record<string, unknown>;
  productId: string;
  invoice: Record<string, unknown>;
  invoiceId: string;
}

export interface InvoiceFixtureOptions {
  clientInput?: Record<string, unknown>;
  productInput?: Record<string, unknown>;
  invoiceInput?: Record<string, unknown>;
}

export async function createInvoiceFixture(lane: string, options: InvoiceFixtureOptions = {}): Promise<InvoiceFixture> {
  const client = await createResource("client", {
    name: resourceName(lane, "client"),
    email: `${resourceName(lane, "billing")}@example.com`,
    tax_no: "9876543210",
    street: "ul. Kliencka 12",
    post_code: "30-001",
    city: "Krakow",
    country: "PL",
    ...options.clientInput,
  }, lane);
  const clientId = toId(client);

  const product = await createResource("product", {
    name: resourceName(lane, "product"),
    code: resourceName(lane, "product-code"),
    price_gross: 123,
    tax: "23",
    ...options.productInput,
  }, lane);
  const productId = toId(product);

  const invoiceCreate = await runJson<Record<string, unknown>>([
    "invoice",
    "create",
    "--input",
    JSON.stringify({
      kind: "vat",
      client_id: Number(clientId),
      ...invoiceRequiredFields(),
      positions: [
        {
          product_id: Number(productId),
          quantity: 1,
        },
      ],
      ...options.invoiceInput,
    }),
  ], {
    lane,
    label: "invoice-create",
  });
  expectSuccessEnvelope(invoiceCreate.json, "invoice create");

  const invoice = invoiceCreate.json.data as Record<string, unknown>;
  const invoiceId = toId(invoice);
  await registerDelete(lane, "invoice", invoiceId);

  return {
    lane,
    client,
    clientId,
    product,
    productId,
    invoice,
    invoiceId,
  };
}

export async function cleanupInvoiceFixture(fixture: InvoiceFixture, extraInvoiceIds: Array<string | undefined> = []): Promise<void> {
  for (const invoiceId of [...extraInvoiceIds, fixture.invoiceId].filter(Boolean)) {
    await deleteBestEffort(["invoice", "delete", "--id", String(invoiceId), "--yes", "--json"], fixture.lane, `invoice-delete-${invoiceId}`);
  }

  if (supportsCommand("product", "delete")) {
    await deleteBestEffort(["product", "delete", "--id", fixture.productId, "--yes", "--json"], fixture.lane, `product-delete-${fixture.productId}`);
  }

  await deleteBestEffort(["client", "delete", "--id", fixture.clientId, "--yes", "--json"], fixture.lane, `client-delete-${fixture.clientId}`);
}

async function deleteBestEffort(args: string[], lane: string, label: string): Promise<CliRunResult> {
  return await runCli(args, {
    allowFailure: true,
    lane,
    label,
  });
}
