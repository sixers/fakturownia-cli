import { expect, test } from "bun:test";

import { expectErrorEnvelope, toId } from "./support/assertions";
import { runJson } from "./support/cli";
import { laneName, resourceName } from "./support/context";
import { createResource, deleteResource, getResource, listResource, updateResource } from "./support/resources";

test("warehouse CRUD works end-to-end", async () => {
  const lane = laneName("warehouse");
  const warehouse = await createResource("warehouse", {
    name: resourceName(lane, "warehouse"),
    kind: null,
    description: "e2e warehouse",
  }, lane);

  const warehouseId = toId(warehouse);
  const fetched = await getResource("warehouse", warehouseId, [], lane);
  expect((fetched.json.data as Record<string, unknown>).name).toBe(warehouse.name);

  const listed = await listResource("warehouse", [], lane);
  expect((listed.json.data as Array<Record<string, unknown>>).some((item) => String(item.id) === warehouseId)).toBe(true);

  await updateResource("warehouse", warehouseId, { description: "updated e2e warehouse" }, lane);
  const refetched = await getResource("warehouse", warehouseId, [], lane);
  expect((refetched.json.data as Record<string, unknown>).description).toBe("updated e2e warehouse");

  await deleteResource("warehouse", warehouseId, lane);
  const missing = await runJson(["warehouse", "get", "--id", warehouseId], { lane, label: "warehouse-get-missing", allowFailure: true });
  expect(missing.exitCode).toBe(3);
  expectErrorEnvelope(missing.json, "warehouse get");
});
