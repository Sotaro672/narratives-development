// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintManagement.ts

import { useMemo, useState, useCallback, useEffect } from "react";
import { useNavigate } from "react-router-dom";

import {
  fetchProductBlueprintManagementRows,
  filterAndSortProductBlueprintRows,
  type UiRow,
  type ProductBlueprintSortKey,
  type SortDirection,
} from "../../application/productBlueprintManagementService";

export interface UseProductBlueprintManagementResult {
  rows: UiRow[];

  // フィルタ状態
  brandFilter: string[];
  assigneeFilter: string[];
  tagFilter: string[];

  // フィルタ変更ハンドラ
  handleBrandFilterChange: (values: string[]) => void;
  handleAssigneeFilterChange: (values: string[]) => void;
  handleTagFilterChange: (values: string[]) => void;

  // ソート変更ハンドラ
  handleSortChange: (key: string | null, dir: "asc" | "desc" | null) => void;

  // 行クリック & 画面操作
  handleRowClick: (row: UiRow) => void;
  handleCreate: () => void;
  handleReset: () => void;
}

/**
 * 商品設計一覧画面のロジック
 * - backend の /product-blueprints を参照
 * - フィルタ・ソート・画面遷移のみ担当
 */
export function useProductBlueprintManagement(): UseProductBlueprintManagementResult {
  const navigate = useNavigate();

  // 一覧データ
  const [allRows, setAllRows] = useState<UiRow[]>([]);

  // フィルタ & ソート状態
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [assigneeFilter, setAssigneeFilter] = useState<string[]>([]);
  const [tagFilter, setTagFilter] = useState<string[]>([]);
  const [sortedKey, setSortedKey] = useState<ProductBlueprintSortKey>(null);
  const [sortedDir, setSortedDir] = useState<SortDirection>(null);

  // ---------------------------
  // 初回ロード: backend から取得
  // ---------------------------
  useEffect(() => {
    (async () => {
      try {
        const uiRows = await fetchProductBlueprintManagementRows();
        console.log(
          "[useProductBlueprintManagement] fetched uiRows:",
          uiRows,
        );
        setAllRows(uiRows);
      } catch (err) {
        console.error(
          "[useProductBlueprintManagement] list load failed",
          err,
        );
        setAllRows([]);
      }
    })();
  }, []);

  // ---------------------------
  // フィルタ・ソート適用
  // ---------------------------
  const rows: UiRow[] = useMemo(
    () =>
      filterAndSortProductBlueprintRows({
        allRows,
        brandFilter,
        assigneeFilter,
        tagFilter,
        sortedKey,
        sortedDir,
      }),
    [allRows, brandFilter, assigneeFilter, tagFilter, sortedKey, sortedDir],
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

  const handleTagFilterChange = useCallback((values: string[]) => {
    setTagFilter(values);
  }, []);

  const handleSortChange = useCallback(
    (key: string | null, dir: "asc" | "desc" | null) => {
      setSortedKey((key as ProductBlueprintSortKey) ?? null);
      setSortedDir(dir as SortDirection);
    },
    [],
  );

  const handleRowClick = useCallback(
    (row: UiRow) => {
      navigate(`/productBlueprint/detail/${encodeURIComponent(row.id)}`);
    },
    [navigate],
  );

  const handleCreate = useCallback(() => {
    navigate("/productBlueprint/create");
  }, [navigate]);

  const handleReset = useCallback(() => {
    setBrandFilter([]);
    setAssigneeFilter([]);
    setTagFilter([]);
    setSortedKey(null);
    setSortedDir(null);
  }, []);

  return {
    rows,
    brandFilter,
    assigneeFilter,
    tagFilter,
    handleBrandFilterChange,
    handleAssigneeFilterChange,
    handleTagFilterChange,
    handleSortChange,
    handleRowClick,
    handleCreate,
    handleReset,
  };
}
