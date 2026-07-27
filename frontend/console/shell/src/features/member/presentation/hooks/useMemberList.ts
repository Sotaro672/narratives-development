// frontend/console/shell/src/features/member/presentation/hooks/useMemberList.ts

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import type { Member } from "../../../../shared/types/member";
import type { MemberFilter } from "../../domain/repository/memberRepository";
import type { Page } from "../../../../shared/types/common/common";
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

// ソート方向（SortableTableHeaderに合わせる）
export type SortDirection = "asc" | "desc";

export function useMemberList(
  initialFilter: MemberFilter = {},
  initialPage?: Page,
) {
  const [members, setMembers] =
    useState<Member[]>([]);

  const [filter] =
    useState<MemberFilter>(initialFilter);

  // PageにtotalPagesを含めた構造
  const [page, setPage] = useState<Page>({
    ...DEFAULT_PAGE,
    ...(initialPage ?? {}),
    totalPages: 1,
  });

  const [loading, setLoading] =
    useState(false);

  const [error, setError] =
    useState<Error | null>(null);

  // リフレッシュボタン回転用
  const [isResetting, setIsResetting] =
    useState(false);

  // ブランドID → 名称
  const [brandMap, setBrandMap] =
    useState<Record<string, string>>({});

  const [
    selectedBrandIds,
    setSelectedBrandIds,
  ] = useState<string[]>([]);

  const [
    selectedPermissionCats,
    setSelectedPermissionCats,
  ] = useState<string[]>([]);

  // ソート状態
  const [sortKey, setSortKey] =
    useState<string | null>(null);

  const [
    sortDirection,
    setSortDirection,
  ] = useState<SortDirection>("desc");

  // ─────────────────────────────────────────────
  // メンバー一覧ロード
  // ─────────────────────────────────────────────
  const load = useCallback(
    async (
      targetPage: Page,
      targetFilter: MemberFilter,
    ) => {
      setLoading(true);
      setIsResetting(true);
      setError(null);

      try {
        const result = await fetchMemberList(
          targetPage,
          targetFilter,
        );

        setMembers(result.members ?? []);

        setPage((previousPage) => ({
          ...previousPage,
          number: targetPage.number,
          perPage:
            targetPage.perPage ??
            previousPage.perPage ??
            DEFAULT_PAGE_LIMIT,
          totalPages:
            result.totalPages ??
            previousPage.totalPages ??
            1,
        }));
      } catch (loadError: unknown) {
        const normalizedError =
          loadError instanceof Error
            ? loadError
            : new Error(String(loadError));

        console.error(
          "[useMemberList] load error:",
          normalizedError,
        );

        setError(normalizedError);
      } finally {
        setLoading(false);
        setIsResetting(false);
      }
    },
    [],
  );

  // 初回ロード
  useEffect(() => {
    void load(page, filter);

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // ─────────────────────────────────────────────
  // ブランド一覧ロード
  // ─────────────────────────────────────────────
  useEffect(() => {
    let cancelled = false;

    async function loadBrands() {
      try {
        const brands =
          await fetchBrandsForCurrentMember();

        if (cancelled) {
          return;
        }

        const nextBrandMap: Record<
          string,
          string
        > = {};

        for (const brand of brands) {
          nextBrandMap[brand.id] =
            brand.name;
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
      const safePageNumber = Math.max(
        1,
        pageNumber,
      );

      const nextPage: Page = {
        ...page,
        number: safePageNumber,
      };

      setPage(nextPage);
      void load(nextPage, filter);
    },
    [page, filter, load],
  );

  // ─────────────────────────────────────────────
  // 権限カテゴリ抽出
  // ─────────────────────────────────────────────
  const extractPermissionCategories = (
    permissions?: string[],
  ): string[] => {
    if (
      !permissions ||
      permissions.length === 0
    ) {
      return [];
    }

    const categories = new Set<string>();

    for (const permission of permissions) {
      const separatorIndex =
        permission.indexOf(".");

      const category =
        separatorIndex > 0
          ? permission.slice(
              0,
              separatorIndex,
            )
          : permission;

      if (category) {
        categories.add(category);
      }
    }

    return Array.from(categories);
  };

  // ─────────────────────────────────────────────
  // フィルタ候補（ブランド）
  // ─────────────────────────────────────────────
  const brandFilterOptions: FilterOption[] =
    useMemo(
      () =>
        Object.entries(brandMap).map(
          ([id, name]) => ({
            value: id,
            label: name || id,
          }),
        ),
      [brandMap],
    );

  // ─────────────────────────────────────────────
  // フィルタ候補（権限カテゴリ）
  // ─────────────────────────────────────────────
  const permissionFilterOptions: FilterOption[] =
    useMemo(() => {
      const categories = new Set<string>();

      for (const member of members) {
        const memberCategories =
          extractPermissionCategories(
            member.permissions,
          );

        for (const category of memberCategories) {
          categories.add(category);
        }
      }

      return Array.from(categories).map(
        (category) => ({
          value: category,
          label: category,
        }),
      );
    }, [members]);

  // ─────────────────────────────────────────────
  // 日付 → YYYY/MM/DD
  // ─────────────────────────────────────────────
  const formatYmd = (
    date: unknown,
  ): string => {
    if (!date) {
      return "";
    }

    if (
      typeof date === "object" &&
      date !== null
    ) {
      const dateValue = date as {
        toDate?: () => Date;
        seconds?: number;
      };

      if (
        typeof dateValue.toDate ===
        "function"
      ) {
        return dateValue
          .toDate()
          .toISOString()
          .slice(0, 10)
          .replace(/-/g, "/");
      }

      if (
        typeof dateValue.seconds ===
        "number"
      ) {
        return new Date(
          dateValue.seconds * 1000,
        )
          .toISOString()
          .slice(0, 10)
          .replace(/-/g, "/");
      }
    }

    if (typeof date === "string") {
      return date
        .slice(0, 10)
        .replace(/-/g, "/");
    }

    return "";
  };

  // ─────────────────────────────────────────────
  // ソート用日付値
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

      const timestamp =
        new Date(raw).getTime();

      return Number.isNaN(timestamp)
        ? 0
        : timestamp;
    },
    [sortKey],
  );

  // ─────────────────────────────────────────────
  // フィルタ適用
  // ─────────────────────────────────────────────
  const filteredMembers = useMemo(() => {
    return members.filter((member) => {
      const assignedBrands =
        member.assignedBrands ?? [];

      const permissionCategories =
        extractPermissionCategories(
          member.permissions,
        );

      const matchesBrandFilter =
        selectedBrandIds.length === 0 ||
        assignedBrands.some((brandId) =>
          selectedBrandIds.includes(
            brandId,
          ),
        );

      const matchesPermissionFilter =
        selectedPermissionCats.length === 0 ||
        permissionCategories.some(
          (category) =>
            selectedPermissionCats.includes(
              category,
            ),
        );

      return (
        matchesBrandFilter &&
        matchesPermissionFilter
      );
    });
  }, [
    members,
    selectedBrandIds,
    selectedPermissionCats,
  ]);

  // ─────────────────────────────────────────────
  // ソート適用
  // ─────────────────────────────────────────────
  const sortedMembers = useMemo(() => {
    if (!sortKey) {
      return filteredMembers;
    }

    return [...filteredMembers].sort(
      (firstMember, secondMember) => {
        const firstValue =
          getDateValue(firstMember);

        const secondValue =
          getDateValue(secondMember);

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

  // ─────────────────────────────────────────────
  // ソート変更
  // ─────────────────────────────────────────────
  const handleSortChange = useCallback(
    (
      key: string,
      nextDirection:
        | SortDirection
        | null,
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

    const nextPage: Page = {
      ...page,
      number: 1,
    };

    setPage(nextPage);
    void load(nextPage, filter);
  }, [page, filter, load]);

  return {
    // 一覧
    members: sortedMembers,

    loading,
    error,

    // リフレッシュ回転用
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