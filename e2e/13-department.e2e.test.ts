import { expect, test } from "bun:test";

import { expectErrorEnvelope, toId } from "./support/assertions";
import { runJson } from "./support/cli";
import { getContext, laneName, resourceName } from "./support/context";
import { createResource, deleteResource, getResource, listResource, updateResource } from "./support/resources";

test("department CRUD and logo upload work end-to-end", async () => {
  const lane = laneName("department");
  const department = await createResource("department", {
    name: resourceName(lane, "department"),
    shortcut: resourceName(lane, "shortcut").slice(-12).toUpperCase(),
    tax_no: "6793188928",
  }, lane);

  const departmentId = toId(department);
  expect(departmentId).not.toBe(getContext().baseline.mainDepartmentId);

  const fetched = await getResource("department", departmentId, [], lane);
  expect((fetched.json.data as Record<string, unknown>).name).toBe(department.name);

  const listed = await listResource("department", [], lane);
  expect((listed.json.data as Array<Record<string, unknown>>).some((item) => String(item.id) === departmentId)).toBe(true);

  await updateResource("department", departmentId, { name: resourceName(lane, "department-updated") }, lane);
  const refetched = await getResource("department", departmentId, [], lane);
  expect((refetched.json.data as Record<string, unknown>).name).toBe(resourceName(lane, "department-updated"));

  const logo = await runJson<Record<string, unknown>>([
    "department",
    "set-logo",
    "--id",
    departmentId,
    "--file",
    getContext().fixtures.logoPath,
  ], {
    lane,
    label: "department-set-logo",
  });
  expect(logo.exitCode).toBe(0);

  await deleteResource("department", departmentId, lane);
  const missing = await runJson(["department", "get", "--id", departmentId], { lane, label: "department-get-missing", allowFailure: true });
  expect(missing.exitCode).toBe(3);
  expectErrorEnvelope(missing.json, "department get");
});
