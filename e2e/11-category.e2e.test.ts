import { expect, test } from "bun:test";

import { expectErrorEnvelope, toId } from "./support/assertions";
import { runJson } from "./support/cli";
import { laneName, resourceName } from "./support/context";
import { createResource, deleteResource, getResource, listResource, updateResource } from "./support/resources";

test("category CRUD works end-to-end", async () => {
  const lane = laneName("category");
  const category = await createResource("category", {
    name: resourceName(lane, "category"),
    description: "e2e category",
  }, lane);

  const categoryId = toId(category);
  const fetched = await getResource("category", categoryId, [], lane);
  expect((fetched.json.data as Record<string, unknown>).name).toBe(category.name);

  const listed = await listResource("category", [], lane);
  expect((listed.json.data as Array<Record<string, unknown>>).some((item) => String(item.id) === categoryId)).toBe(true);

  const updated = await updateResource("category", categoryId, { description: "updated category description" }, lane);
  expect(updated.description).toBe("updated category description");

  await deleteResource("category", categoryId, lane);
  const missing = await runJson(["category", "get", "--id", categoryId], { lane, label: "category-get-missing", allowFailure: true });
  expect(missing.exitCode).toBe(3);
  expectErrorEnvelope(missing.json, "category get");
});
