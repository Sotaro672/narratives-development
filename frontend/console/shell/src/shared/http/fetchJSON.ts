// frontend/console/shell/src/shared/http/fetchJSON.ts

/**
 * Shared fetch helpers for module federation remotes.
 *
 * - Ensures JSON response (unless allowNonJson=true)
 * - Throws a structured HttpError on non-2xx
 * - Extracts small body snippet for debugging (safe size)
 */

export type HttpErrorInit = {
  url: string;
  method?: string;
  status: number;
  statusText?: string;
  contentType?: string;
  bodyText?: string;
};

export class HttpError extends Error {
  name = "HttpError" as const;

  url: string;
  method?: string;
  status: number;
  statusText?: string;
  contentType?: string;
  bodyText?: string;

  constructor(init: HttpErrorInit) {
    const msg = `${init.method ?? "GET"} ${init.status} ${init.url}`;
    super(msg);

    this.url = init.url;
    this.method = init.method;
    this.status = init.status;
    this.statusText = init.statusText;
    this.contentType = init.contentType;
    this.bodyText = init.bodyText;
  }
}

/** Best-effort read text with size limit to avoid huge logs. */
async function readTextSafely(res: Response, limit = 2000): Promise<string> {
  try {
    const text = await res.text();
    if (!text) return "";
    return text.length > limit ? text.slice(0, limit) : text;
  } catch {
    return "";
  }
}

export type FetchJSONOptions = RequestInit & {
  /**
   * If true, do not enforce "application/json" content-type.
   * Useful for endpoints that return empty body (204) or text.
   */
  allowNonJson?: boolean;

  /**
   * If set, include at most this many characters from error bodyText.
   * Defaults to 2000.
   */
  errorBodyLimit?: number;
};

/**
 * fetchJSON fetches and parses JSON, throwing HttpError on failures.
 *
 * Example:
 *   const data = await fetchJSON<MyType>(url, { headers })
 */
export async function fetchJSON<T = unknown>(
  input: RequestInfo | URL,
  init?: FetchJSONOptions,
): Promise<T> {
  const res = await fetch(input, init);

  const url =
    typeof input === "string"
      ? input
      : input instanceof URL
        ? input.toString()
        : (res.url || "");

  const method = (init?.method ?? "GET").toUpperCase();
  const ct = res.headers.get("content-type") ?? "";
  const allowNonJson = Boolean(init?.allowNonJson);

  // Non-OK -> throw with body snippet
  if (!res.ok) {
    const bodyText = await readTextSafely(res, init?.errorBodyLimit ?? 2000);
    throw new HttpError({
      url,
      method,
      status: res.status,
      statusText: res.statusText,
      contentType: ct,
      bodyText,
    });
  }

  // 204 No Content
  if (res.status === 204) {
    return undefined as unknown as T;
  }

  // Enforce JSON unless allowed
  if (!allowNonJson && !ct.toLowerCase().includes("application/json")) {
    const bodyText = await readTextSafely(res, init?.errorBodyLimit ?? 2000);
    throw new HttpError({
      url,
      method,
      status: res.status,
      statusText: res.statusText,
      contentType: ct,
      bodyText: `Unexpected content-type: ${ct}\n${bodyText}`.slice(0, init?.errorBodyLimit ?? 2000),
    });
  }

  // Parse JSON (best-effort)
  try {
    return (await res.json()) as T;
  } catch (e) {
    const bodyText = await readTextSafely(res, init?.errorBodyLimit ?? 2000);
    throw new HttpError({
      url,
      method,
      status: res.status,
      statusText: res.statusText,
      contentType: ct,
      bodyText: `Failed to parse JSON: ${(e as Error)?.message ?? String(e)}\n${bodyText}`.slice(
        0,
        init?.errorBodyLimit ?? 2000,
      ),
    });
  }
}
