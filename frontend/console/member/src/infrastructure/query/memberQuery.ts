/// <reference types="vite/client" />

/**
 * Member クエリ（REST API アダプタ）
 *
 * - Module Federation 前提:
 *   - 認証ヘッダは Shell 側の auth から取得する
 *   - currentMember の取得は Shell 側の useCurrentMember() に委譲
 *
 * このファイルは「メンバー一覧・詳細など、member bounded context の読み取り専用クエリ」を担当します。
 */

import { getAuthHeaders } from "../../../../shell/src/auth/application/authService"; // Shell 側で再エクスポートしておく前提

// -------------------------------
// Backend base URL（他モジュールと同一ルール）
// -------------------------------
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

const API_BASE = ENV_BASE || FALLBACK_BASE;

// -------------------------------
// Types
// -------------------------------

/**
 * Backend の Member エンティティに対応する DTO
 * （必要に応じて backend/internal/domain/member/entity.go に合わせて拡張）
 */
export type MemberDTO = {
  id: string;
  firstName?: string | null;
  lastName?: string | null;
  firstNameKana?: string | null;
  lastNameKana?: string | null;
  email?: string | null;
  companyId: string;
  permissions?: string[];
  assignedBrands?: string[];
  createdAt?: string | null; // ISO8601 文字列想定
  updatedAt?: string | null; // ISO8601 文字列想定

  /** 表示用の結合済み氏名（backend が返す場合） */
  fullName?: string | null;
};

/**
 * メンバー一覧取得用のフィルタ
 * （サーバ側の仕様に合わせて適宜パラメータを追加）
 */
export type MemberListFilter = {
  companyId?: string;  // 会社単位で絞り込みたい場合
  keyword?: string;    // 氏名 / メール検索など
  limit?: number;
  offset?: number;
};

/**
 * ページングレスポンスの例（必要に応じて共通 Page 型と揃える）
 */
export type MemberListResponse = {
  items: MemberDTO[];
  total: number;
  limit: number;
  offset: number;
};

// -------------------------------
// Helper
// -------------------------------

function buildUrl(
  path: string,
  query?: Record<string, string | number | boolean | undefined>,
): string {
  const base = `${API_BASE.replace(/\/+$/g, "")}/${path.replace(/^\/+/g, "")}`;
  if (!query) return base;

  const params = new URLSearchParams();
  Object.entries(query).forEach(([key, value]) => {
    if (value === undefined || value === null || value === "") return;
    params.set(key, String(value));
  });

  const qs = params.toString();
  return qs ? `${base}?${qs}` : base;
}

// -------------------------------
// Queries
// -------------------------------

/**
 * メンバー詳細取得（ID 指定）
 *
 * GET /members/{id}
 */
export async function fetchMemberById(memberId: string): Promise<MemberDTO | null> {
  const id = (memberId ?? "").trim();
  if (!id) return null;

  const headers: HeadersInit = {
    "Content-Type": "application/json",
    ...(await getAuthHeaders()),
  };

  const url = buildUrl(`/members/${encodeURIComponent(id)}`);
  const res = await fetch(url, { method: "GET", headers });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    console.warn("[memberQuery] fetchMemberById failed:", res.status, text);
    return null;
  }

  const ct = res.headers.get("Content-Type") ?? "";
  if (!ct.includes("application/json")) {
    console.error(
      `[memberQuery] fetchMemberById: unexpected content-type=${ct}. Check API_BASE=${API_BASE}`,
    );
    return null;
  }

  const raw = (await res.json()) as any;
  if (!raw) return null;

  return normalizeMember(raw);
}

/**
 * メンバー一覧取得
 *
 * GET /members?companyId=...&keyword=...&limit=...&offset=...
 * （実際のクエリパラメータ名はサーバ仕様に合わせてください）
 */
export async function fetchMemberList(
  filter: MemberListFilter = {},
): Promise<MemberListResponse> {
  const headers: HeadersInit = {
    "Content-Type": "application/json",
    ...(await getAuthHeaders()),
  };

  const url = buildUrl("/members", {
    companyId: filter.companyId,
    keyword: filter.keyword,
    limit: filter.limit ?? 20,
    offset: filter.offset ?? 0,
  });

  const res = await fetch(url, { method: "GET", headers });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    console.warn("[memberQuery] fetchMemberList failed:", res.status, text);
    return {
      items: [],
      total: 0,
      limit: filter.limit ?? 20,
      offset: filter.offset ?? 0,
    };
  }

  const ct = res.headers.get("Content-Type") ?? "";
  if (!ct.includes("application/json")) {
    console.error(
      `[memberQuery] fetchMemberList: unexpected content-type=${ct}. Check API_BASE=${API_BASE}`,
    );
    return {
      items: [],
      total: 0,
      limit: filter.limit ?? 20,
      offset: filter.offset ?? 0,
    };
  }

  const raw = (await res.json()) as any;

  // サーバのレスポンス形状に合わせてパース
  // 例:
  // {
  //   items: [...],
  //   total: 123,
  //   limit: 20,
  //   offset: 0
  // }
  const items = Array.isArray(raw.items) ? raw.items.map(normalizeMember) : [];
  const total = typeof raw.total === "number" ? raw.total : items.length;
  const limit = typeof raw.limit === "number" ? raw.limit : filter.limit ?? 20;
  const offset =
    typeof raw.offset === "number" ? raw.offset : filter.offset ?? 0;

  return { items, total, limit, offset };
}

// -------------------------------
// Normalizer
// -------------------------------

function normalizeMember(raw: any): MemberDTO {
  const noFirst =
    raw.firstName === null ||
    raw.firstName === undefined ||
    raw.firstName === "";
  const noLast =
    raw.lastName === null ||
    raw.lastName === undefined ||
    raw.lastName === "";

  const firstName = noFirst ? null : (raw.firstName as string);
  const lastName = noLast ? null : (raw.lastName as string);

  const full =
    (raw.fullName as string | undefined | null)?.trim() ||
    `${lastName ?? ""} ${firstName ?? ""}`.trim() ||
    null;

  return {
    id: raw.id ?? "",
    firstName,
    lastName,
    firstNameKana: raw.firstNameKana ?? null,
    lastNameKana: raw.lastNameKana ?? null,
    email: raw.email ?? null,
    companyId: raw.companyId ?? "",
    permissions: Array.isArray(raw.permissions) ? raw.permissions : [],
    assignedBrands: Array.isArray(raw.assignedBrands)
      ? raw.assignedBrands
      : [],
    createdAt: raw.createdAt ?? null,
    updatedAt: raw.updatedAt ?? null,
    fullName: full,
  };
}
