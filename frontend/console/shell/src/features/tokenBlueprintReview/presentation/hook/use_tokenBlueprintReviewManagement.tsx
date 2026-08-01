// frontend/console/shell/src/features/tokenBlueprintReview/presentation/hook/use_tokenBlueprintReviewManagement.tsx

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useNavigate } from "react-router-dom";

import {
  useAuthContext,
} from "../../../../auth/application/AuthContext";

import {
  type SortKey,
  type SortDir,
  fetchTokenBlueprintReviewsForCompany,
  filterAndSortTokenBlueprintReviews,
} from "../../application/tokenBlueprintReviewManagementService";

import type {
  TokenBlueprintReviewAggregate,
} from "../../../../shared/types/tokenBlueprintReview";

export type UseTokenBlueprintReviewManagementResult = {
  rows: TokenBlueprintReviewAggregate[];

  brandOptions: {
    value: string;
    label: string;
  }[];

  brandFilter: string[];

  handleChangeBrandFilter: (
    values: string[],
  ) => void;

  sortKey: SortKey;
  sortDir: SortDir;

  isResetting: boolean;

  handleChangeSort: (
    key: string | null,
    direction: SortDir,
  ) => void;

  handleReset: () => void;

  handleRowClick: (
    tokenBlueprintId: string,
  ) => void;
};

/**
 * TokenBlueprintReview Managementページ用ロジック。
 *
 * - backendはcompanyIdを認証コンテキストから解決する想定
 * - brandNameフィルタ、ソート、行クリックなどの
 *   UI以外の処理を集約する
 */
export function useTokenBlueprintReviewManagement(): UseTokenBlueprintReviewManagementResult {
  const navigate = useNavigate();

  const {
    currentMember,
  } = useAuthContext();

  const [
    rows,
    setRows,
  ] = useState<
    TokenBlueprintReviewAggregate[]
  >([]);

  const [
    brandFilter,
    setBrandFilter,
  ] = useState<string[]>([]);

  const [
    sortKey,
    setSortKey,
  ] = useState<SortKey>(
    null,
  );

  const [
    sortDir,
    setSortDir,
  ] = useState<SortDir>(
    null,
  );

  const [
    isResetting,
    setIsResetting,
  ] = useState(false);

  /**
   * 集計一覧を取得する。
   */
  const reload = useCallback(
    async (): Promise<void> => {
      const companyId = String(
        currentMember?.companyId ?? "",
      );

      setIsResetting(true);

      try {
        const result =
          await fetchTokenBlueprintReviewsForCompany(
            companyId,
          );

        setRows(
          result ?? [],
        );
      } catch {
        setRows([]);
      } finally {
        setIsResetting(false);
      }
    },
    [
      currentMember?.companyId,
    ],
  );

  useEffect(() => {
    void reload();
  }, [reload]);

  /**
   * brandNameの選択肢を生成する。
   */
  const brandOptions = useMemo(() => {
    const brandNames =
      new Set<string>();

    for (const row of rows) {
      const brandName = String(
        row.brandName ?? "",
      );

      if (brandName) {
        brandNames.add(
          brandName,
        );
      }
    }

    return Array.from(
      brandNames,
    )
      .sort(
        (
          first,
          second,
        ) =>
          first.localeCompare(
            second,
          ),
      )
      .map((brandName) => ({
        value:
          brandName,
        label:
          brandName,
      }));
  }, [rows]);

  /**
   * brandNameフィルタを適用する。
   */
  const brandFilteredRows =
    useMemo(() => {
      if (
        brandFilter.length === 0
      ) {
        return rows;
      }

      return rows.filter(
        (row) => {
          const brandName =
            String(
              row.brandName ?? "",
            );

          return (
            brandName !== "" &&
            brandFilter.includes(
              brandName,
            )
          );
        },
      );
    }, [
      rows,
      brandFilter,
    ]);

  /**
   * ソートを適用する。
   *
   * SortKey:
   * - createdAt
   * - updatedAt
   */
  const sortedRows =
    useMemo(() => {
      return filterAndSortTokenBlueprintReviews(
        brandFilteredRows,
        {
          sortKey,
          sortDir,
        },
      );
    }, [
      brandFilteredRows,
      sortKey,
      sortDir,
    ]);

  const handleRowClick =
    useCallback(
      (
        tokenBlueprintId: string,
      ) => {
        const normalizedId =
          String(
            tokenBlueprintId ?? "",
          ).trim();

        if (!normalizedId) {
          return;
        }

        navigate(
          `/tokenBlueprintReview/${encodeURIComponent(
            normalizedId,
          )}`,
        );
      },
      [navigate],
    );

  const handleReset =
    useCallback(() => {
      setBrandFilter([]);
      setSortKey(null);
      setSortDir(null);

      void reload();
    }, [reload]);

  const handleChangeBrandFilter =
    useCallback(
      (
        values: string[],
      ) => {
        setBrandFilter(
          values ?? [],
        );
      },
      [],
    );

  const handleChangeSort =
    useCallback(
      (
        key: string | null,
        direction: SortDir,
      ) => {
        if (
          key === "createdAt" ||
          key === "updatedAt" ||
          key === null
        ) {
          setSortKey(
            key,
          );
        } else {
          setSortKey(
            null,
          );
        }

        setSortDir(
          direction,
        );
      },
      [],
    );

  return {
    rows:
      sortedRows,

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