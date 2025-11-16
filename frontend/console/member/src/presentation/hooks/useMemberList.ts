// frontend/member/src/hooks/useMemberList.ts
import { useEffect, useState, useCallback, useRef } from "react";

import type { Member } from "../../domain/entity/member";
import type { MemberFilter } from "../../domain/repository/memberRepository";
import type { Page } from "../../../../shell/src/shared/types/common/common";
import {
  DEFAULT_PAGE,
  DEFAULT_PAGE_LIMIT,
} from "../../../../shell/src/shared/types/common/common";

// 認証（IDトークン取得用）
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// ★ API 呼び出しロジックは infrastructure/query 側に委譲
import {
  fetchMemberListWithToken,
  fetchMemberByIdWithToken,
  formatLastFirst,
} from "../../infrastructure/query/memberQuery";

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

  const load = useCallback(
    async (override?: { page?: Page; filter?: MemberFilter }) => {
      setLoading(true);
      setError(null);
      try {
        const usePage = override?.page ?? page;
        const useFilter = override?.filter ?? filter;

        // Firebase Auth から ID トークンを取得
        const currentUser = auth.currentUser;
        if (!currentUser) {
          throw new Error(
            "未認証のためメンバー一覧を取得できません。（currentUser が null）",
          );
        }
        const token = await currentUser.getIdToken();
        // eslint-disable-next-line no-console
        console.log("[useMemberList] currentUser.uid:", currentUser.uid);

        // ★ API 呼び出しは infrastructure/query に委譲
        const { items } = await fetchMemberListWithToken(token, usePage, useFilter);

        // 姓・名が未設定なら firstName を「招待中」に補正（UI用途）
        const normalized: Member[] = items.map((m) => {
          const noFirst = !String(m.firstName ?? "").trim();
          const noLast = !String(m.lastName ?? "").trim();
          if (noFirst && noLast) return { ...m, firstName: "招待中" } as Member;
          return m;
        });

        // 名前キャッシュ
        const cache = nameCacheRef.current;
        for (const m of items) {
          const disp = formatLastFirst(m.lastName as any, m.firstName as any);
          if (disp) cache.set(m.id, disp);
        }

        setMembers(normalized);
      } catch (e: any) {
        const err = e instanceof Error ? e : new Error(String(e));
        // eslint-disable-next-line no-console
        console.error("[useMemberList] load error:", err);
        setError(err);
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
  const getNameLastFirstByID = useCallback(
    async (memberId: string): Promise<string> => {
      const id = String(memberId ?? "").trim();
      if (!id) return "";

      const cache = nameCacheRef.current;
      const cached = cache.get(id);
      if (cached !== undefined) return cached;

      const existing = members.find((m) => m.id === id);
      if (existing) {
        const disp = formatLastFirst(
          existing.lastName as any,
          existing.firstName as any,
        );
        if (disp) cache.set(id, disp);
        return disp;
      }

      const currentUser = auth.currentUser;
      if (!currentUser) return "";
      const token = await currentUser.getIdToken();

      const member = await fetchMemberByIdWithToken(token, id);
      if (!member) return "";

      const disp = formatLastFirst(
        member.lastName as any,
        member.firstName as any,
      );
      if (disp) cache.set(id, disp);
      return disp;
    },
    [members],
  );

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
