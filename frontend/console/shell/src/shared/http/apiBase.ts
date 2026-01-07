// frontend/console/shell/src/shared/http/apiBase.ts
/// <reference types="vite/client" />

/**
 * Shared API base resolver for module federation remotes.
 *
 * Policy (修正案A):
 * - env VITE_BACKEND_BASE_URL は「originのみ」（例: https://...run.app）を想定
 * - コード側で /console を付与して console API base を作る
 * - ただし事故防止のため、env に /console 等が入っていても除去して正規化する
 */

type ApiScope = "console" | "mall" | "sns";

/** Cloud Run fallback (keep as last resort). */
export const FALLBACK_BACKEND_ORIGIN =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

/** Read Vite env safely and normalize. */
function readEnvBackendBase(): string {
  const raw =
    ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined) ?? "";
  return String(raw).trim();
}

/**
 * Normalize backend origin:
 * - trim
 * - strip trailing slashes
 * - strip known path suffixes (/console, /mall, /sns) if mistakenly included
 */
function normalizeOrigin(input: string): string {
  let s = String(input ?? "").trim();
  if (!s) return "";

  // remove trailing slashes
  s = s.replace(/\/+$/g, "");

  // If user mistakenly sets "https://.../console", strip it back to origin.
  // (Also strip nested /console/ etc.)
  s = s.replace(/\/(console|mall|sns)(\/.*)?$/i, "");

  // remove trailing slashes again after stripping
  s = s.replace(/\/+$/g, "");

  return s;
}

/** Join base and path safely (avoid double slashes). */
function join(base: string, path: string): string {
  const b = (base ?? "").replace(/\/+$/g, "");
  const p = (path ?? "").replace(/^\/+/g, "");
  if (!b) return `/${p}`;
  if (!p) return b;
  return `${b}/${p}`;
}

/** Backend origin (no /console). */
export function getBackendOrigin(): string {
  const env = normalizeOrigin(readEnvBackendBase());
  return env || FALLBACK_BACKEND_ORIGIN;
}

/** Base URL for a scope, e.g. /console */
export function getApiBase(scope: ApiScope): string {
  const origin = getBackendOrigin();
  return join(origin, scope);
}

/** Convenience for 修正案A */
export function getConsoleApiBase(): string {
  return getApiBase("console");
}

/**
 * ✅ Backward-compatible export:
 * repositories can import { API_BASE } as "console API base".
 */
export const API_BASE = getConsoleApiBase();

/** Optional: build a full URL under /console with a given path. */
export function buildConsoleUrl(path: string): string {
  // allow callers to pass "/members/xxx" or "members/xxx"
  return join(getConsoleApiBase(), path);
}

/** Optional: for other scopes */
export function buildApiUrl(scope: ApiScope, path: string): string {
  return join(getApiBase(scope), path);
}
