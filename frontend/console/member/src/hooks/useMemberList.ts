// frontend/member/src/hooks/useMemberList.ts
import { useEffect, useState, useCallback, useRef } from "react";

import type { Member } from "../domain/entity/member";
import type { MemberFilter } from "../domain/repository/memberRepository";
import type { Page } from "../../../shell/src/shared/types/common/common";
import {
  DEFAULT_PAGE,
  DEFAULT_PAGE_LIMIT,
} from "../../../shell/src/shared/types/common/common";

// ★ 認証（IDトークンを付けてバックエンドに問い合わせる）
import { auth } from "../../../shell/src/auth/config/firebaseClient";

// ─────────────────────────────────────────────
// Backend base URL（.env 未設定でも Cloud Run にフォールバック）
// ─────────────────────────────────────────────
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

const API_BASE = (ENV_BASE || FALLBACK_BASE).replace(/\/+$/, "");

// ログ付き URL 組み立て
function apiUrl(path: string, qs?: string) {
  const url = `${API_BASE}${path.startsWith("/") ? path : `/${path}`}`;
  const full = qs ? `${url}?${qs}` : url;
  // eslint-disable-next-line no-console
  console.log("[useMemberList] GET", full);
  return full;
}

// ─────────────────────────────────────────────
// member.Service の挙動をTSで再現（FormatLastFirst）
// ─────────────────────────────────────────────
function formatLastFirst(lastName?: string | null, firstName?: string | null) {
  const ln = String(lastName ?? "").trim();
  const fn = String(firstName ?? "").trim();
  if (ln && fn) return `${ln} ${fn}`;
  if (ln) return ln;
  if (fn) return fn;
  return "";
}

// 受信JSONを camelCase に寄せるワイヤ正規化（PascalCase も吸収）
function normalizeMemberWire(w: any): Member {
  // まず候補を拾う（camelCase 優先、無ければ PascalCase）
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

  // createdAt / updatedAt は文字列/秒/Firestore Timestamp など混在に対応
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

/**
 * メンバー一覧取得用フック（バックエンドAPI経由版）
 * - /members?sort=updatedAt&order=desc&page=1&perPage=50&q=... などで取得
 * - companyId はクエリに付けず、サーバ（Usecase）が認証から強制スコープする
 * - 姓・名が両方未設定なら firstName に「招待中」を入れる（一覧表示の利便性）
 * - ★ 追加: memberID → 「姓 名」を解決する getNameLastFirstByID を提供
 */
export function useMemberList(
  initialFilter: MemberFilter = {},
  initialPage?: Page,
) {
  const [members, setMembers] = useState<Member[]>([]);
  const [filter, setFilter] = useState<MemberFilter>(initialFilter);
  const [page, setPage] = useState<Page>(initialPage ?? DEFAULT_PAGE);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  // ID→表示名のキャッシュ（コンポーネント存続中は保持）
  const nameCacheRef = useRef<Map<string, string>>(new Map());

  // クエリ文字列を作る（companyId は絶対に付けない）
  function buildQuery(usePage: Page, useFilter: MemberFilter): string {
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

    if (useFilter.searchQuery?.trim()) params.set("q", useFilter.searchQuery.trim());
    if (useFilter.brandIds?.length) params.set("brandIds", useFilter.brandIds.join(","));
    if (useFilter.status?.trim()) params.set("status", useFilter.status.trim());

    return params.toString();
  }

  const load = useCallback(
    async (override?: { page?: Page; filter?: MemberFilter }) => {
      setLoading(true);
      setError(null);
      try {
        const usePage = override?.page ?? page;
        const useFilter = override?.filter ?? filter;

        const token = await auth.currentUser?.getIdToken?.();
        if (!token) throw new Error("未認証のためメンバー一覧を取得できません。（IDトークン未取得）");

        const qs = buildQuery(usePage, useFilter);
        const url = apiUrl("/members", qs);

        const res = await fetch(url, {
          method: "GET",
          headers: { Authorization: `Bearer ${token}`, Accept: "application/json" },
        });

        const ct = res.headers.get("content-type") ?? "";
        if (!ct.includes("application/json")) {
          const text = await res.text().catch(() => "");
          const head = text.slice(0, 160).replace(/\s+/g, " ");
          throw new Error(`Unexpected content-type: ${ct} (url=${url}) body_head="${head}"`);
        }
        if (!res.ok) {
          const text = await res.text().catch(() => "");
          if (res.status === 401 || res.status === 403) {
            throw new Error(`認証/認可エラー (${res.status}). 再ログイン後に再試行してください。 ${text || ""}`);
          }
          throw new Error(`メンバー一覧の取得に失敗しました (status ${res.status}) ${text || ""}`);
        }

        const raw = (await res.json()) as any[];
        // ←★ ここで正規化して camelCase に寄せる
        const items = raw.map(normalizeMemberWire);

        // 姓・名が未設定なら firstName を「招待中」に補正（UI用途）
        const normalized: Member[] = items.map((m) => {
          const noFirst = !m.firstName?.trim();
          const noLast = !m.lastName?.trim();
          if (noFirst && noLast) return { ...m, firstName: "招待中" };
          return m;
        });

        // 名前キャッシュ
        const cache = nameCacheRef.current;
        for (const m of items) {
          const disp = formatLastFirst(m.lastName, m.firstName);
          if (disp) cache.set(m.id, disp);
        }

        setMembers(normalized);
      } catch (e: any) {
        setError(e instanceof Error ? e : new Error(String(e)));
      } finally {
        setLoading(false);
      }
    },
    [page, filter],
  );

  useEffect(() => {
    void load();
  }, [load]);

  /** 1始まりのページ番号を受けて Page(offset) を更新 */
  const setPageNumber = (pageNumber: number) => {
    const safe = pageNumber > 0 ? pageNumber : 1;
    setPage((prev) => ({
      ...prev,
      offset: (safe - 1) * (prev.limit || DEFAULT_PAGE_LIMIT),
    }));
  };

  // ID → 「姓 名」を解決（member.Service.GetNameLastFirstByID 相当）
  const getNameLastFirstByID = useCallback(async (memberId: string): Promise<string> => {
    const id = String(memberId ?? "").trim();
    if (!id) return "";

    const cache = nameCacheRef.current;
    const cached = cache.get(id);
    if (cached !== undefined) return cached;

    const existing = members.find((m) => m.id === id);
    if (existing) {
      const disp = formatLastFirst(existing.lastName, existing.firstName);
      if (disp) cache.set(id, disp);
      return disp;
    }

    const token = await auth.currentUser?.getIdToken?.();
    if (!token) return "";

    const url = apiUrl(`/members/${encodeURIComponent(id)}`);
    const res = await fetch(url, {
      headers: { Authorization: `Bearer ${token}`, Accept: "application/json" },
    });

    const ct = res.headers.get("content-type") ?? "";
    if (!ct.includes("application/json")) return "";
    if (res.status === 404) return "";
    if (!res.ok) return "";

    const m = normalizeMemberWire(await res.json());
    const disp = formatLastFirst(m.lastName, m.firstName);
    if (disp) cache.set(id, disp);
    return disp;
  }, [members]);

  return {
    members,
    loading,
    error,
    filter,
    setFilter,
    page,
    setPage,
    reload: () => load(),
    setPageNumber,
    getNameLastFirstByID,
  };
}
