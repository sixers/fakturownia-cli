import { expect, test } from "bun:test";

import { expectErrorEnvelope, expectSuccessEnvelope } from "./support/assertions";
import { runHuman, runJson } from "./support/cli";
import { laneName } from "./support/context";

test("smoke: auth, doctor, schema, output modes, dry-run, and usage errors", async () => {
  const lane = laneName("smoke");

  const auth = await runJson(["auth", "status"], { lane, label: "auth-status" });
  expectSuccessEnvelope(auth.json, "auth status");

  const doctor = await runJson(["doctor", "run"], { lane, label: "doctor-run" });
  expectSuccessEnvelope(doctor.json, "doctor run");

  const schema = await runJson(["schema", "list"], { lane, label: "schema-list" });
  expectSuccessEnvelope(schema.json, "schema list");
  expect(Array.isArray(schema.json.data)).toBe(true);

  const fields = await runJson(["product", "list", "--fields", "id,name"], { lane, label: "product-list-fields" });
  expectSuccessEnvelope(fields.json, "product list");
  if ((fields.json.data as Array<Record<string, unknown>>).length > 0) {
    const first = (fields.json.data as Array<Record<string, unknown>>)[0];
    expect(Object.keys(first).sort()).toEqual(["id", "name"]);
  }

  const human = await runHuman(["product", "list", "--columns", "id,name"], { lane, label: "product-list-columns" });
  expect(human.exitCode).toBe(0);
  expect(human.stdout).toContain("id");

  const raw = await runHuman(["product", "list", "--raw"], { lane, label: "product-list-raw" });
  expect(raw.exitCode).toBe(0);
  expect(Array.isArray(JSON.parse(raw.stdout))).toBe(true);

  const dryRun = await runJson(["client", "create", "--input", JSON.stringify({ name: "dry-run-client" }), "--dry-run"], {
    lane,
    label: "client-create-dry-run",
  });
  expectSuccessEnvelope(dryRun.json, "client create");
  expect((dryRun.json.data as Record<string, unknown>).method).toBe("POST");

  const invalid = await runJson(["schema"], { lane, label: "schema-invalid", allowFailure: true });
  expect(invalid.exitCode).toBe(2);
  expectErrorEnvelope(invalid.json, "schema");
});
