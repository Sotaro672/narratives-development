// frontend/console/shell/src/shared/http/queryString.ts

/**
 * Shared query string builder for module federation remotes.
 *
 * - Skips undefined/null/""
 * - Appends arrays as repeated keys: key=a&key=b
 * - Converts values to string
 */

export type QueryValue =
  | string
  | number
  | boolean
  | null
  | undefined
  | Date
  | Array<string | number | boolean | Date | null | undefined>;

export type QueryParams = Record<string, QueryValue>;

function isEmptyValue(v: unknown): boolean {
  return v === undefined || v === null || v === "";
}

function valueToString(v: string | number | boolean | Date): string {
  if (v instanceof Date) return v.toISOString();
  return String(v);
}

/**
 * Build query string without leading "?".
 *
 * Example:
 *   toQuery({ q: "abc", brandIds: ["1","2"], page: 1 })
 *   -> "q=abc&brandIds=1&brandIds=2&page=1"
 */
export function toQuery(params: QueryParams): string {
  const sp = new URLSearchParams();

  Object.entries(params ?? {}).forEach(([k, v]) => {
    if (isEmptyValue(v)) return;

    if (Array.isArray(v)) {
      v.forEach((x) => {
        if (isEmptyValue(x)) return;
        sp.append(k, valueToString(x as any));
      });
      return;
    }

    sp.set(k, valueToString(v as any));
  });

  return sp.toString();
}

/**
 * Append query string to a base URL.
 *
 * Example:
 *   withQuery("/members", { page: 1 }) -> "/members?page=1"
 */
export function withQuery(baseUrl: string, params: QueryParams): string {
  const qs = toQuery(params);
  if (!qs) return baseUrl;

  // keep existing query if present
  const hasQuery = baseUrl.includes("?");
  if (!hasQuery) return `${baseUrl}?${qs}`;

  // baseUrl already has '?', append with '&' if needed
  const endsWithQorAmp = baseUrl.endsWith("?") || baseUrl.endsWith("&");
  return endsWithQorAmp ? `${baseUrl}${qs}` : `${baseUrl}&${qs}`;
}
