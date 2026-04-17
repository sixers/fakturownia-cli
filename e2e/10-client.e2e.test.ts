import { expect, test } from "bun:test";

import { expectErrorEnvelope, expectSuccessEnvelope, toId } from "./support/assertions";
import { runJson } from "./support/cli";
import { compactToken, laneName, resourceName } from "./support/context";
import { createResource, deleteResource, getResource, listResource, updateResource } from "./support/resources";

test("client CRUD works end-to-end", async () => {
  const lane = laneName("client");
  const externalId = compactToken("client-ext", 20);
  const client = await createResource("client", {
    name: resourceName(lane, "client"),
    email: `${resourceName(lane, "billing")}@example.com`,
    external_id: externalId,
    city: "Krakow",
    country: "PL",
  }, lane);

  const clientId = toId(client);
  const fetched = await getResource("client", clientId, [], lane);
  expectSuccessEnvelope(fetched.json, "client get");
  expect((fetched.json.data as Record<string, unknown>).external_id).toBe(externalId);

  const byExternalId = await runJson<Record<string, unknown>>(["client", "get", "--external-id", externalId], {
    lane,
    label: "client-get-external",
  });
  expectSuccessEnvelope(byExternalId.json, "client get");
  expect(toId(byExternalId.json.data)).toBe(clientId);

  const listed = await listResource("client", [], lane);
  expect((listed.json.data as Array<Record<string, unknown>>).some((item) => String(item.id) === clientId)).toBe(true);

  await updateResource("client", clientId, { city: "Warsaw", note: "updated by e2e" }, lane);
  const refetched = await getResource("client", clientId, [], lane);
  expect((refetched.json.data as Record<string, unknown>).city).toBe("Warsaw");

  await deleteResource("client", clientId, lane);
  const missing = await runJson(["client", "get", "--id", clientId], { lane, label: "client-get-missing", allowFailure: true });
  expect(missing.exitCode).toBe(3);
  expectErrorEnvelope(missing.json, "client get");
});
