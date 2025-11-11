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

/**
 * Repository 実装をここで束ねる
 * 将来別実装（REST / GraphQL 等）に差し替える場合もこの1行を変更すればOK。
 */
const repository: MemberRepository = new MemberRepositoryFS();

/**
 * メンバー一覧取得用フック
 * - MemberRepository.list(page, filter) を利用
 * - limit/offset ベースのシンプルなページング + フィルタ管理
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

  const load = useCallback(
    async (override?: { page?: Page; filter?: MemberFilter }) => {
      setLoading(true);
      setError(null);
      try {
        const usePage = override?.page ?? page;
        const useFilter = override?.filter ?? filter;
        const result = await repository.list(usePage, useFilter);
        setMembers(result.items);
      } catch (e: any) {
        setError(e);
      } finally {
        setLoading(false);
      }
    },
    [page, filter]
  );

  useEffect(() => {
    // 初回 & 条件変更時にロード
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
