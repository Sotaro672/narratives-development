// frontend/member/src/infrastructure/query/memberQuery.ts
/// <reference types="vite/client" />

import type { Member } from "../../domain/entity/member";
import type { MemberFilter } from "../../domain/repository/memberRepository";
import type { Page } from "../../../../shell/src/shared/types/common/common";
import { DEFAULT_PAGE_LIMIT } from "../../../../shell/src/shared/types/common/common";

// ─────────────────────────────────────────────
// Backend base URL（.env 未設定でも Cloud Run にフォールバック）
// ─────────────────────────────────────────────
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = (ENV_BASE || FALLBACK_BASE).replace(/\/+$/g, "");

// ログ付き URL 組み立て
function apiUrl(path: string, qs?: string) {
  const url = `${API_BASE}${path.startsWith("/") ? path : `/${path}`}`;
  const full = qs ? `${url}?${qs}` : url;
  // eslint-disable-next-line no-console
  console.log("[memberQuery] GET", full);
  return full;
}

// クエリ文字列を作る（companyId は絶対に付けない）
export function buildMemberQuery(usePage: Page, useFilter: MemberFilter): string {
  const perPage =
    usePage && typeof usePage.limit === "number" && usePage.limit > 0
      ? usePage.limit
      : DEFAULT_PAGE_LIMIT;

  const offset =
    usePage && typeof usePage.offset === "number" && usePage.offset >= 0
      ? usePage.offset
      : 0;

  const pageNumber = Math.floor(offset / perPage) + 1;

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
// member.Service の挙動をTSで再現（FormatLastFirst）
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

// 受信JSONを camelCase に寄せるワイヤ正規化（PascalCase も吸収）
function normalizeMemberWire(w: any): Member {
  const id = String(w.id ?? w.ID ?? "").trim();
  const firstName = (w.firstName ?? w.FirstName ?? null) as string | null;
  const lastName = (w.lastName ?? w.LastName ?? null) as string | null;
  const firstNameKana = (w.firstNameKana ?? w.FirstNameKana ?? null) as string | null;
  const lastNameKana = (w.lastNameKana ?? w.LastNameKana ?? null) as string | null;
  const email = (w.email ?? w.Email ?? null) as string | null;
  const companyId = (w.companyId ?? w.CompanyID ?? "") as string;
  const permissions = (w.permissions ?? w.Permissions ?? []) as string[];
  const assignedBrands =
    (w.assignedBrands ?? w.AssignedBrands ?? null) as string[] | null;

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
    permissions: Array.isArray(permissions) ? permissions : [],
    assignedBrands: Array.isArray(assignedBrands) ? assignedBrands : null,
    createdAt,
    updatedAt,
    deletedAt,
    deletedBy,
    updatedBy,
  } as Member;
}

export type MemberListResult = {
  items: Member[];
};

/**
 * メンバー一覧取得（純粋なクエリ関数）
 * - React Hook に依存しない
 * - トークンは呼び出し側（Hookなど）から渡す
 */
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

  // eslint-disable-next-line no-console
  console.log("[memberQuery] response", res.status, res.statusText);

  const ct = res.headers.get("content-type") ?? "";
  if (!ct.includes("application/json")) {
    const text = await res.text().catch(() => "");
    const head = text.slice(0, 160).replace(/\s+/g, " ");
    throw new Error(
      `Unexpected content-type: ${ct} (url=${url}) body_head="${head}"`,
    );
  }

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    if (res.status === 401 || res.status === 403) {
      throw new Error(
        `認証/認可エラー (${res.status}). 再ログイン後に再試行してください。 ${
          text || ""
        }`,
      );
    }
    throw new Error(
      `メンバー一覧の取得に失敗しました (status ${res.status}) ${text || ""}`,
    );
  }

  const raw = (await res.json()) as any[];
  const items = raw.map(normalizeMemberWire);
  return { items };
}

/**
 * 単一メンバー取得（ID指定）
 * - useMemberList の getNameLastFirstByID から利用
 */
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
