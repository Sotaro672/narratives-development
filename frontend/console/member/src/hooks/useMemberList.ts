// frontend/member/src/hooks/useMemberList.ts
import { useEffect, useState, useCallback } from "react";

import type { Member } from "../domain/entity/member";
import type {
  MemberRepository,
  MemberFilter,
} from "../domain/repository/memberRepository";
import type { Page } from "../../../shell/src/shared/types/common/common";
import {
  DEFAULT_PAGE,
  DEFAULT_PAGE_LIMIT,
} from "../../../shell/src/shared/types/common/common";
import { MemberRepositoryFS } from "../infrastructure/firestore/memberRepositoryFS";

// ★ 認証情報（companyId を得るために利用）
import { useAuthContext } from "../../../shell/src/auth/application/AuthContext";

/**
 * Repository 実装をここで束ねる
 * 将来別実装（REST / GraphQL 等）に差し替える場合もこの1行を変更すればOK。
 */
const repository: MemberRepository = new MemberRepositoryFS();

/**
 * メンバー一覧取得用フック
 * - MemberRepository.list(page, filter) を利用
 * - limit/offset ベースのシンプルなページング + フィルタ管理
 * - ★ フィルタで companyId が未指定なら、ログインユーザーの companyId を自動で付与
 * - ★ 姓・名がどちらも未設定のときは firstName に「招待中」を入れて返す
 */
export function useMemberList(
  initialFilter: MemberFilter = {},
  initialPage?: Page
) {
  const { user } = useAuthContext(); // ← auth から companyId を取得
  const authCompanyId = user?.companyId ?? null;

  const [members, setMembers] = useState<Member[]>([]);
  const [filter, setFilter] = useState<MemberFilter>(initialFilter);
  const [page, setPage] = useState<Page>(initialPage ?? DEFAULT_PAGE);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const load = useCallback(
    async (override?: { page?: Page; filter?: MemberFilter }) => {
      setLoading(true);
      setError(null);
      try {
        const usePage = override?.page ?? page;

        // ▼ フィルタを合成し、companyId が未指定なら auth の companyId を補完
        const baseFilter = override?.filter ?? filter;
        const useFilter: MemberFilter = {
          ...baseFilter,
          ...(baseFilter.companyId
            ? {}
            : authCompanyId
            ? { companyId: authCompanyId }
            : {}),
        };

        const result = await repository.list(usePage, useFilter);

        // ★ 姓・名がどちらも未設定のときは firstName に「招待中」を入れて返す
        const normalized: Member[] = result.items.map((m) => {
          const noFirst =
            m.firstName === null ||
            m.firstName === undefined ||
            m.firstName === "";
          const noLast =
            m.lastName === null ||
            m.lastName === undefined ||
            m.lastName === "";

          if (noFirst && noLast) {
            return {
              ...m,
              firstName: "招待中",
            };
          }
          return m;
        });

        setMembers(normalized);
      } catch (e: any) {
        setError(e);
      } finally {
        setLoading(false);
      }
    },
    [page, filter, authCompanyId]
  );

  useEffect(() => {
    // 初回 & 条件変更 & companyId 取得後にロード
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
