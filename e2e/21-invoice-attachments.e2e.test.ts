import { expect, test } from "bun:test";

import { expectSuccessEnvelope, toId } from "./support/assertions";
import { runJson } from "./support/cli";
import { artifactPath, getContext, invoiceRequiredFields, laneName } from "./support/context";
import { deleteResource } from "./support/resources";

test("invoice attachments can be uploaded and downloaded", async () => {
  const lane = laneName("invoice-attachments");
  const invoiceCreate = await runJson<Record<string, unknown>>([
    "invoice",
    "create",
    "--input",
    JSON.stringify({
      kind: "vat",
      ...invoiceRequiredFields({ buyer_name: "Attachment Test Buyer" }),
      buyer_name: "Attachment Test Buyer",
      positions: [
        { name: "Attachment test item", quantity: 1, total_price_gross: 10, tax: 23 },
      ],
    }),
  ], {
    lane,
    label: "invoice-create",
  });
  expectSuccessEnvelope(invoiceCreate.json, "invoice create");
  const invoiceId = toId(invoiceCreate.json.data);

  const addAttachment = await runJson<Record<string, unknown>>([
    "invoice",
    "add-attachment",
    "--id",
    invoiceId,
    "--file",
    getContext().fixtures.attachmentPdfPath,
  ], {
    lane,
    label: "invoice-add-attachment",
  });
  expectSuccessEnvelope(addAttachment.json, "invoice add-attachment");

  const customAttachment = await runJson<Record<string, unknown>>([
    "invoice",
    "download-attachment",
    "--id",
    invoiceId,
    "--kind",
    "custom",
    "--path",
    artifactPath(lane, "attachment.bin"),
  ], {
    lane,
    label: "invoice-download-attachment",
  });
  expectSuccessEnvelope(customAttachment.json, "invoice download-attachment");
  expect((customAttachment.json.data as Record<string, unknown>).bytes).toBeTruthy();

  const allAttachments = await runJson<Record<string, unknown>>([
    "invoice",
    "download-attachments",
    "--id",
    invoiceId,
    "--path",
    artifactPath(lane, "attachments.zip"),
  ], {
    lane,
    label: "invoice-download-attachments",
  });
  expectSuccessEnvelope(allAttachments.json, "invoice download-attachments");
  expect((allAttachments.json.data as Record<string, unknown>).bytes).toBeTruthy();

  await deleteResource("invoice", invoiceId, lane);
});
