// frontend/console/member/src/infrastructure/query/memberQuery.ts
/// <reference types="vite/client" />

import type { Member } from "../../domain/entity/member";
import type { MemberFilter } from "../../domain/repository/memberRepository";
import type { Page } from "../../../../shell/src/shared/types/common/common";
import { DEFAULT_PAGE_LIMIT } from "../../../../shell/src/shared/types/common/common";

// ─────────────────────────────────────────────
// Backend base URL（Query 層では参照のみ）
// ─────────────────────────────────────────────
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)
    ?.replace(/\/+$/g, "") ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = (ENV_BASE || FALLBACK_BASE).replace(/\/+$/g, "");

// ─────────────────────────────────────────────
// URL builder（HTTP は叩かない）
// ─────────────────────────────────────────────
export function apiUrl(path: string, qs?: string) {
  const url = `${API_BASE}${path.startsWith("/") ? path : `/${path}`}`;
  return qs ? `${url}?${qs}` : url;
}

// ─────────────────────────────────────────────
// Query String Builder（HTTP なし）
// ─────────────────────────────────────────────
export function buildMemberQuery(
  usePage: Page,
  useFilter: MemberFilter,
): string {
  const perPage =
    usePage?.perPage && usePage.perPage > 0
      ? usePage.perPage
      : DEFAULT_PAGE_LIMIT;

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
// FormatLastFirst（表示用の姓名整形）
// ─────────────────────────────────────────────
export function formatLastFirst(
  lastName?: string | null,
  firstName?: string | null,
): string {
  const ln = String(lastName ?? "").trim();
  const fn = String(firstName ?? "").trim();
  if (ln && fn) return `${ln} ${fn}`;
  if (ln) return ln;
  if (fn) return fn;
  return "";
}

// ─────────────────────────────────────────────
// データ整形（HTTP なし）
// ─────────────────────────────────────────────
export function normalizeMemberWire(w: any): Member {
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
    createdAt: w.createdAt ?? w.CreatedAt ?? null,
    updatedAt: w.updatedAt ?? w.UpdatedAt ?? null,
    deletedAt: w.deletedAt ?? w.DeletedAt ?? null,
    deletedBy: w.deletedBy ?? w.DeletedBy ?? null,
    updatedBy: w.updatedBy ?? w.UpdatedBy ?? null,
  } as Member;
}

// ─────────────────────────────────────────────
// HTTP 呼び出し付き メンバー一覧取得
//   - useAdminCard などから利用
// ─────────────────────────────────────────────
export async function fetchMemberListWithToken(
  token: string,
  page: Page,
  filter: MemberFilter,
): Promise<{ items: Member[] }> {
  const qs = buildMemberQuery(page, filter);
  const url = apiUrl("/members", qs);

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
  });

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `fetchMemberListWithToken failed: ${res.status} ${res.statusText}${
        body ? ` - ${body}` : ""
      }`,
    );
  }

  const raw = await res.json();

  // backend が { items: [...] } を返す場合と、配列そのものを返す場合の両対応
  const rawItems: any[] = Array.isArray(raw)
    ? raw
    : Array.isArray(raw?.items)
    ? raw.items
    : [];

  const items: Member[] = rawItems.map((w: any) => normalizeMemberWire(w));

  return { items };
}
