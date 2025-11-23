// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintManagement.ts

import { useMemo, useState, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import {
  fetchProductBlueprintListRows,
  type ProductBlueprintListRow,
} from "../../infrastructure/api/productBlueprintApi";

// "YYYY/MM/DD" → timestamp（ソート用・UI 側の責務）
const toTs = (yyyyMd: string) => {
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

// 一覧表示用のUI行モデル（API からの型に別名を付けて利用）
export type UiRow = ProductBlueprintListRow;

type SortKey = "createdAt" | "lastModifiedAt" | null;

export interface UseProductBlueprintManagementResult {
  rows: UiRow[];
  brandFilter: string[];
  handleBrandFilterChange: (values: string[]) => void;
  handleSortChange: (
    key: string | null,
    dir: "asc" | "desc" | null
  ) => void;
  handleRowClick: (row: UiRow) => void;
  handleCreate: () => void;
  handleReset: () => void;
}

/**
 * 商品設計一覧画面用の状態管理・ロジックをまとめたフック
 * - API 呼び出し（fetchProductBlueprintListRows）は infrastructure/api に委譲
 * - ここではフィルタ・ソート・画面遷移のみを担当する
 */
export function useProductBlueprintManagement(): UseProductBlueprintManagementResult {
  const navigate = useNavigate();

  // フィルタ & ソート状態
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [sortedKey, setSortedKey] = useState<SortKey>(null);
  const [sortedDir, setSortedDir] = useState<"asc" | "desc" | null>(null);

  // ProductBlueprintListRow → UiRow へ変換＋フィルタ＋ソート
  const rows: UiRow[] = useMemo(() => {
    const all: UiRow[] = fetchProductBlueprintListRows();

    let work = all;

    if (brandFilter.length > 0) {
      work = work.filter((r) => brandFilter.includes(r.brandLabel));
    }

    if (sortedKey && sortedDir) {
      work = [...work].sort((a, b) => {
        const av = toTs(a[sortedKey]);
        const bv = toTs(b[sortedKey]);
        return sortedDir === "asc" ? av - bv : bv - av;
      });
    }

    return work;
  }, [brandFilter, sortedKey, sortedDir]);

  const handleBrandFilterChange = useCallback((values: string[]) => {
    setBrandFilter(values);
  }, []);

  const handleSortChange = useCallback(
    (key: string | null, dir: "asc" | "desc" | null) => {
      setSortedKey((key as SortKey) ?? null);
      setSortedDir(dir);
    },
    []
  );

  const handleRowClick = useCallback(
    (row: UiRow) => {
      navigate(`/productBlueprint/detail/${encodeURIComponent(row.id)}`);
    },
    [navigate]
  );

  const handleCreate = useCallback(() => {
    navigate("/productBlueprint/create");
  }, [navigate]);

  const handleReset = useCallback(() => {
    setBrandFilter([]);
    setSortedKey(null);
    setSortedDir(null);
  }, []);

  return {
    rows,
    brandFilter,
    handleBrandFilterChange,
    handleSortChange,
    handleRowClick,
    handleCreate,
    handleReset,
  };
}
