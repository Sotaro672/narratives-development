// frontend/console/shell/src/shared/http/apiBase.ts
/// <reference types="vite/client" />

/**
 * Shared API base resolver for module federation remotes.
 *
 * Policy (現状のバックエンドに合わせた修正):
 * - env VITE_BACKEND_BASE_URL は「originのみ」（例: https://...run.app）を想定
 * - console API は "現状 /console プレフィックス無し" で動いているため、console は origin を返す
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

/**
 * Base URL for a scope.
 *
 * ✅ IMPORTANT:
 * - 現状の backend は console API を /console ではなくルート直下で提供しているため、
 *   scope==="console" は origin を返す。
 * - mall/sns は将来 prefix を切りたい時のために残す（必要なら backend 側も合わせる）
 */
export function getApiBase(scope: ApiScope): string {
  const origin = getBackendOrigin();
  if (scope === "console") return origin; // ←ここが404解消の要点
  return join(origin, scope);
}

/** Convenience */
export function getConsoleApiBase(): string {
  return getApiBase("console");
}

/**
 * ✅ Backward-compatible export:
 * repositories can import { API_BASE } as "console API base".
 */
export const API_BASE = getConsoleApiBase();

/** Optional: build a full URL under console base with a given path. */
export function buildConsoleUrl(path: string): string {
  // allow callers to pass "/members/xxx" or "members/xxx"
  return join(getConsoleApiBase(), path);
}

/** Optional: for other scopes */
export function buildApiUrl(scope: ApiScope, path: string): string {
  return join(getApiBase(scope), path);
}
