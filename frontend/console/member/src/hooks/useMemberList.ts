// frontend/member/src/hooks/useMemberList.ts
import { useEffect, useState, useCallback } from "react";

import type { Member } from "../domain/entity/member";
import type { MemberFilter } from "../domain/repository/memberRepository";
import type { Page } from "../../../shell/src/shared/types/common/common";
import {
  DEFAULT_PAGE,
  DEFAULT_PAGE_LIMIT,
} from "../../../shell/src/shared/types/common/common";

// ★ 認証（IDトークンを付けてバックエンドに問い合わせる）
import { auth } from "../../../shell/src/auth/config/firebaseClient";

// 環境変数からバックエンドのベースURLを取得（末尾スラッシュ除去）
const API_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/,
    ""
  ) ?? "";

/**
 * メンバー一覧取得用フック（バックエンドAPI経由版）
 * - /members?sort=updatedAt&order=desc&page=1&perPage=50&q=... などで取得
 * - companyId はクエリに付けず、サーバ（Usecase）が認証から強制スコープする
 * - 姓・名が両方未設定なら firstName に「招待中」を入れる
 */
export function useMemberList(
  initialFilter: MemberFilter = {},
  initialPage?: Page
) {
  const [members, setMembers] = useState<Member[]>([]);
  const [filter, setFilter] = useState<MemberFilter>(initialFilter);
  const [page, setPage] = useState<Page>(initialPage ?? DEFAULT_PAGE);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

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

    // サーバは page / perPage を受ける想定（handler の clampInt 実装に対応）
    const pageNumber = Math.floor(offset / perPage) + 1;

    const params = new URLSearchParams();
    params.set("page", String(pageNumber));
    params.set("perPage", String(perPage));

    // ソートは登録日（= createdAt）優先でなければ updatedAt desc をデフォルト
    params.set("sort", "updatedAt");
    params.set("order", "desc");

    // キーワード
    if (useFilter.searchQuery && useFilter.searchQuery.trim().length > 0) {
      params.set("q", useFilter.searchQuery.trim());
    }

    // brandIds（必要ならカンマ区切りで送る。バックエンド側は brandIds or brands を許容）
    if (useFilter.brandIds && useFilter.brandIds.length > 0) {
      params.set("brandIds", useFilter.brandIds.join(","));
    }

    // status
    if (useFilter.status && useFilter.status.trim().length > 0) {
      params.set("status", useFilter.status.trim());
    }

    // ★ companyId はここでは一切渡さない（サーバで強制スコープ）
    return params.toString();
  }

  const load = useCallback(
    async (override?: { page?: Page; filter?: MemberFilter }) => {
      setLoading(true);
      setError(null);
      try {
        const usePage = override?.page ?? page;
        const useFilter = override?.filter ?? filter;

        // Firebase ID トークンを取得
        const token = await auth.currentUser?.getIdToken();
        if (!token) {
          throw new Error("未認証のためメンバー一覧を取得できません。");
        }

        const qs = buildQuery(usePage, useFilter);
        const res = await fetch(`${API_BASE}/members?${qs}`, {
          method: "GET",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
        });

        if (!res.ok) {
          const text = await res.text().catch(() => "");
          throw new Error(
            `メンバー一覧の取得に失敗しました (status ${res.status}) ${text || ""}`
          );
        }

        // handler.list は res.Items ではなく items 配列をそのまま返却する実装
        const items = (await res.json()) as Member[];

        // 姓・名が未設定なら firstName を「招待中」に補正
        const normalized: Member[] = items.map((m) => {
          const noFirst =
            m.firstName === null || m.firstName === undefined || m.firstName === "";
          const noLast =
            m.lastName === null || m.lastName === undefined || m.lastName === "";
          if (noFirst && noLast) {
            return { ...m, firstName: "招待中" };
          }
          return m;
        });

        setMembers(normalized);
      } catch (e: any) {
        setError(e instanceof Error ? e : new Error(String(e)));
      } finally {
        setLoading(false);
      }
    },
    [page, filter]
  );

  useEffect(() => {
    void load();
  }, [load]);

  /**
   * 1始まりのページ番号を受けて Page(offset) を更新
   */
  const setPageNumber = (pageNumber: number) => {
    const safe = pageNumber > 0 ? pageNumber : 1;
    setPage((prev) => ({
      ...prev,
      offset: (safe - 1) * (prev.limit || DEFAULT_PAGE_LIMIT),
    }));
  };

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
  };
}
