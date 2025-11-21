// frontend/console/member/src/presentation/hooks/useMemberList.ts

import {
  useEffect,
  useState,
  useCallback,
  useRef,
  useMemo,
} from "react";

import type { Member } from "../../domain/entity/member";
import type { MemberFilter } from "../../domain/repository/memberRepository";
import type { Page } from "../../../../shell/src/shared/types/common/common";
import {
  DEFAULT_PAGE,
  DEFAULT_PAGE_LIMIT,
} from "../../../../shell/src/shared/types/common/common";

import {
  fetchMemberList,
  fetchMemberNameLastFirstById,
  fetchCurrentMember,
} from "../../application/memberListService";

import {
  listBrands,
  type BrandRow,
} from "../../../../brand/src/application/brandService";

type FilterOption = { value: string; label: string };

export function useMemberList(
  initialFilter: MemberFilter = {},
  initialPage?: Page,
) {
  const [members, setMembers] = useState<Member[]>([]);
  const [filter, setFilter] = useState<MemberFilter>(initialFilter);

  // Page に totalPages を含めた構造
  const [page, setPage] = useState<Page>({
    ...DEFAULT_PAGE,
    ...(initialPage ?? {}),
    totalPages: 1,
  });

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const nameCacheRef = useRef<Map<string, string>>(new Map());

  // ブランドID→名称
  const [brandMap, setBrandMap] = useState<Record<string, string>>({});

  const [selectedBrandIds, setSelectedBrandIds] = useState<string[]>([]);
  const [selectedPermissionCats, setSelectedPermissionCats] = useState<string[]>([]);

  // ─────────────────────────────────────────────
  // メンバー一覧ロード（無限ループを防ぐため、依存は空配列）
  // ─────────────────────────────────────────────
  const load = useCallback(
    async (targetPage: Page, targetFilter: MemberFilter) => {
      setLoading(true);
      setError(null);

      try {
        const result = await fetchMemberList(targetPage, targetFilter);

        setMembers(result.members ?? []);

        // 氏名キャッシュ
        const cache = nameCacheRef.current;
        for (const [id, disp] of Object.entries(result.nameMap ?? {})) {
          cache.set(id, disp);
        }

        // ページング更新
        setPage((prev) => ({
          ...prev,
          number: targetPage.number,
          perPage: targetPage.perPage,
          totalPages: result.totalPages ?? prev.totalPages,
        }));
      } catch (e) {
        const err = e instanceof Error ? e : new Error(String(e));
        console.error("[useMemberList] load error:", err);
        setError(err);
      } finally {
        setLoading(false);
      }
    },
    [], // ← 無限ループ回避：page/filter に依存しない！
  );

  // 初回ロード（1回だけ）
  useEffect(() => {
    void load(page, filter);
  }, []); // ← load を依存に入れる必要なし（安定関数なので）

  // ─────────────────────────────────────────────
  // ブランド一覧（初回のみ読込）
  // ─────────────────────────────────────────────
  useEffect(() => {
    (async () => {
      try {
        const current = await fetchCurrentMember();
        const companyId = String(current?.companyId ?? "").trim();

        if (!companyId) {
          console.warn("[useMemberList] companyId が取得できませんでした");
          setBrandMap({});
          return;
        }

        const rows: BrandRow[] = await listBrands(companyId);

        console.log("[useMemberList] brand rows for filter header =", rows);

        const map: Record<string, string> = {};
        for (const b of rows) {
          map[b.id] = b.name;
        }
        setBrandMap(map);
      } catch (e) {
        console.error("[useMemberList] failed to load brands", e);
        setBrandMap({});
      }
    })();
  }, []); // ← 依存なし：初回一回だけ

  // ─────────────────────────────────────────────
  // ページ番号変更
  // ─────────────────────────────────────────────
  const setPageNumber = (pageNumber: number) => {
    const safe = Math.max(1, pageNumber);

    const nextPage = {
      ...page,
      number: safe,
    };

    setPage(nextPage);
    void load(nextPage, filter);
  };

  // ─────────────────────────────────────────────
  // MemberID → 氏名
  // ─────────────────────────────────────────────
  const getNameLastFirstByID = useCallback(
    async (memberId: string): Promise<string> => {
      const id = memberId.trim();
      if (!id) return "";

      const cache = nameCacheRef.current;
      if (cache.has(id)) return cache.get(id)!;

      const disp = await fetchMemberNameLastFirstById(id);
      if (disp) cache.set(id, disp);
      return disp;
    },
    [],
  );

  // ─────────────────────────────────────────────
  // 権限カテゴリ抽出
  // ─────────────────────────────────────────────
  const extractPermissionCategories = (perms?: string[]): string[] => {
    if (!perms || perms.length === 0) return [];
    const set = new Set<string>();

    for (const p of perms) {
      const dot = p.indexOf(".");
      const cat = dot > 0 ? p.slice(0, dot) : p;
      if (cat) set.add(cat);
    }
    return Array.from(set);
  };

  // ─────────────────────────────────────────────
  // フィルタ候補（ブランド）
  // ─────────────────────────────────────────────
  const brandFilterOptions: FilterOption[] = useMemo(
    () =>
      Object.entries(brandMap).map(([id, name]) => ({
        value: id,
        label: name || id,
      })),
    [brandMap],
  );

  // ─────────────────────────────────────────────
  // フィルタ候補（権限カテゴリ）
  // ─────────────────────────────────────────────
  const permissionFilterOptions: FilterOption[] = useMemo(() => {
    const set = new Set<string>();

    for (const m of members) {
      const cats = extractPermissionCategories(m.permissions ?? []);
      for (const c of cats) set.add(c);
    }

    return Array.from(set).map((c) => ({ value: c, label: c }));
  }, [members]);

  return {
    members,
    loading,
    error,
    filter,
    setFilter,

    // ページング
    page,
    setPage,
    setPageNumber,

    // 氏名
    getNameLastFirstByID,

    // ブランド
    brandMap,

    brandFilterOptions,
    permissionFilterOptions,

    selectedBrandIds,
    setSelectedBrandIds,
    selectedPermissionCats,
    setSelectedPermissionCats,

    extractPermissionCategories,
  };
}
