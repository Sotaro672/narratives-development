// frontend/console/mintRequest/src/infrastructure/http/httpClient.ts

import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../shell/src/shared/http/authHeaders";

export type HttpJsonResult<T> = {
  url: string;
  method: string;
  status: number;
  statusText: string;
  ok: boolean;
  text: string;
  data: T | null;
};

function isAbsoluteUrl(s: string): boolean {
  return /^https?:\/\//i.test(String(s ?? "").trim());
}

function normalizePath(path: string): string {
  const p = String(path ?? "").trim();
  if (!p) return "/";
  if (isAbsoluteUrl(p)) return p;
  return p.startsWith("/") ? p : `/${p}`;
}

export function buildUrl(path: string): string {
  const p = normalizePath(path);
  if (isAbsoluteUrl(p)) return p;

  const base = String(API_BASE ?? "").replace(/\/+$/g, "");
  if (!base) throw new Error("API_BASE is empty");

  return `${base}${p}`;
}

function mergeHeaders(a: HeadersInit | undefined, b: HeadersInit | undefined): HeadersInit {
  return { ...(a as any), ...(b as any) };
}

function parseJsonOrThrow<T>(text: string, url: string): T {
  const trimmed = String(text ?? "").trim();
  if (!trimmed) return null as any;

  try {
    return JSON.parse(trimmed) as T;
  } catch (e: any) {
    throw new Error(`Response is not JSON: url=${url} err=${e?.message ?? String(e)}`);
  }
}

/**
 * Authorization 付きで JSON を取得する薄いユーティリティ（GET/POST 等共通）
 *
 * - 404 を null 扱いにしたい場合は treat404AsNull を true
 * - 返却は「data + text + status」を含め、呼び出し側で分岐できるようにする
 */
export async function requestJsonWithAuth<T>(
  opName: string,
  path: string,
  init?: (RequestInit & { treat404AsNull?: boolean }) | null,
): Promise<HttpJsonResult<T>> {
  const effectiveInit = (init ?? {}) as RequestInit & { treat404AsNull?: boolean };

  const authHeaders = await getAuthHeadersOrThrow();

  const url = buildUrl(path);
  const method = String(effectiveInit.method ?? "GET").toUpperCase();

  // FormData のときは Content-Type を固定しない（ブラウザが boundary を付与する）
  const isForm = effectiveInit.body instanceof FormData;

  const baseHeaders: HeadersInit = isForm
    ? authHeaders
    : { ...authHeaders, "Content-Type": "application/json" };

  const headers = mergeHeaders(baseHeaders, effectiveInit.headers);

  // body: object を渡されたら JSON stringify（呼び出し側が string を渡してもOK）
  let body: any = effectiveInit.body;
  if (!isForm && body != null && typeof body !== "string") {
    body = JSON.stringify(body);
  }

  let res: Response;

  try {
    res = await fetch(url, {
      ...effectiveInit,
      method,
      headers,
      body,
    });
  } catch (e: any) {
    throw new Error(`Failed to fetch: ${method} ${url} err=${e?.message ?? String(e)}`);
  }

  const text = await res.text().catch(() => "");

  if (effectiveInit.treat404AsNull && res.status === 404) {
    return {
      url,
      method,
      status: res.status,
      statusText: res.statusText,
      ok: res.ok,
      text,
      data: null,
    };
  }

  if (!res.ok) {
    throw new Error(
      `${opName} error: ${res.status} ${res.statusText}${text ? `\n${text}` : ""}`,
    );
  }

  const data = parseJsonOrThrow<T>(text, url);

  return {
    url,
    method,
    status: res.status,
    statusText: res.statusText,
    ok: res.ok,
    text,
    data: (data ?? null) as any,
  };
}

/**
 * Authorization 付き JSON GET（Query 用）
 * - 404 を null にしたい場合は treat404AsNull を true
 */
export async function getJsonWithAuth<T>(
  opName: string,
  path: string,
  opts?: { treat404AsNull?: boolean } | null,
): Promise<T | null> {
  const r = await requestJsonWithAuth<T>(opName, path, {
    method: "GET",
    treat404AsNull: opts?.treat404AsNull ?? false,
  });

  return (r.data ?? null) as T | null;
}