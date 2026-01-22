// frontend/console/shell/src/shared/http/authHeaders.ts
import { getAuthHeaders as getAuthHeadersFromShell } from "../../auth/application/authService";

/**
 * Shared HTTP helper: returns auth headers for backend requests.
 * - Delegates to shell authService.
 * - Always returns a plain Record<string, string>.
 * - Best-effort: if authService throws, return {} (so public endpoints still work).
 *
 * NOTE:
 * If you want strict behavior (throw on auth failure), use getAuthHeadersOrThrow().
 */
export async function getAuthHeaders(): Promise<Record<string, string>> {
  try {
    const h = await getAuthHeadersFromShell();
    // normalize to simple object (avoid Headers instance surprises)
    return { ...(h as Record<string, string>) };
  } catch {
    return {};
  }
}

/**
 * Strict helper: returns auth headers for backend requests.
 * - Throws if authService fails (e.g., not logged in, token refresh failed).
 * - Always returns a plain Record<string, string>.
 */
export async function getAuthHeadersOrThrow(): Promise<Record<string, string>> {
  const h = await getAuthHeadersFromShell();
  return { ...(h as Record<string, string>) };
}

/**
 * Convenience helper: merge auth headers with JSON content-type.
 */
export async function getAuthJsonHeaders(): Promise<Record<string, string>> {
  const auth = await getAuthHeaders();
  return {
    ...auth,
    "Content-Type": "application/json",
  };
}

/**
 * Strict JSON helper: merge strict auth headers with JSON content-type.
 */
export async function getAuthJsonHeadersOrThrow(): Promise<Record<string, string>> {
  const auth = await getAuthHeadersOrThrow();
  return {
    ...auth,
    "Content-Type": "application/json",
  };
}

/**
 * Convenience helper: merge auth headers with extra headers.
 */
export async function withAuthHeaders(
  extra?: Record<string, string>,
): Promise<Record<string, string>> {
  const auth = await getAuthHeaders();
  return {
    ...auth,
    ...(extra ?? {}),
  };
}

/**
 * Strict merge helper: merge strict auth headers with extra headers.
 */
export async function withAuthHeadersOrThrow(
  extra?: Record<string, string>,
): Promise<Record<string, string>> {
  const auth = await getAuthHeadersOrThrow();
  return {
    ...auth,
    ...(extra ?? {}),
  };
}
