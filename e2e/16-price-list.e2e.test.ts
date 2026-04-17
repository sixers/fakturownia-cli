import { expect, test } from "bun:test";

import { expectErrorEnvelope, toId } from "./support/assertions";
import { runJson } from "./support/cli";
import { laneName, resourceName, supportsCommand } from "./support/context";
import { createResource, deleteResource, getResource, listResource, updateResource } from "./support/resources";

test("price-list CRUD works end-to-end", async () => {
  const lane = laneName("price-list");
  const product = await createResource("product", {
    name: resourceName(lane, "product"),
    code: resourceName(lane, "product-code"),
    price_gross: 33.16,
    tax: "23",
  }, lane);

  const productId = Number(toId(product));
  const priceList = await createResource("price-list", {
    name: resourceName(lane, "price-list"),
    currency: "PLN",
    description: "e2e price list",
    price_list_positions_attributes: {
      "0": {
        priceable_id: productId,
        priceable_name: product.name,
        priceable_type: "Product",
        use_percentage: "0",
        price_net: "26.96",
        price_gross: "33.16",
        use_tax: "1",
        tax: "23",
      },
    },
  }, lane);

  const priceListId = toId(priceList);
  const fetched = await getResource("price-list", priceListId, [], lane);
  expect((fetched.json.data as Record<string, unknown>).name).toBe(priceList.name);

  const listed = await listResource("price-list", [], lane);
  expect((listed.json.data as Array<Record<string, unknown>>).some((item) => String(item.id) === priceListId)).toBe(true);

  await updateResource("price-list", priceListId, { description: "updated e2e price list" }, lane);
  const refetched = await getResource("price-list", priceListId, [], lane);
  expect((refetched.json.data as Record<string, unknown>).description).toBe("updated e2e price list");

  await deleteResource("price-list", priceListId, lane);
  const missing = await runJson(["price-list", "get", "--id", priceListId], {
    lane,
    label: "price-list-get-missing",
    allowFailure: true,
  });
  expect(missing.exitCode).toBe(3);
  expectErrorEnvelope(missing.json, "price-list get");

  await deleteResource("product", productId, lane, { required: false });
  if (!supportsCommand("product", "delete")) {
    expect(productId).toBeGreaterThan(0);
  }
});
