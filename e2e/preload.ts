import { beforeAll, afterAll } from "bun:test";
import { mkdirSync } from "node:fs";
import { writeFile } from "node:fs/promises";
import path from "node:path";

import { expectSuccessEnvelope } from "./support/assertions";
import { runJson } from "./support/cli";
import { runCleanup } from "./support/cleanup";
import { commandKey, setContext, type E2EContext } from "./support/context";
import { assertMailtestAuth, assertMailtestHealth } from "./support/mailtest";

function requiredEnv(name: string, fallback?: string): string {
  const value = process.env[name] ?? fallback;
  if (!value) {
    throw new Error(`missing required environment variable ${name}`);
  }
  return value;
}

function createRunId(): string {
  const now = new Date();
  const parts = [
    now.getUTCFullYear(),
    String(now.getUTCMonth() + 1).padStart(2, "0"),
    String(now.getUTCDate()).padStart(2, "0"),
    String(now.getUTCHours()).padStart(2, "0"),
    String(now.getUTCMinutes()).padStart(2, "0"),
    String(now.getUTCSeconds()).padStart(2, "0"),
  ];
  return parts.join("");
}

async function writeFixtures(artifactDir: string) {
  const fixturesDir = path.join(artifactDir, "fixtures");
  mkdirSync(fixturesDir, { recursive: true });

  const logoPath = path.join(fixturesDir, "tiny-logo.gif");
  const attachmentTextPath = path.join(fixturesDir, "attachment.txt");
  const attachmentPdfPath = path.join(fixturesDir, "attachment.pdf");

  const tinyGif = Buffer.from(
    "R0lGODdhAQABAIAAAAAAAP///ywAAAAAAQABAAACAUwAOw==",
    "base64",
  );
  await writeFile(logoPath, tinyGif);
  await writeFile(attachmentTextPath, "fakturownia e2e attachment\n", "utf8");
  await writeFile(attachmentPdfPath, "%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n<< >>\n%%EOF\n", "utf8");

  return {
    logoPath,
    attachmentTextPath,
    attachmentPdfPath,
  };
}

beforeAll(async () => {
  const runId = createRunId();
  const artifactDir = path.join(process.cwd(), "tmp", "e2e-artifacts", runId);
  mkdirSync(artifactDir, { recursive: true });

  const context: E2EContext = {
    runId,
    artifactDir,
    binaryPath: requiredEnv("FAKTUROWNIA_BIN", "/Users/mateusz/.local/bin/fakturownia"),
    profile: requiredEnv("FAKTUROWNIA_PROFILE", "cli"),
    supportedCommands: [],
    mailtest: {
      baseUrl: requiredEnv("MAILTEST_API_BASE_URL", "https://api.mailtestapi.com"),
      apiKey: requiredEnv("MAILTEST_API_KEY"),
    },
    baseline: {
      account: {} as Record<string, unknown>,
      mainDepartmentId: undefined as string | undefined,
      defaultWarehouseId: undefined as string | undefined,
    },
    fixtures: await writeFixtures(artifactDir),
    cleanupFile: path.join(artifactDir, "cleanup.jsonl"),
  };

  setContext(context);

  const doctor = await runJson<Record<string, unknown>>(["doctor", "run"], { lane: "preload", label: "doctor-run" });
  expectSuccessEnvelope(doctor.json, "doctor run");

  const account = await runJson<Record<string, unknown>>(["account", "get"], { lane: "preload", label: "account-get" });
  expectSuccessEnvelope(account.json, "account get");
  context.baseline.account = account.json.data as Record<string, unknown>;

  const schemaList = await runJson<Array<Record<string, unknown>>>(["schema", "list"], {
    lane: "preload",
    label: "schema-list",
  });
  expectSuccessEnvelope(schemaList.json, "schema list");
  context.supportedCommands = (schemaList.json.data as Array<Record<string, unknown>>).map((item) =>
    commandKey(String(item.noun), String(item.verb))
  );

  const departments = await runJson<Array<Record<string, unknown>>>(["department", "list"], {
    lane: "preload",
    label: "department-list",
  });
  expectSuccessEnvelope(departments.json, "department list");
  const mainDepartment = (departments.json.data as Array<Record<string, unknown>>).find((item) => item.main === true);
  context.baseline.mainDepartmentId = mainDepartment?.id ? String(mainDepartment.id) : undefined;

  const warehouses = await runJson<Array<Record<string, unknown>>>(["warehouse", "list"], {
    lane: "preload",
    label: "warehouse-list",
  });
  expectSuccessEnvelope(warehouses.json, "warehouse list");
  const defaultWarehouse = (warehouses.json.data as Array<Record<string, unknown>>).find((item) => item.kind === "main") ??
    (warehouses.json.data as Array<Record<string, unknown>>)[0];
  context.baseline.defaultWarehouseId = defaultWarehouse?.id ? String(defaultWarehouse.id) : undefined;

  await assertMailtestHealth();
  await assertMailtestAuth();
}, { timeout: 30000 });

afterAll(async () => {
  if (!globalThis.__FAKTUROWNIA_E2E_CONTEXT__) {
    return;
  }
  await runCleanup();
}, { timeout: 60000 });
