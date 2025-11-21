// frontend/console/member/src/presentation/hooks/useMemberList.ts

import { useEffect, useState, useCallback, useRef } from "react";

import type { Member } from "../../domain/entity/member";
import type { MemberFilter } from "../../domain/repository/memberRepository";
import type { Page } from "../../../../shell/src/shared/types/common/common";
import {
  DEFAULT_PAGE,
  DEFAULT_PAGE_LIMIT,
} from "../../../../shell/src/shared/types/common/common";

// ★ アプリケーションサービスへ処理を委譲
import {
  fetchMemberList,
  fetchMemberNameLastFirstById,
} from "../../application/memberListService";

// ★ ログイン情報（companyId）とブランド一覧取得用サービス
import { useAuthContext } from "../../../../shell/src/auth/application/AuthContext";
import {
  listBrands,
  type BrandRow,
} from "../../../../brand/src/application/brandService";

/**
 * メンバー一覧取得用フック（バックエンドAPI経由版）
 * - /members?sort=updatedAt&order=desc&page=1&perPage=50&q=... などで取得
 * - companyId はクエリに付けず、サーバ（Usecase）が認証から強制スコープする
 * - 姓・名が両方未設定なら firstName に「招待中」を入れる（一覧表示の利便性）
 * - ★ 追加: memberID → 「姓 名」を解決する getNameLastFirstByID を提供
 * - ★ 追加: brandId -> brandName を解決する brandMap を提供
 */
export function useMemberList(
  initialFilter: MemberFilter = {},
  initialPage?: Page,
) {
  // 認証中ユーザ（companyId をフロントでも把握しておく）
  const { user } = useAuthContext();
  const authCompanyId = user?.companyId ?? null;

  const [members, setMembers] = useState<Member[]>([]);
  const [filter, setFilter] = useState<MemberFilter>(initialFilter);
  const [page, setPage] = useState<Page>(initialPage ?? DEFAULT_PAGE);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  // ID→表示名のキャッシュ（コンポーネント存続中は保持）
  const nameCacheRef = useRef<Map<string, string>>(new Map());

  // brandId -> brandName のマップ
  const [brandMap, setBrandMap] = useState<Record<string, string>>({});

  const load = useCallback(
    async (override?: { page?: Page; filter?: MemberFilter }) => {
      setLoading(true);
      setError(null);
      try {
        const usePage = override?.page ?? page;
        const useFilter = override?.filter ?? filter;

        // ★ アプリケーションサービスに委譲
        const { members: normalized, nameMap } = await fetchMemberList(
          usePage,
          useFilter,
        );

        // 名前キャッシュ更新
        const cache = nameCacheRef.current;
        for (const [id, disp] of Object.entries(nameMap)) {
          cache.set(id, disp);
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

  // マウント時 / companyId 変更時にブランド一覧を取得
  useEffect(() => {
    if (!authCompanyId) return;

    (async () => {
      try {
        // brandService.ts で既に動作確認済みの listBrands を利用
        const rows: BrandRow[] = await listBrands(authCompanyId);

        const map: Record<string, string> = {};
        for (const b of rows) {
          map[b.id] = b.name;
        }

        // eslint-disable-next-line no-console
        console.log("[useMemberList] brandMap =", map);
        setBrandMap(map);
      } catch (e) {
        // eslint-disable-next-line no-console
        console.error("[useMemberList] failed to load brands", e);
        setBrandMap({});
      }
    })();
  }, [authCompanyId]);

  const setPageNumber = (pageNumber: number) => {
    const safe = pageNumber > 0 ? pageNumber : 1;
    setPage((prev) => ({
      ...prev,
      number: safe,
      perPage: prev.perPage ?? DEFAULT_PAGE_LIMIT,
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

      // ★ 単体取得はアプリケーションサービスに委譲
      const disp = await fetchMemberNameLastFirstById(id);
      if (disp) cache.set(id, disp);
      return disp;
    },
    [],
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
    brandMap, // brandId -> brandName
  };
}
