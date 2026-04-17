import { expect, test } from "bun:test";

import { expectSuccessEnvelope, toId } from "./support/assertions";
import { runJson } from "./support/cli";
import { invoiceRequiredFields, laneName, laneRecipient, resourceName } from "./support/context";
import { waitForEmail } from "./support/mailtest";
import { createResource, deleteResource } from "./support/resources";

test("invoice emails are delivered to MailtestAPI", async () => {
  const lane = laneName("invoice-email");
  const recipient = laneRecipient(lane);
  const client = await createResource("client", {
    name: resourceName(lane, "client"),
    email: recipient,
    tax_no: "9876543210",
    street: "ul. Mailowa 7",
    post_code: "30-002",
    city: "Krakow",
    country: "PL",
  }, lane);

  const invoiceCreate = await runJson<Record<string, unknown>>([
    "invoice",
    "create",
    "--input",
    JSON.stringify({
      kind: "vat",
      client_id: Number(toId(client)),
      buyer_email: recipient,
      ...invoiceRequiredFields({ buyer_email: recipient }),
      positions: [
        { name: "Email test item", quantity: 1, total_price_gross: 50, tax: 23 },
      ],
    }),
  ], {
    lane,
    label: "invoice-create",
  });
  expectSuccessEnvelope(invoiceCreate.json, "invoice create");
  const invoiceId = toId(invoiceCreate.json.data);

  const send = await runJson<Record<string, unknown>>([
    "invoice",
    "send-email",
    "--id",
    invoiceId,
    "--email-to",
    recipient,
    "--update-buyer-email",
    "--email-pdf",
  ], {
    lane,
    label: "invoice-send-email",
  });
  expectSuccessEnvelope(send.json, "invoice send-email");
  expect((send.json.data as Record<string, unknown>).status).toBe("ok");
  expect(String((send.json.data as Record<string, unknown>).message ?? "")).toContain(recipient);

  const email = await waitForEmail(recipient);
  expect(email.recipient).toBe(recipient);
  expect(email.body).toBeTruthy();
  expect(email.body?.attachments.length ?? 0).toBeGreaterThan(0);

  await deleteResource("invoice", invoiceId, lane);
  await deleteResource("client", toId(client), lane);
}, 30000);
