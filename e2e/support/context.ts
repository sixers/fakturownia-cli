import { mkdirSync } from "node:fs";
import path from "node:path";

export interface CleanupEntry {
  key: string;
  lane: string;
  label: string;
  args: string[];
  registeredAt: string;
}

export interface MailtestConfig {
  baseUrl: string;
  apiKey: string;
}

export interface FixturePaths {
  logoPath: string;
  attachmentTextPath: string;
  attachmentPdfPath: string;
}

export interface BaselineState {
  account: Record<string, unknown>;
  mainDepartmentId?: string;
  defaultWarehouseId?: string;
}

export interface E2EContext {
  runId: string;
  artifactDir: string;
  binaryPath: string;
  profile: string;
  supportedCommands: string[];
  mailtest: MailtestConfig;
  baseline: BaselineState;
  fixtures: FixturePaths;
  cleanupFile: string;
}

declare global {
  var __FAKTUROWNIA_E2E_CONTEXT__: E2EContext | undefined;
}

export function setContext(context: E2EContext): void {
  globalThis.__FAKTUROWNIA_E2E_CONTEXT__ = context;
}

export function getContext(): E2EContext {
  if (!globalThis.__FAKTUROWNIA_E2E_CONTEXT__) {
    throw new Error("E2E context is not initialized. Did preload.ts run?");
  }
  return globalThis.__FAKTUROWNIA_E2E_CONTEXT__;
}

export function laneName(prefix: string): string {
  return prefix;
}

export function commandKey(noun: string, verb: string): string {
  return `${noun}:${verb}`;
}

export function supportsCommand(noun: string, verb: string): boolean {
  return getContext().supportedCommands.includes(commandKey(noun, verb));
}

export function resourceName(lane: string, kind: string): string {
  const { runId } = getContext();
  return `e2e-${runId}-${lane}-${kind}`;
}

export function compactToken(label: string, maxLength = 24): string {
  const { runId } = getContext();
  return `${label}-${runId}`.slice(0, maxLength);
}

export function laneRecipient(lane: string, localPart = "invoice-email"): string {
  const { runId } = getContext();
  return `${runId}-${lane}-${localPart}@mailtestapi.com`;
}

export function laneArtifactDir(lane: string): string {
  const dir = path.join(getContext().artifactDir, lane);
  mkdirSync(dir, { recursive: true });
  return dir;
}

export function artifactPath(lane: string, fileName: string): string {
  return path.join(laneArtifactDir(lane), fileName);
}

export function invoiceRequiredFields(overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    seller_name: "E2E Seller Sp. z o.o.",
    seller_tax_no: "5252445767",
    seller_street: "ul. Przykładowa 10",
    seller_post_code: "00-001",
    seller_city: "Warszawa",
    buyer_company: true,
    buyer_tax_no: "9876543210",
    buyer_country: "PL",
    ...overrides,
  };
}

export function bankAccountNumber(seed: string): string {
  const { runId } = getContext();
  const suffixDigits = Array.from(`${runId}${seed}`)
    .map((char) => (/\d/.test(char) ? char : String(char.charCodeAt(0) % 10)))
    .join("")
    .slice(-12)
    .padStart(12, "0");
  return `PL61 1090 1014 0000 ${suffixDigits.slice(0, 4)} ${suffixDigits.slice(4, 8)} ${suffixDigits.slice(8, 12)}`;
}
