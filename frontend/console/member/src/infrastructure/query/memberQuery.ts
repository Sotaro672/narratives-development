// frontend/console/member/src/infrastructure/query/memberQuery.ts
/// <reference types="vite/client" />

import type { Member } from "../../domain/entity/member";
import type { MemberFilter } from "../../domain/repository/memberRepository";
import type { Page } from "../../../../shell/src/shared/types/common/common";
import { DEFAULT_PAGE_LIMIT } from "../../../../shell/src/shared/types/common/common";
import { buildConsoleUrl } from "../../../../shell/src/shared/http/apiBase";

// ─────────────────────────────────────────────
// URL builder（HTTP は叩かない）
// ─────────────────────────────────────────────
export function apiUrl(path: string, qs?: string): string {
  const url = buildConsoleUrl(path);
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

  if (useFilter.uid?.trim()) {
    params.set("uid", useFilter.uid.trim());
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
  return {
    id: String(w?.id ?? "").trim(),
    uid:
      w?.uid !== undefined && w?.uid !== null
        ? String(w.uid).trim()
        : null,

    firstName: w?.firstName ?? null,
    lastName: w?.lastName ?? null,
    firstNameKana: w?.firstNameKana ?? null,
    lastNameKana: w?.lastNameKana ?? null,

    displayName: w?.displayName ?? null,
    email: w?.email ?? null,

    permissions: Array.isArray(w?.permissions) ? w.permissions : [],
    assignedBrands: Array.isArray(w?.assignedBrands)
      ? w.assignedBrands
      : null,

    companyId: w?.companyId ?? null,
    status: w?.status ?? null,

    createdAt: w?.createdAt ?? "",
    updatedAt: w?.updatedAt ?? null,
    deletedAt: w?.deletedAt ?? null,
    deletedBy: w?.deletedBy ?? null,
    updatedBy: w?.updatedBy ?? null,
  };
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

  const rawItems: any[] = Array.isArray(raw)
    ? raw
    : Array.isArray(raw?.items)
      ? raw.items
      : [];

  const items: Member[] = rawItems.map((w: any) => normalizeMemberWire(w));

  return { items };
}