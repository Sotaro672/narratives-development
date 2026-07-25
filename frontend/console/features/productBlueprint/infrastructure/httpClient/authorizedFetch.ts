// frontend/console/productBlueprint/src/infrastructure/httpClient/authorizedFetch.ts

import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../shell/src/shared/http/authHeaders";

/**
 * Normalized error for HTTP failures.
 * - includes status / statusText
 * - tries to include response body text (best-effort)
 */
export class HttpError extends Error {
  readonly status: number;
  readonly statusText: string;
  readonly url: string;
  readonly bodyText?: string;

  constructor(args: { status: number; statusText: string; url: string; bodyText?: string }) {
    super(`HTTP ${args.status} ${args.statusText} (${args.url})`);
    this.name = "HttpError";
    this.status = args.status;
    this.statusText = args.statusText;
    this.url = args.url;
    this.bodyText = args.bodyText;
  }
}

/**
 * Build a full URL from a relative API path or pass-through absolute URL.
 */
function toUrl(pathOrUrl: string): string {
  const s = String(pathOrUrl ?? "").trim();
  if (!s) throw new Error("authorizedFetch: url/path is empty");

  // If absolute, pass-through.
  if (/^https?:\/\//i.test(s)) return s;

  // Ensure exactly one slash between base and path.
  const base = String(API_BASE ?? "").replace(/\/+$/, "");
  const path = s.replace(/^\/+/, "");
  return `${base}/${path}`;
}

type AuthorizedFetchOptions = Omit<RequestInit, "headers"> & {
  /**
   * If true, do NOT attach auth headers.
   * Default: false
   */
  noAuth?: boolean;

  /**
   * Additional headers to merge.
   * These override default headers when key conflicts.
   */
  headers?: Record<string, string>;

  /**
   * If true, automatically set "Content-Type: application/json"
   * when body is a plain object (and not FormData/Blob/etc).
   * Default: true
   */
  json?: boolean;

  /**
   * If true, throw HttpError when response is not ok.
   * Default: true
   */
  throwOnError?: boolean;

  /**
   * If true, add "Accept: application/json" unless already provided.
   * Default: true
   */
  acceptJson?: boolean;
};

/**
 * Fetch wrapper that:
 * - prefixes API_BASE for relative paths
 * - attaches auth headers (unless noAuth)
 * - normalizes JSON request headers (optional)
 * - throws HttpError on non-2xx by default
 */
export async function authorizedFetch(
  pathOrUrl: string,
  options: AuthorizedFetchOptions = {},
): Promise<Response> {
  const url = toUrl(pathOrUrl);

  const {
    noAuth = false,
    headers: extraHeaders = {},
    json = true,
    throwOnError = true,
    acceptJson = true,
    ...rest
  } = options;

  const authHeaders = noAuth ? {} : await getAuthHeadersOrThrow();

  const headers: Record<string, string> = {
    ...authHeaders,
    ...(acceptJson ? { Accept: extraHeaders.Accept ?? "application/json" } : {}),
    ...extraHeaders,
  };

  // If the caller passes a plain object as body and json=true, stringify it.
  // NOTE: RequestInit.body type is BodyInit | null; we allow object and coerce here.
  const anyBody = (rest as any).body;
  const isBodyPlainObject =
    anyBody != null &&
    typeof anyBody === "object" &&
    !(anyBody instanceof FormData) &&
    !(anyBody instanceof Blob) &&
    !(anyBody instanceof ArrayBuffer) &&
    !(anyBody instanceof URLSearchParams) &&
    !(anyBody instanceof ReadableStream);

  const finalInit: RequestInit = {
    ...rest,
    headers,
  };

  if (json && isBodyPlainObject) {
    if (!headers["Content-Type"] && !headers["content-type"]) {
      headers["Content-Type"] = "application/json";
    }
    finalInit.body = JSON.stringify(anyBody);
  }

  const res = await fetch(url, finalInit);

  if (!throwOnError || res.ok) return res;

  // Best-effort read response body for debugging
  let bodyText: string | undefined;
  try {
    bodyText = await res.text();
  } catch {
    bodyText = undefined;
  }

  // Throw a normalized error (caller can inspect status/bodyText)
  throw new HttpError({
    status: res.status,
    statusText: res.statusText ?? "",
    url,
    bodyText,
  });
}

/**
 * Convenience: fetch JSON response with proper typing.
 * - throws HttpError for non-2xx by default
 */
export async function authorizedFetchJson<T>(
  pathOrUrl: string,
  options: AuthorizedFetchOptions = {},
): Promise<T> {
  const res = await authorizedFetch(pathOrUrl, options);
  // If the API returns 204, caller should not use this helper.
  return (await res.json()) as T;
}
