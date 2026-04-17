import { expect, test } from "bun:test";

import { expectErrorEnvelope, toId } from "./support/assertions";
import { runJson } from "./support/cli";
import { laneName, resourceName } from "./support/context";
import { createResource, deleteResource, getResource, listResource, updateResource } from "./support/resources";

test("issuer CRUD works end-to-end", async () => {
  const lane = laneName("issuer");
  const issuer = await createResource("issuer", {
    name: resourceName(lane, "issuer"),
    tax_no: "1234567890",
  }, lane);

  const issuerId = toId(issuer);
  const fetched = await getResource("issuer", issuerId, [], lane);
  expect((fetched.json.data as Record<string, unknown>).name).toBe(issuer.name);

  const listed = await listResource("issuer", [], lane);
  expect((listed.json.data as Array<Record<string, unknown>>).some((item) => String(item.id) === issuerId)).toBe(true);

  const updated = await updateResource("issuer", issuerId, { tax_no: "0987654321" }, lane);
  expect(updated.tax_no).toBe("0987654321");

  await deleteResource("issuer", issuerId, lane);
  const missing = await runJson(["issuer", "get", "--id", issuerId], { lane, label: "issuer-get-missing", allowFailure: true });
  expect(missing.exitCode).toBe(3);
  expectErrorEnvelope(missing.json, "issuer get");
});
