// frontend/console/member/src/presentation/hooks/useMemberList.ts

import { useEffect, useState, useCallback, useMemo } from "react";

import type { Member } from "../../domain/entity/member";
import type { MemberFilter } from "../../domain/repository/memberRepository";
import type { Page } from "../../../../shell/src/shared/types/common/common";
import {
  DEFAULT_PAGE,
  DEFAULT_PAGE_LIMIT,
} from "../../../../shell/src/shared/types/common/common";

import {
  fetchMemberList,
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
  const [filter] = useState<MemberFilter>(initialFilter);

  // Page に totalPages を含めた構造
  const [page, setPage] = useState<Page>({
    ...DEFAULT_PAGE,
    ...(initialPage ?? {}),
    totalPages: 1,
  });

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  // ✅ リフレッシュボタン回転用（List の isResetting に渡す）
  const [isResetting, setIsResetting] = useState(false);

  // ブランドID→名称
  const [brandMap, setBrandMap] = useState<Record<string, string>>({});

  const [selectedBrandIds, setSelectedBrandIds] = useState<string[]>([]);
  const [selectedPermissionCats, setSelectedPermissionCats] = useState<string[]>(
    [],
  );

  // ソート状態
  const [sortKey, setSortKey] = useState<string | null>(null);
  const [sortDirection, setSortDirection] = useState<SortDirection>("desc");

  // ─────────────────────────────────────────────
  // メンバー一覧ロード（無限ループを防ぐため、依存は空配列）
  // ─────────────────────────────────────────────
  const load = useCallback(
    async (targetPage: Page, targetFilter: MemberFilter) => {
      setLoading(true);
      setIsResetting(true);
      setError(null);

      try {
        const result = await fetchMemberList(targetPage, targetFilter);

        setMembers(result.members ?? []);

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
        setIsResetting(false);
      }
    },
    [],
  );

  // 初回ロード（1回だけ）
  useEffect(() => {
    void load(page, filter);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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
  }, []);

  // ─────────────────────────────────────────────
  // ページ番号変更（バックエンド側のページ）
  // ─────────────────────────────────────────────
  const setPageNumber = useCallback(
    (pageNumber: number) => {
      const safe = Math.max(1, pageNumber);

      const nextPage = {
        ...page,
        number: safe,
      };

      setPage(nextPage);
      void load(nextPage, filter);
    },
    [page, filter, load],
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

    const nextPage = { ...page, number: 1 };
    setPage(nextPage);
    void load(nextPage, filter);
  }, [page, filter, load]);

  return {
    // 一覧（フィルタ＆ソート済み）
    members: sortedMembers,

    loading,
    error,

    // ✅ リフレッシュ回転用
    isResetting,

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