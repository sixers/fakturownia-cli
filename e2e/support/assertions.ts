import { expect } from "bun:test";

export interface EnvelopeMeta {
  command: string;
  duration_ms: number;
  profile?: string;
  request_id?: string;
  pagination?: {
    page: number;
    per_page: number;
    returned: number;
    has_next: boolean;
  };
}

export interface Envelope<T = unknown> {
  schema_version: string;
  status: "success" | "error";
  data: T;
  errors: Array<Record<string, unknown>>;
  warnings: Array<Record<string, unknown>> | null;
  meta: EnvelopeMeta;
}

export function expectSuccessEnvelope<T>(value: unknown, command: string): asserts value is Envelope<T> {
  const envelope = value as Envelope<T>;
  expect(envelope.schema_version).toBe("fakturownia-cli/v1alpha1");
  expect(envelope.status).toBe("success");
  expect(Array.isArray(envelope.errors)).toBe(true);
  expect(envelope.errors.length).toBe(0);
  expect(envelope.meta.command).toBe(command);
}

export function expectErrorEnvelope<T>(value: unknown, command: string): asserts value is Envelope<T> {
  const envelope = value as Envelope<T>;
  expect(envelope.schema_version).toBe("fakturownia-cli/v1alpha1");
  expect(envelope.status).toBe("error");
  expect(Array.isArray(envelope.errors)).toBe(true);
  expect(envelope.errors.length).toBeGreaterThan(0);
  expect(envelope.meta.command).toBe(command);
}

export function expectObjectWithId(value: unknown): { id: string | number } & Record<string, unknown> {
  expect(value).toBeTruthy();
  expect(typeof value).toBe("object");
  const objectValue = value as { id?: string | number } & Record<string, unknown>;
  expect(objectValue.id).toBeTruthy();
  return objectValue as { id: string | number } & Record<string, unknown>;
}

export function toId(value: unknown): string {
  const objectValue = expectObjectWithId(value);
  return String(objectValue.id);
}
