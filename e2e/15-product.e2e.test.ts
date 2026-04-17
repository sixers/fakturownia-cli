import { expect, test } from "bun:test";

import { expectErrorEnvelope, toId } from "./support/assertions";
import { runJson } from "./support/cli";
import { laneName, resourceName, supportsCommand } from "./support/context";
import { createResource, deleteResource, getResource, listResource, updateResource } from "./support/resources";

test("product CRUD and package products work end-to-end", async () => {
  const lane = laneName("product");
  const simple = await createResource("product", {
    name: resourceName(lane, "simple"),
    code: resourceName(lane, "simple-code"),
    price_gross: 123,
    tax: "23",
    quantity_unit: "szt",
  }, lane);

  const child = await createResource("product", {
    name: resourceName(lane, "child"),
    code: resourceName(lane, "child-code"),
    price_gross: 50,
    tax: "23",
    quantity_unit: "szt",
  }, lane);

  const childId = toId(child);
  const packageProduct = await createResource("product", {
    name: resourceName(lane, "package"),
    code: resourceName(lane, "package-code"),
    package: "1",
    package_products_details: {
      "0": {
        id: Number(childId),
        quantity: 1,
      },
    },
  }, lane);

  const simpleId = toId(simple);
  const packageId = toId(packageProduct);

  const fetched = await getResource("product", simpleId, [], lane);
  expect((fetched.json.data as Record<string, unknown>).name).toBe(simple.name);

  const listed = await listResource("product", [], lane);
  expect((listed.json.data as Array<Record<string, unknown>>).some((item) => String(item.id) === simpleId)).toBe(true);

  await updateResource("product", simpleId, { price_gross: 150 }, lane);
  const refetched = await getResource("product", simpleId, [], lane);
  expect(String((refetched.json.data as Record<string, unknown>).price_gross ?? "")).toContain("150");

  await deleteResource("product", packageId, lane, { required: false });
  await deleteResource("product", childId, lane, { required: false });
  await deleteResource("product", simpleId, lane, { required: false });

  if (supportsCommand("product", "delete")) {
    const missing = await runJson(["product", "get", "--id", simpleId], { lane, label: "product-get-missing", allowFailure: true });
    expect(missing.exitCode).toBe(3);
    expectErrorEnvelope(missing.json, "product get");
  }
});
