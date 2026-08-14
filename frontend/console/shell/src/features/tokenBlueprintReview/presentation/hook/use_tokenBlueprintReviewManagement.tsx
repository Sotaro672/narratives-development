// frontend/console/shell/src/features/tokenBlueprintReview/presentation/hook/use_tokenBlueprintReviewManagement.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import {
  type SortKey,
  type SortDir,
  fetchTokenBlueprintReviews,
  filterAndSortTokenBlueprintReviews,
} from "../../application/tokenBlueprintReviewManagementService";

import type { TokenBlueprintReviewAggregate } from "../../../../shared/types/tokenBlueprintReview";

export type UseTokenBlueprintReviewManagementResult = {
  rows: TokenBlueprintReviewAggregate[];
  brandOptions: {
    value: string;
    label: string;
  }[];
  brandFilter: string[];
  handleChangeBrandFilter: (values: string[]) => void;
  sortKey: SortKey;
  sortDir: SortDir;
  isResetting: boolean;
  handleChangeSort: (key: string | null, direction: SortDir) => void;
  handleReset: () => void;
  handleRowClick: (tokenBlueprintId: string) => void;
};

/**
 * TokenBlueprintReview Managementページ用ロジック。
 *
 * - backend は companyId を認証コンテキストから解決する
 * - brandName フィルタ、ソート、行クリックなどの UI ロジックを管理する
 */
export function useTokenBlueprintReviewManagement(): UseTokenBlueprintReviewManagementResult {
  const navigate = useNavigate();

  const [rows, setRows] = useState<TokenBlueprintReviewAggregate[]>([]);
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [sortKey, setSortKey] = useState<SortKey>(null);
  const [sortDir, setSortDir] = useState<SortDir>(null);
  const [isResetting, setIsResetting] = useState(false);

  const reload = useCallback(async (): Promise<void> => {
    setIsResetting(true);

    try {
      const result = await fetchTokenBlueprintReviews();
      setRows(result);
    } catch {
      setRows([]);
    } finally {
      setIsResetting(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const brandOptions = useMemo(() => {
    const brandNames = new Set<string>();

    for (const row of rows) {
      if (row.brandName) {
        brandNames.add(row.brandName);
      }
    }

    return Array.from(brandNames)
      .sort((first, second) => first.localeCompare(second))
      .map((brandName) => ({
        value: brandName,
        label: brandName,
      }));
  }, [rows]);

  const brandFilteredRows = useMemo(() => {
    if (brandFilter.length === 0) {
      return rows;
    }

    return rows.filter((row) => brandFilter.includes(row.brandName));
  }, [rows, brandFilter]);

  const sortedRows = useMemo(() => {
    return filterAndSortTokenBlueprintReviews(brandFilteredRows, {
      sortKey,
      sortDir,
    });
  }, [brandFilteredRows, sortKey, sortDir]);

  const handleRowClick = useCallback(
    (tokenBlueprintId: string) => {
      if (!tokenBlueprintId) {
        return;
      }

      navigate(
        `/tokenBlueprintReview/${encodeURIComponent(tokenBlueprintId)}`,
      );
    },
    [navigate],
  );

  const handleReset = useCallback(() => {
    setBrandFilter([]);
    setSortKey(null);
    setSortDir(null);
    void reload();
  }, [reload]);

  const handleChangeBrandFilter = useCallback((values: string[]) => {
    setBrandFilter(values);
  }, []);

  const handleChangeSort = useCallback(
    (key: string | null, direction: SortDir) => {
      if (
        key === "createdAt" ||
        key === "updatedAt" ||
        key === null
      ) {
        setSortKey(key);
      } else {
        setSortKey(null);
      }

      setSortDir(direction);
    },
    [],
  );

  return {
    rows: sortedRows,
    brandOptions,
    brandFilter,
    handleChangeBrandFilter,
    sortKey,
    sortDir,
    isResetting,
    handleChangeSort,
    handleReset,
    handleRowClick,
  };
}