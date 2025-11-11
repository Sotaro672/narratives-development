// frontend/member/src/hooks/useMemberList.ts
import { useEffect, useState, useCallback } from "react";

import type { Member } from "../domain/entity/member";
import type {
  MemberRepository,
  MemberFilter,
} from "../domain/repository/memberRepository";
import type {
  Page,
} from "../../../shell/src/shared/types/common/common";
import { getMemberRepository } from "../infrastructure/firestore/memberRepositoryFS";

/**
 * メンバー一覧取得用フック
 * - MemberRepository.list(page, filter) を利用
 * - シンプルなページング + フィルタ管理
 */
export function useMemberList(
  initialFilter: MemberFilter = {},
  initialPage?: Page
) {
  const repo: MemberRepository = getMemberRepository();

  const [members, setMembers] = useState<Member[]>([]);
  const [filter, setFilter] = useState<MemberFilter>(initialFilter);
  const [page, setPage] = useState<Page>(
    // Page の実体形は shared 側に依存するため any で初期値を与える
    (initialPage ??
      ({
        page: 1,
        perPage: 20,
      } as any)) as Page
  );
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const load = useCallback(
    async (override?: { page?: Page; filter?: MemberFilter }) => {
      setLoading(true);
      setError(null);
      try {
        const usePage = override?.page ?? page;
        const useFilter = override?.filter ?? filter;
        const result = await repo.list(usePage, useFilter);
        setMembers(result.items);
      } catch (e: any) {
        setError(e);
      } finally {
        setLoading(false);
      }
    },
    [repo, page, filter]
  );

  useEffect(() => {
    // 初回 & 条件変更時にロード
    void load();
  }, [load]);

  return {
    members,
    loading,
    error,
    filter,
    setFilter,
    page,
    setPage,
    reload: () => load(),
    // ページ変更のユーティリティ（Page の実装に依存するので any 経由）
    setPageNumber: (pageNumber: number) =>
      setPage((prev) => {
        const p: any = { ...prev };
        if ("page" in p) p.page = pageNumber;
        if ("number" in p) p.number = pageNumber;
        // offset 系の場合は呼び出し側で調整
        return p as Page;
      }),
  };
}
