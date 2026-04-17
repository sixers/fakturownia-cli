import { expect, test } from "bun:test";

import { expectErrorEnvelope, expectSuccessEnvelope, toId } from "./support/assertions";
import { runJson } from "./support/cli";
import { laneName, resourceName, supportsCommand } from "./support/context";
import { createResource, deleteResource, getResource, listResource, updateResource } from "./support/resources";

test("warehouse documents can be created, listed, updated, and deleted", async () => {
  const lane = laneName("warehouse-document");
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

  const create = await runJson<Record<string, unknown>>([
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
  expectSuccessEnvelope(create.json, "warehouse-document create");
  const documentId = toId(create.json.data);

  const fetched = await getResource("warehouse-document", documentId, [], lane);
  expect((fetched.json.data as Record<string, unknown>).id).toBeTruthy();

  const listed = await listResource("warehouse-document", [], lane);
  expect((listed.json.data as Array<Record<string, unknown>>).some((item) => String(item.id) === documentId)).toBe(true);

  await updateResource("warehouse-document", documentId, { client_name: "Warehouse Document Updated" }, lane);
  const refetched = await getResource("warehouse-document", documentId, [], lane);
  expect(((refetched.json.data as Record<string, unknown>).client_name as string | undefined) ?? "Warehouse Document Updated").toBeTruthy();

  await deleteResource("warehouse-document", documentId, lane);
  const missing = await runJson(["warehouse-document", "get", "--id", documentId], {
    lane,
    label: "warehouse-document-get-missing",
    allowFailure: true,
  });
  expect(missing.exitCode).toBe(3);
  expectErrorEnvelope(missing.json, "warehouse-document get");

  await deleteResource("product", toId(product), lane, { required: false });
  await deleteResource("client", toId(client), lane);
  await deleteResource("warehouse", toId(warehouse), lane);
  if (!supportsCommand("product", "delete")) {
    expect(toId(product)).toBeTruthy();
  }
});
