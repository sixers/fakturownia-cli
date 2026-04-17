import { appendFile, readFile, writeFile } from "node:fs/promises";

import { getContext, type CleanupEntry } from "./context";
import { runCli } from "./cli";

export async function registerCleanup(entry: Omit<CleanupEntry, "registeredAt" | "key">): Promise<void> {
  const fullEntry: CleanupEntry = {
    ...entry,
    key: `${entry.label}:${entry.args.join(" ")}`,
    registeredAt: new Date().toISOString(),
  };
  await appendFile(getContext().cleanupFile, `${JSON.stringify(fullEntry)}\n`, "utf8");
}

export async function registerDelete(lane: string, noun: string, id: string | number): Promise<void> {
  await registerCleanup({
    lane,
    label: `${noun}-${id}`,
    args: [noun, "delete", "--id", String(id), "--yes", "--json"],
  });
}

export async function runCleanup(): Promise<void> {
  const context = getContext();
  let entries: CleanupEntry[] = [];
  try {
    const content = await readFile(context.cleanupFile, "utf8");
    entries = content
      .split("\n")
      .map((line: string) => line.trim())
      .filter(Boolean)
      .map((line: string) => JSON.parse(line) as CleanupEntry);
  } catch {
    entries = [];
  }

  const seen = new Set<string>();
  const reversed = entries.reverse().filter((entry) => {
    if (seen.has(entry.key)) {
      return false;
    }
    seen.add(entry.key);
    return true;
  });

  const results: Array<Record<string, unknown>> = [];
  for (const entry of reversed) {
    const result = await runCli(entry.args, {
      allowFailure: true,
      lane: entry.lane,
      label: `cleanup-${entry.label}`,
    });
    results.push({
      entry,
      exitCode: result.exitCode,
      stdout: result.stdout,
      stderr: result.stderr,
    });
  }

  await writeFile(`${context.cleanupFile}.summary.json`, JSON.stringify(results, null, 2));
}
