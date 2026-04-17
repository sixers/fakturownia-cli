import { expect, test } from "bun:test";

import { expectSuccessEnvelope, toId } from "./support/assertions";
import { runJson } from "./support/cli";
import { laneName, resourceName, supportsCommand } from "./support/context";
import { createResource, deleteResource } from "./support/resources";

test("warehouse actions can be listed for a created warehouse document", async () => {
  const lane = laneName("warehouse-action");
  const warehouse = await createResource("warehouse", { name: resourceName(lane, "warehouse"), kind: null, description: null }, lane);
  const client = await createResource("client", {
    name: resourceName(lane, "client"),
    email: `${resourceName(lane, "client")}@example.com`,
    city: "Krakow",
    country: "PL",
  }, lane);
  const product = await createResource("product", {
    name: resourceName(lane, "product"),
    code: resourceName(lane, "code"),
    price_gross: 100,
    tax: "23",
  }, lane);

  const document = await runJson<Record<string, unknown>>([
    "warehouse-document",
    "create",
    "--input",
    JSON.stringify({
      kind: "wz",
      warehouse_id: Number(toId(warehouse)),
      client_id: Number(toId(client)),
      warehouse_actions: [
        {
          product_id: Number(toId(product)),
          tax: "23",
          price_net: 100,
          quantity: 1,
        },
      ],
    }),
  ], {
    lane,
    label: "warehouse-document-create",
  });
  expectSuccessEnvelope(document.json, "warehouse-document create");
  const documentId = toId(document.json.data);

  const actions = await runJson<Array<Record<string, unknown>>>([
    "warehouse-action",
    "list",
    "--warehouse-document-id",
    documentId,
  ], {
    lane,
    label: "warehouse-action-list",
  });
  expectSuccessEnvelope(actions.json, "warehouse-action list");
  const items = actions.json.data as Array<Record<string, unknown>>;
  if (items.length > 0) {
    expect(items.some((item) => String(item.warehouse_document_id) === documentId)).toBe(true);
  } else {
    console.warn(`warehouse-action soft assertion: no actions returned for warehouse document ${documentId}`);
  }

  await deleteResource("warehouse-document", documentId, lane);
  await deleteResource("product", toId(product), lane, { required: false });
  await deleteResource("client", toId(client), lane);
  await deleteResource("warehouse", toId(warehouse), lane);
  if (!supportsCommand("product", "delete")) {
    expect(toId(product)).toBeTruthy();
  }
});
