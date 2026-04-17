import { getContext } from "./context";

export interface MailtestEmailSummary {
  id: string;
  recipient: string;
  sender: string;
  subject: string | null;
  status: string;
}

export interface MailtestEmailDetail extends MailtestEmailSummary {
  body: {
    raw: string;
    text: string | null;
    html: string | null;
    headers: Record<string, string>;
    attachments: Array<{
      filename: string | null;
      contentType: string | null;
      disposition: string | null;
      size: number;
    }>;
  } | null;
}

function headers(): HeadersInit {
  return {
    authorization: `Bearer ${getContext().mailtest.apiKey}`,
  };
}

export async function assertMailtestHealth(): Promise<void> {
  const response = await fetch(`${getContext().mailtest.baseUrl}/health`);
  if (!response.ok) {
    throw new Error(`mailtest health failed with ${response.status}`);
  }
}

export async function assertMailtestAuth(): Promise<void> {
  const response = await fetch(`${getContext().mailtest.baseUrl}/v1/emails?limit=1`, {
    headers: headers(),
  });
  if (!response.ok) {
    throw new Error(`mailtest auth failed with ${response.status}`);
  }
}

export async function listEmails(recipient: string): Promise<MailtestEmailSummary[]> {
  const url = new URL("/v1/emails", getContext().mailtest.baseUrl);
  url.searchParams.set("recipient", recipient);
  url.searchParams.set("limit", "20");
  const response = await fetch(url, { headers: headers() });
  if (!response.ok) {
    throw new Error(`mailtest list failed with ${response.status}`);
  }
  const body = (await response.json()) as { items: MailtestEmailSummary[] };
  return body.items;
}

export async function getEmail(emailId: string): Promise<MailtestEmailDetail> {
  const response = await fetch(`${getContext().mailtest.baseUrl}/v1/emails/${emailId}`, {
    headers: headers(),
  });
  if (!response.ok) {
    throw new Error(`mailtest get failed with ${response.status}`);
  }
  const body = (await response.json()) as { item: MailtestEmailDetail };
  return body.item;
}

export async function waitForEmail(recipient: string, timeoutMs = 60000, intervalMs = 2000): Promise<MailtestEmailDetail> {
  const started = Date.now();
  while (Date.now() - started < timeoutMs) {
    const items = await listEmails(recipient);
    if (items.length > 0) {
      return await getEmail(items[0].id);
    }
    await Bun.sleep(intervalMs);
  }
  throw new Error(`timed out waiting for email to ${recipient}`);
}
