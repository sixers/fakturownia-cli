import { expect, test } from "bun:test";

import { expectErrorEnvelope, toId } from "./support/assertions";
import { runJson } from "./support/cli";
import { bankAccountNumber, compactToken, laneName } from "./support/context";
import { createResource, deleteResource, getResource, listResource, updateResource } from "./support/resources";

test("bank-account CRUD works end-to-end", async () => {
  const lane = laneName("bank-account");
  const account = await createResource("bank-account", {
    name: compactToken("bank-account", 20),
    bank_name: "Santander Bank Polska",
    bank_account_number: bankAccountNumber(lane),
    bank_currency: "PLN",
  }, lane);

  const accountId = toId(account);
  const fetched = await getResource("bank-account", accountId, [], lane);
  expect((fetched.json.data as Record<string, unknown>).bank_currency).toBe("PLN");

  const listed = await listResource("bank-account", [], lane);
  expect((listed.json.data as Array<Record<string, unknown>>).some((item) => String(item.id) === accountId)).toBe(true);

  const updatedName = compactToken("bank-updated", 20);
  await updateResource("bank-account", accountId, { name: updatedName }, lane);
  const refetched = await getResource("bank-account", accountId, [], lane);
  expect((refetched.json.data as Record<string, unknown>).name).toBe(updatedName);

  await deleteResource("bank-account", accountId, lane);
  const missing = await runJson(["bank-account", "get", "--id", accountId], {
    lane,
    label: "bank-account-get-missing",
    allowFailure: true,
  });
  expect(missing.exitCode).toBe(3);
  expectErrorEnvelope(missing.json, "bank-account get");
}, 30000);
