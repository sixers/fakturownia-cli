import { expect } from "bun:test";

import { expectSuccessEnvelope, toId, type Envelope } from "./assertions";
import { runJson, type ParsedCliRunResult } from "./cli";
import { registerDelete } from "./cleanup";
import { supportsCommand } from "./context";

export async function listResource(noun: string, args: string[] = [], lane?: string): Promise<ParsedCliRunResult<unknown[]>> {
  const result = await runJson<unknown[]>([noun, "list", ...args], { lane, label: `${noun}-list` });
  expectSuccessEnvelope(result.json, `${noun} list`);
  expect(Array.isArray(result.json.data)).toBe(true);
  return result;
}

export async function getResource(noun: string, id: string | number, args: string[] = [], lane?: string): Promise<ParsedCliRunResult<Record<string, unknown>>> {
  const result = await runJson<Record<string, unknown>>([noun, "get", "--id", String(id), ...args], { lane, label: `${noun}-get-${id}` });
  expectSuccessEnvelope(result.json, `${noun} get`);
  return result;
}

export async function createResource(
  noun: string,
  input: Record<string, unknown>,
  lane: string,
  options: { args?: string[]; registerCleanup?: boolean } = {},
): Promise<Record<string, unknown>> {
  const result = await runJson<Record<string, unknown>>([noun, "create", ...(options.args ?? []), "--input", JSON.stringify(input)], {
    lane,
    label: `${noun}-create`,
  });
  expectSuccessEnvelope(result.json, `${noun} create`);
  const data = result.json.data as Record<string, unknown>;
  if (options.registerCleanup !== false && supportsCommand(noun, "delete")) {
    await registerDelete(lane, noun, toId(data));
  }
  return data;
}

export async function updateResource(
  noun: string,
  id: string | number,
  input: Record<string, unknown>,
  lane: string,
  args: string[] = [],
): Promise<Record<string, unknown>> {
  const result = await runJson<Record<string, unknown>>([noun, "update", "--id", String(id), ...args, "--input", JSON.stringify(input)], {
    lane,
    label: `${noun}-update-${id}`,
  });
  expectSuccessEnvelope(result.json, `${noun} update`);
  return result.json.data as Record<string, unknown>;
}

export async function deleteResource(
  noun: string,
  id: string | number,
  lane: string,
  options: { required?: boolean } = {},
): Promise<Envelope<Record<string, unknown>> | null> {
  if (!supportsCommand(noun, "delete")) {
    if (options.required === false) {
      return null;
    }
    throw new Error(`delete command is not supported by the active binary for ${noun}`);
  }
  const result = await runJson<Record<string, unknown>>([noun, "delete", "--id", String(id), "--yes"], {
    lane,
    label: `${noun}-delete-${id}`,
  });
  expectSuccessEnvelope(result.json, `${noun} delete`);
  return result.json;
}
