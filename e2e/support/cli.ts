import { writeFile } from "node:fs/promises";
import path from "node:path";

import { type Envelope } from "./assertions";
import { artifactPath, getContext } from "./context";

export interface CliRunOptions {
  stdin?: string;
  cwd?: string;
  env?: Record<string, string>;
  allowFailure?: boolean;
  lane?: string;
  label?: string;
}

export interface CliRunResult<T = unknown> {
  args: string[];
  exitCode: number;
  stdout: string;
  stderr: string;
  durationMs: number;
  json?: Envelope<T>;
}

export type ParsedCliRunResult<T = unknown> = CliRunResult<T> & { json: Envelope<T> };

async function readStream(stream: ReadableStream<Uint8Array> | null): Promise<string> {
  if (!stream) {
    return "";
  }
  return await new Response(stream).text();
}

export async function runCli<T = unknown>(args: string[], options: CliRunOptions = {}): Promise<CliRunResult<T>> {
  const context = getContext();
  const start = performance.now();
  const proc = Bun.spawn([context.binaryPath, ...args], {
    cwd: options.cwd,
    env: {
      ...process.env,
      FAKTUROWNIA_PROFILE: context.profile,
      ...options.env,
    },
    stdin: options.stdin === undefined ? undefined : new Blob([options.stdin]).stream(),
    stdout: "pipe",
    stderr: "pipe",
  });

  const [stdout, stderr, exitCode] = await Promise.all([
    readStream(proc.stdout),
    readStream(proc.stderr),
    proc.exited,
  ]);
  const result: CliRunResult<T> = {
    args,
    exitCode,
    stdout,
    stderr,
    durationMs: Math.round(performance.now() - start),
  };

  if (options.lane && options.label) {
    const filePath = artifactPath(options.lane, `${options.label}.command.json`);
    await writeFile(filePath, JSON.stringify(result, null, 2));
  }

  if (!options.allowFailure && exitCode !== 0) {
    throw new Error(`command failed (${exitCode}): ${[context.binaryPath, ...args].join(" ")}\nstdout:\n${stdout}\nstderr:\n${stderr}`);
  }
  return result;
}

export async function runJson<T = unknown>(args: string[], options: CliRunOptions = {}): Promise<ParsedCliRunResult<T>> {
  const withJson = args.includes("--json") ? args : [...args, "--json"];
  const result = await runCli<T>(withJson, options);
  try {
    result.json = JSON.parse(result.stdout) as Envelope<T>;
  } catch (error) {
    throw new Error(`failed to parse JSON output for ${withJson.join(" ")}: ${(error as Error).message}\nstdout:\n${result.stdout}`);
  }
  return result as ParsedCliRunResult<T>;
}

export async function runHuman(args: string[], options: CliRunOptions = {}): Promise<CliRunResult> {
  return await runCli(args, options);
}

export async function downloadPathFor(lane: string, name: string): Promise<string> {
  return path.join(path.dirname(artifactPath(lane, name)), name);
}
