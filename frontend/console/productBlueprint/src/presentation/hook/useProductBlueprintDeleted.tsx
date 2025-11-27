// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintDeleted.tsx

import { useMemo, useState, useCallback, useEffect } from "react";
import { useNavigate } from "react-router-dom";

import {
  fetchProductBlueprintDeletedRows,
  filterAndSortProductBlueprintDeletedRows,
  type DeletedUiRow as UiRow,
  type ProductBlueprintDeletedSortKey,
  type SortDirection,
} from "../../application/productBlueprintDeletedService";

export interface UseProductBlueprintDeletedResult {
  rows: UiRow[];

  // フィルタ状態
  brandFilter: string[];
  assigneeFilter: string[];

  // フィルタ変更ハンドラ
  handleBrandFilterChange: (values: string[]) => void;
  handleAssigneeFilterChange: (values: string[]) => void;

  // ソート変更ハンドラ
  handleSortChange: (key: string | null, dir: "asc" | "desc" | null) => void;

  // 行クリック & 画面操作
  handleRowClick: (row: UiRow) => void;
  handleReset: () => void;

  // ★ キャンセルボタン押下
  handleCancel: () => void;
}

/**
 * 論理削除済み商品設計一覧画面のロジック
 */
export function useProductBlueprintDeleted(): UseProductBlueprintDeletedResult {
  const navigate = useNavigate();

  // 一覧データ
  const [allRows, setAllRows] = useState<UiRow[]>([]);

  // フィルタ & ソート状態
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [assigneeFilter, setAssigneeFilter] = useState<string[]>([]);
  const [sortedKey, setSortedKey] =
    useState<ProductBlueprintDeletedSortKey>(null);
  const [sortedDir, setSortedDir] = useState<SortDirection>(null);

  // ---------------------------
  // 初回ロード
  // ---------------------------
  useEffect(() => {
    (async () => {
      try {
        const uiRows = await fetchProductBlueprintDeletedRows();
        console.log(
          "[useProductBlueprintDeleted] fetched deleted uiRows:",
          uiRows
        );
        setAllRows(uiRows);
      } catch (err) {
        console.error("[useProductBlueprintDeleted] list load failed", err);
        setAllRows([]);
      }
    })();
  }, []);

  // ---------------------------
  // フィルタ・ソート適用
  // ---------------------------
  const rows: UiRow[] = useMemo(
    () =>
      filterAndSortProductBlueprintDeletedRows({
        allRows,
        brandFilter,
        assigneeFilter,
        sortedKey,
        sortedDir,
      }),
    [allRows, brandFilter, assigneeFilter, sortedKey, sortedDir]
  );

  // ---------------------------
  // ハンドラ群
  // ---------------------------
  const handleBrandFilterChange = useCallback((values: string[]) => {
    setBrandFilter(values);
  }, []);

  const handleAssigneeFilterChange = useCallback((values: string[]) => {
    setAssigneeFilter(values);
  }, []);

  const handleSortChange = useCallback(
    (key: string | null, dir: "asc" | "desc" | null) => {
      setSortedKey((key as ProductBlueprintDeletedSortKey) ?? null);
      setSortedDir(dir as SortDirection);
    },
    []
  );

  const handleRowClick = useCallback(
    (row: UiRow) => {
      navigate(`/productBlueprint/detail/${encodeURIComponent(row.id)}`);
    },
    [navigate]
  );

  const handleReset = useCallback(() => {
    setBrandFilter([]);
    setAssigneeFilter([]);
    setSortedKey(null);
    setSortedDir(null);
  }, []);

  // ---------------------------
  // ★ キャンセル → 通常一覧へ戻る
  // ---------------------------
  const handleCancel = useCallback(() => {
    navigate("/productBlueprint");
  }, [navigate]);

  return {
    rows,
    brandFilter,
    assigneeFilter,
    handleBrandFilterChange,
    handleAssigneeFilterChange,
    handleSortChange,
    handleRowClick,
    handleReset,
    handleCancel, // ★ 追加
  };
}
