// frontend/console/tokenBlueprintReview/src/presentation/hook/use_tokenBlueprintReviewManagement.tsx
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";
import {
  type SortKey,
  type SortDir,
  fetchTokenBlueprintReviewsForCompany,
  filterAndSortTokenBlueprintReviews,
} from "../../application/tokenBlueprintReviewManagementService";
import type { TokenBlueprintReviewAggregate } from "../../domain/entity";

export type UseTokenBlueprintReviewManagementResult = {
  rows: TokenBlueprintReviewAggregate[];

  brandOptions: { value: string; label: string }[];
  brandFilter: string[];
  handleChangeBrandFilter: (vals: string[]) => void;

  sortKey: SortKey;
  sortDir: SortDir;

  isResetting: boolean;

  handleChangeSort: (key: string | null, dir: SortDir) => void;
  handleReset: () => void;

  handleRowClick: (tokenBlueprintId: string) => void;
};

/**
 * TokenBlueprintReview Management ページ用ロジック（Hook）
 * - backend は companyId を auth context から解決する想定のため、フロントでは companyId ガードしない
 * - brandName フィルタ / ソート / 行クリック など UI 以外の要素を集約
 */
export function useTokenBlueprintReviewManagement(): UseTokenBlueprintReviewManagementResult {
  const navigate = useNavigate();
  const { currentMember } = useAuth();

  const [rows, setRows] = useState<TokenBlueprintReviewAggregate[]>([]);
  const [brandFilter, setBrandFilter] = useState<string[]>([]);

  const [sortKey, setSortKey] = useState<SortKey>(null);
  const [sortDir, setSortDir] = useState<SortDir>(null);

  const [isResetting, setIsResetting] = useState(false);

  // ─────────────────────────────
  // データ取得: 集計一覧を取得（service に委譲）
  // ─────────────────────────────
  const reload = useCallback(async () => {
    const companyId = String(currentMember?.companyId ?? "");

    setIsResetting(true);
    try {
      const result = await fetchTokenBlueprintReviewsForCompany(companyId);
      setRows(result ?? []);
    } catch {
      setRows([]);
    } finally {
      setIsResetting(false);
    }
  }, [currentMember?.companyId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  // brandName options（ユニーク）
  const brandOptions = useMemo(() => {
    const set = new Set<string>();

    for (const r of rows) {
      const name = String(r.brandName ?? "");
      if (name) set.add(name);
    }

    return Array.from(set)
      .sort((a, b) => a.localeCompare(b))
      .map((v) => ({ value: v, label: v }));
  }, [rows]);

  // brandName フィルタ適用
  const brandFilteredRows = useMemo(() => {
    if (!brandFilter || brandFilter.length === 0) return rows;

    return rows.filter((r) => {
      const brandName = String(r.brandName ?? "");
      return brandName !== "" && brandFilter.includes(brandName);
    });
  }, [rows, brandFilter]);

  // ソート適用後の行（SortKey は camelCase: createdAt / updatedAt）
  const sortedRows = useMemo(() => {
    return filterAndSortTokenBlueprintReviews(brandFilteredRows, {
      sortKey,
      sortDir,
    });
  }, [brandFilteredRows, sortKey, sortDir]);

  const handleRowClick = useCallback(
    (tokenBlueprintId: string) => {
      navigate(`/tokenBlueprintReview/${encodeURIComponent(tokenBlueprintId)}`);
    },
    [navigate],
  );

  const handleReset = useCallback(() => {
    setBrandFilter([]);
    setSortKey(null);
    setSortDir(null);
    void reload();
  }, [reload]);

  const handleChangeBrandFilter = useCallback((vals: string[]) => {
    setBrandFilter(vals ?? []);
  }, []);

  const handleChangeSort = useCallback((key: string | null, dir: SortDir) => {
    if (key === "createdAt" || key === "updatedAt" || key === null) {
      setSortKey(key);
    } else {
      setSortKey(null);
    }
    setSortDir(dir);
  }, []);

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