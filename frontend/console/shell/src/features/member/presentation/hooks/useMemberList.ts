// frontend/console/shell/src/features/member/presentation/hooks/useMemberList.ts

import { useCallback, useEffect, useMemo, useState } from "react";

import type { Member, MemberFilter } from "../../../../shared/types/member";
import type { PageState } from "../../../../shared/types/common/common";
import {
  DEFAULT_PAGE,
  DEFAULT_PAGE_LIMIT,
} from "../../../../shared/types/common/common";

import {
  fetchMemberList,
  fetchBrandsForCurrentMember,
} from "../../application/memberListService";

type FilterOption = {
  value: string;
  label: string;
};

export type SortDirection = "asc" | "desc";

export function useMemberList(
  initialFilter: MemberFilter = {},
  initialPage?: PageState,
) {
  const [members, setMembers] = useState<Member[]>([]);
  const [filter] = useState<MemberFilter>(initialFilter);

  const [page, setPage] = useState<PageState>({
    ...DEFAULT_PAGE,
    ...(initialPage ?? {}),
    totalPages: 1,
  });

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [isResetting, setIsResetting] = useState(false);

  const [brandMap, setBrandMap] = useState<Record<string, string>>({});
  const [selectedBrandIds, setSelectedBrandIds] = useState<string[]>([]);
  const [selectedPermissionCats, setSelectedPermissionCats] = useState<string[]>([]);

  const [sortKey, setSortKey] = useState<string | null>(null);
  const [sortDirection, setSortDirection] = useState<SortDirection>("desc");

  // ─────────────────────────────────────────────
  // メンバー一覧ロード
  // ─────────────────────────────────────────────

  const load = useCallback(
    async (
      targetPage: PageState,
      targetFilter: MemberFilter,
      resetting: boolean,
    ) => {
      if (resetting) {
        setIsResetting(true);
      } else {
        setLoading(true);
      }

      setError(null);

      try {
        const result = await fetchMemberList(targetPage, targetFilter);

        setMembers(result.items);

        setPage((previousPage) => ({
          ...previousPage,
          number: result.page,
          perPage: result.perPage ?? DEFAULT_PAGE_LIMIT,
          totalPages: result.totalPages,
        }));
      } catch (loadError: unknown) {
        const normalizedError =
          loadError instanceof Error
            ? loadError
            : new Error(String(loadError));

        console.error("[useMemberList] load error:", normalizedError);
        setError(normalizedError);
      } finally {
        if (resetting) {
          setIsResetting(false);
        } else {
          setLoading(false);
        }
      }
    },
    [],
  );

  useEffect(() => {
    void load(page, filter, false);

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // ─────────────────────────────────────────────
  // ブランド一覧ロード
  // ─────────────────────────────────────────────

  useEffect(() => {
    let cancelled = false;

    async function loadBrands() {
      try {
        const brands = await fetchBrandsForCurrentMember();

        if (cancelled) {
          return;
        }

        const nextBrandMap: Record<string, string> = {};

        for (const brand of brands) {
          nextBrandMap[brand.id] = brand.name;
        }

        setBrandMap(nextBrandMap);
      } catch (loadError: unknown) {
        console.error(
          "[useMemberList] failed to load brands",
          loadError,
        );

        if (!cancelled) {
          setBrandMap({});
        }
      }
    }

    void loadBrands();

    return () => {
      cancelled = true;
    };
  }, []);

  // ─────────────────────────────────────────────
  // ページ番号変更
  // ─────────────────────────────────────────────

  const setPageNumber = useCallback(
    (pageNumber: number) => {
      const safePageNumber = Math.max(1, pageNumber);

      const nextPage: PageState = {
        ...page,
        number: safePageNumber,
      };

      setPage(nextPage);
      void load(nextPage, filter, false);
    },
    [page, filter, load],
  );

  // ─────────────────────────────────────────────
  // 権限カテゴリ抽出
  // ─────────────────────────────────────────────

  const extractPermissionCategories = (
    permissions?: string[],
  ): string[] => {
    if (!permissions || permissions.length === 0) {
      return [];
    }

    const categories = new Set<string>();

    for (const permission of permissions) {
      const separatorIndex = permission.indexOf(".");
      const category =
        separatorIndex > 0
          ? permission.slice(0, separatorIndex)
          : permission;

      if (category) {
        categories.add(category);
      }
    }

    return Array.from(categories);
  };

  // ─────────────────────────────────────────────
  // フィルタ候補
  // ─────────────────────────────────────────────

  const brandFilterOptions: FilterOption[] = useMemo(
    () =>
      Object.entries(brandMap).map(([id, name]) => ({
        value: id,
        label: name || id,
      })),
    [brandMap],
  );

  const permissionFilterOptions: FilterOption[] = useMemo(() => {
    const categories = new Set<string>();

    for (const member of members) {
      const memberCategories = extractPermissionCategories(
        member.permissions,
      );

      for (const category of memberCategories) {
        categories.add(category);
      }
    }

    return Array.from(categories).map((category) => ({
      value: category,
      label: category,
    }));
  }, [members]);

  // ─────────────────────────────────────────────
  // 日付 → YYYY/MM/DD
  // Backend BFFのMember型ではISO 8601文字列を正とする。
  // ─────────────────────────────────────────────

  const formatYmd = (date: string | null): string => {
    if (!date) {
      return "";
    }

    return date.slice(0, 10).replace(/-/g, "/");
  };

  // ─────────────────────────────────────────────
  // ソート
  // ─────────────────────────────────────────────

  const getDateValue = useCallback(
    (member: Member): number => {
      const raw =
        sortKey === "updatedAt"
          ? member.updatedAt
          : member.createdAt;

      if (!raw) {
        return 0;
      }

      const timestamp = new Date(raw).getTime();
      return Number.isNaN(timestamp) ? 0 : timestamp;
    },
    [sortKey],
  );

  // ─────────────────────────────────────────────
  // フィルタ適用
  // ─────────────────────────────────────────────

  const filteredMembers = useMemo(() => {
    return members.filter((member) => {
      const assignedBrands = member.assignedBrands ?? [];
      const permissionCategories = extractPermissionCategories(
        member.permissions,
      );

      const matchesBrandFilter =
        selectedBrandIds.length === 0 ||
        assignedBrands.some((brandId) =>
          selectedBrandIds.includes(brandId),
        );

      const matchesPermissionFilter =
        selectedPermissionCats.length === 0 ||
        permissionCategories.some((category) =>
          selectedPermissionCats.includes(category),
        );

      return matchesBrandFilter && matchesPermissionFilter;
    });
  }, [
    members,
    selectedBrandIds,
    selectedPermissionCats,
  ]);

  const sortedMembers = useMemo(() => {
    if (!sortKey) {
      return filteredMembers;
    }

    return [...filteredMembers].sort(
      (firstMember, secondMember) => {
        const firstValue = getDateValue(firstMember);
        const secondValue = getDateValue(secondMember);

        return sortDirection === "asc"
          ? firstValue - secondValue
          : secondValue - firstValue;
      },
    );
  }, [
    filteredMembers,
    sortKey,
    sortDirection,
    getDateValue,
  ]);

  const handleSortChange = useCallback(
    (
      key: string,
      nextDirection: SortDirection | null,
    ) => {
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
  // リセット
  // ─────────────────────────────────────────────

  const handleReset = useCallback(() => {
    setSelectedBrandIds([]);
    setSelectedPermissionCats([]);
    setSortKey(null);
    setSortDirection("desc");

    const nextPage: PageState = {
      ...page,
      number: 1,
    };

    setPage(nextPage);
    void load(nextPage, filter, true);
  }, [page, filter, load]);

  return {
    members: sortedMembers,
    loading,
    error,
    isResetting,

    page,
    setPage,
    setPageNumber,

    sortKey,
    sortDirection,
    handleSortChange,

    handleReset,

    brandMap,
    brandFilterOptions,
    permissionFilterOptions,

    selectedBrandIds,
    setSelectedBrandIds,

    selectedPermissionCats,
    setSelectedPermissionCats,

    extractPermissionCategories,
    formatYmd,
  };
}