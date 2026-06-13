// frontend\console\shell\src\shared\http\apiBase.ts
/// <reference types="vite/client" />

/**
 * Shared API base resolver for module federation remotes.
 *
 * Policy:
 * - env VITE_BACKEND_BASE_URL は「originのみ」（例: https://...run.app）を想定
 * - console API は現状 /console プレフィックス無しで動いているため、console は origin を返す
 * - 事故防止のため、env に /console 等が入っていても除去して正規化する
 */

type ApiScope = "console" | "mall";

/** Cloud Run fallback. */
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

  s = s.replace(/\/+$/g, "");
  s = s.replace(/\/(console|mall|sns)(\/.*)?$/i, "");
  s = s.replace(/\/+$/g, "");

  return s;
}

/** Join base and path safely. */
function join(base: string, path: string): string {
  const b = (base ?? "").replace(/\/+$/g, "");
  const p = (path ?? "").replace(/^\/+/g, "");
  if (!b) return `/${p}`;
  if (!p) return b;
  return `${b}/${p}`;
}

/** Backend origin. */
export function getBackendOrigin(): string {
  const env = normalizeOrigin(readEnvBackendBase());
  return env || FALLBACK_BACKEND_ORIGIN;
}

/**
 * Base URL for a scope.
 *
 * console API は現状 /console ではなくルート直下で提供しているため、
 * scope === "console" は origin を返す。
 */
export function getApiBase(scope: ApiScope): string {
  const origin = getBackendOrigin();
  if (scope === "console") return origin;
  return join(origin, scope);
}

/** Console API base. */
export function getConsoleApiBase(): string {
  return getApiBase("console");
}

/**
 * Console API base.
 *
 * Existing repositories can import this as:
 * import { API_BASE } from ".../apiBase";
 */
export const API_BASE = getConsoleApiBase();

/** Build a full URL under console base with a given path. */
export function buildConsoleUrl(path: string): string {
  return join(getConsoleApiBase(), path);
}

/** Build a full URL for a given API scope. */
export function buildApiUrl(scope: ApiScope, path: string): string {
  return join(getApiBase(scope), path);
}