//frontend\console\member\src\infrastructure\query\memberQuery.ts
/// <reference types="vite/client" />

import type { Member } from "../../domain/entity/member";
import type { MemberFilter } from "../../domain/repository/memberRepository";
import type { Page } from "../../../../shell/src/shared/types/common/common";
import { DEFAULT_PAGE_LIMIT } from "../../../../shell/src/shared/types/common/common";

// ─────────────────────────────────────────────
// Backend base URL
// ─────────────────────────────────────────────
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)
    ?.replace(/\/+$/g, "") ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = (ENV_BASE || FALLBACK_BASE).replace(/\/+$/g, "");

// URL builder
function apiUrl(path: string, qs?: string) {
  const url = `${API_BASE}${path.startsWith("/") ? path : `/${path}`}`;
  const full = qs ? `${url}?${qs}` : url;
  console.log("[memberQuery] GET", full);
  return full;
}

// ─────────────────────────────────────────────
// Query String Builder
// ─────────────────────────────────────────────
export function buildMemberQuery(usePage: Page, useFilter: MemberFilter): string {
  const perPage =
    usePage?.perPage && usePage.perPage > 0 ? usePage.perPage : DEFAULT_PAGE_LIMIT;

  const pageNumber =
    usePage?.number && usePage.number > 0 ? usePage.number : 1;

  const params = new URLSearchParams();
  params.set("page", String(pageNumber));
  params.set("perPage", String(perPage));
  params.set("sort", "updatedAt");
  params.set("order", "desc");

  if (useFilter.searchQuery?.trim()) {
    params.set("q", useFilter.searchQuery.trim());
  }
  if (useFilter.brandIds?.length) {
    params.set("brandIds", useFilter.brandIds.join(","));
  }
  if (useFilter.status?.trim()) {
    params.set("status", useFilter.status.trim());
  }

  return params.toString();
}

// ─────────────────────────────────────────────
// FormatLastFirst
// ─────────────────────────────────────────────
export function formatLastFirst(
  lastName?: string | null,
  firstName?: string | null,
) {
  const ln = String(lastName ?? "").trim();
  const fn = String(firstName ?? "").trim();
  if (ln && fn) return `${ln} ${fn}`;
  if (ln) return ln;
  if (fn) return fn;
  return "";
}

// ─────────────────────────────────────────────
// Normalize Wire Format
// ─────────────────────────────────────────────
function normalizeMemberWire(w: any): Member {
  const id = String(w.id ?? w.ID ?? "").trim();
  const firstName = w.firstName ?? w.FirstName ?? null;
  const lastName = w.lastName ?? w.LastName ?? null;
  const firstNameKana = w.firstNameKana ?? w.FirstNameKana ?? null;
  const lastNameKana = w.lastNameKana ?? w.LastNameKana ?? null;
  const email = w.email ?? w.Email ?? null;
  const companyId = w.companyId ?? w.CompanyID ?? "";
  const permissions = Array.isArray(w.permissions ?? w.Permissions)
    ? w.permissions ?? w.Permissions
    : [];
  const assignedBrands =
    Array.isArray(w.assignedBrands ?? w.AssignedBrands)
      ? w.assignedBrands ?? w.AssignedBrands
      : null;

  const createdAt = w.createdAt ?? w.CreatedAt ?? null;
  const updatedAt = w.updatedAt ?? w.UpdatedAt ?? null;
  const deletedAt = w.deletedAt ?? w.DeletedAt ?? null;
  const deletedBy = w.deletedBy ?? w.DeletedBy ?? null;
  const updatedBy = w.updatedBy ?? w.UpdatedBy ?? null;

  return {
    id,
    firstName,
    lastName,
    firstNameKana,
    lastNameKana,
    email,
    companyId,
    permissions,
    assignedBrands,
    createdAt,
    updatedAt,
    deletedAt,
    deletedBy,
    updatedBy,
  } as Member;
}

// ─────────────────────────────────────────────
// 型：PageResult<Member> 相当
// ─────────────────────────────────────────────
export type MemberListResult = {
  items: Member[];
  totalPages: number;
};

// ─────────────────────────────────────────────
// fetchMemberListWithToken（ページング対応）
// ─────────────────────────────────────────────
export async function fetchMemberListWithToken(
  token: string,
  page: Page,
  filter: MemberFilter,
): Promise<MemberListResult> {

  const qs = buildMemberQuery(page, filter);
  const url = apiUrl("/members", qs);

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
      Accept: "application/json",
    },
  });

  console.log("[memberQuery] response", res.status, res.statusText);

  const ct = res.headers.get("content-type") ?? "";
  if (!ct.includes("application/json")) {
    const text = await res.text().catch(() => "");
    throw new Error(`Unexpected content-type: ${ct}\n${text}`);
  }

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    if (res.status === 401)
      throw new Error(`認証エラー (401): ${text}`);
    if (res.status === 403)
      throw new Error(`認可エラー (403): ${text}`);
    throw new Error(`一覧取得に失敗: ${res.status} ${text}`);
  }

  const raw = await res.json();

  let rawItems: any[] = [];
  let totalPages = 1;

  if (Array.isArray(raw)) {
    // 旧仕様
    rawItems = raw;
    totalPages = 1;
  } else if (raw && Array.isArray(raw.items)) {
    // PageResult<T>
    rawItems = raw.items;
    totalPages = Number(raw.totalPages ?? 1);
  } else {
    console.warn("[memberQuery] unexpected response shape:", raw);
  }

  const items = rawItems.map(normalizeMemberWire);

  return {
    items,
    totalPages,
  };
}

// ─────────────────────────────────────────────
// 単一メンバー取得
// ─────────────────────────────────────────────
export async function fetchMemberByIdWithToken(
  token: string,
  memberId: string,
): Promise<Member | null> {
  const id = String(memberId ?? "").trim();
  if (!id) return null;

  const url = apiUrl(`/members/${encodeURIComponent(id)}`);

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
      Accept: "application/json",
    },
  });

  const ct = res.headers.get("content-type") ?? "";
  if (!ct.includes("application/json")) return null;
  if (res.status === 404) return null;
  if (!res.ok) return null;

  const raw = await res.json();
  return normalizeMemberWire(raw);
}
