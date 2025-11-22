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
  fetchBrandsForCurrentMember,
} from "../../application/memberListService";

type FilterOption = { value: string; label: string };

// ソート方向（SortableTableHeader に合わせておく）
export type SortDirection = "asc" | "desc";

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

  // ソート状態
  const [sortKey, setSortKey] = useState<string | null>(null);
  const [sortDirection, setSortDirection] =
    useState<SortDirection>("desc");

  // 氏名（表示用）キャッシュ
  const [resolvedNames, setResolvedNames] = useState<Record<string, string>>({});

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
          perPage: targetPage.perPage ?? prev.perPage ?? DEFAULT_PAGE_LIMIT,
          totalPages: result.totalPages ?? prev.totalPages ?? 1,
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // ← load を依存に入れる必要なし（安定関数なので）

  // ─────────────────────────────────────────────
  // ブランド一覧（初回のみ読込） — アプリケーションサービス経由
  // ─────────────────────────────────────────────
  useEffect(() => {
    (async () => {
      try {
        const brands = await fetchBrandsForCurrentMember();

        const map: Record<string, string> = {};
        for (const b of brands) {
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
  // ページ番号変更（バックエンド側のページ）
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

  // ─────────────────────────────────────────────
  // 日付 → "YYYY/MM/DD" フォーマット
  // ─────────────────────────────────────────────
  const formatYmd = (date: any): string => {
    if (!date) return "";
    if (typeof date === "object" && date !== null) {
      if (typeof (date as any).toDate === "function") {
        return (date as any)
          .toDate()
          .toISOString()
          .slice(0, 10)
          .replace(/-/g, "/");
      }
      if (typeof (date as any).seconds === "number") {
        return new Date((date as any).seconds * 1000)
          .toISOString()
          .slice(0, 10)
          .replace(/-/g, "/");
      }
    }
    if (typeof date === "string") {
      return date.slice(0, 10).replace(/-/g, "/");
    }
    return "";
  };

  // ▼ ソート用：createdAt / updatedAt を number に変換
  const getDateValue = useCallback(
    (m: any): number => {
      const raw =
        sortKey === "updatedAt" ? (m as any).updatedAt : (m as any).createdAt;
      if (!raw) return 0;

      if (typeof raw === "object" && raw !== null) {
        if (typeof raw.toDate === "function") return raw.toDate().getTime();
        if (typeof raw.seconds === "number") return raw.seconds * 1000;
      }
      if (typeof raw === "string") {
        const t = new Date(raw).getTime();
        return Number.isNaN(t) ? 0 : t;
      }
      return 0;
    },
    [sortKey],
  );

  // ─────────────────────────────────────────────
  // フィルタ適用（ブランド & 権限）
  // ─────────────────────────────────────────────
  const filteredMembers = useMemo(() => {
    return members.filter((m) => {
      const assigned = m.assignedBrands ?? [];
      const categories = extractPermissionCategories(
        (m.permissions ?? []) as string[],
      );

      const matchesBrandFilter =
        selectedBrandIds.length === 0 ||
        assigned.some((brandId) => selectedBrandIds.includes(brandId));

      const matchesPermissionFilter =
        selectedPermissionCats.length === 0 ||
        categories.some((cat) => selectedPermissionCats.includes(cat));

      return matchesBrandFilter && matchesPermissionFilter;
    });
  }, [members, selectedBrandIds, selectedPermissionCats]);

  // ─────────────────────────────────────────────
  // ソート適用
  // ─────────────────────────────────────────────
  const sortedMembers = useMemo(() => {
    if (!sortKey) return filteredMembers;

    return [...filteredMembers].sort((a, b) => {
      const av = getDateValue(a);
      const bv = getDateValue(b);
      return sortDirection === "asc" ? av - bv : bv - av;
    });
  }, [filteredMembers, sortKey, sortDirection, getDateValue]);

  // ─────────────────────────────────────────────
  // 氏名補完（表示用名前の解決）
  // ─────────────────────────────────────────────
  useEffect(() => {
    let disposed = false;

    (async () => {
      const entries = await Promise.all(
        members.map(async (m) => {
          const inline = `${m.lastName ?? ""} ${m.firstName ?? ""}`.trim();
          if (inline) return [m.id, inline] as const;

          const resolved = await getNameLastFirstByID(m.id);
          return [m.id, resolved] as const;
        }),
      );

      if (!disposed) {
        const next: Record<string, string> = {};
        for (const [id, name] of entries) {
          if (name) next[id] = name;
        }
        setResolvedNames(next);
      }
    })();

    return () => {
      disposed = true;
    };
  }, [members, getNameLastFirstByID]);

  // ─────────────────────────────────────────────
  // ソート変更ハンドラ（ヘッダから呼ばれる）
  // ─────────────────────────────────────────────
  const handleSortChange = useCallback(
    (key: string, nextDirection: SortDirection | null) => {
      if (!nextDirection) {
        setSortKey(null);
        setSortDirection("desc");
        return;
      }
      setSortKey(key);
      setSortDirection(nextDirection);
    },
    [],
  );

  // ─────────────────────────────────────────────
  // Reset ボタン押下時の処理
  // ─────────────────────────────────────────────
  const handleReset = useCallback(() => {
    setSelectedBrandIds([]);
    setSelectedPermissionCats([]);
    setSortKey(null);
    setSortDirection("desc");
    setPageNumber(1);
  }, [setSelectedBrandIds, setSelectedPermissionCats, setPageNumber]);

  return {
    // 一覧（フィルタ＆ソート済み）
    members: sortedMembers,

    loading,
    error,

    // バックエンドページング
    page,
    setPage,
    setPageNumber,

    // ソート
    sortKey,
    sortDirection,
    handleSortChange,

    // リセット
    handleReset,

    // 氏名
    resolvedNames,
    getNameLastFirstByID, // 他の用途向けに残しておく

    // フィルタ関連
    brandMap,
    brandFilterOptions,
    permissionFilterOptions,

    selectedBrandIds,
    setSelectedBrandIds,
    selectedPermissionCats,
    setSelectedPermissionCats,

    extractPermissionCategories,

    // 日付フォーマッタ
    formatYmd,
  };
}
