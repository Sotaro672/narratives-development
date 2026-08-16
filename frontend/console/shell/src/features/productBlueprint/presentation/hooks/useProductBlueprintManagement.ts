// frontend/console/shell/src/features/productBlueprint/presentation/hooks/useProductBlueprintManagement.ts

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import {
  fetchProductBlueprintManagementRows,
  filterAndSortProductBlueprintRows,
  type ProductBlueprintSortKey,
  type SortDirection,
} from "../../application/productBlueprintManagementService";
import type { ProductBlueprintListRow } from "../../infrastructure/repository/productBlueprintRepositoryHTTP";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";

export interface UseProductBlueprintManagementResult {
  rows: ProductBlueprintListRow[];
  brandFilter: string[];
  assigneeFilter: string[];
  printedFilter: string[];
  handleBrandFilterChange: (values: string[]) => void;
  handleAssigneeFilterChange: (values: string[]) => void;
  handlePrintedFilterChange: (values: string[]) => void;
  handleSortChange: (key: string | null, dir: "asc" | "desc" | null) => void;
  handleRowClick: (row: ProductBlueprintListRow) => void;
  handleCreate: () => void;
  handleReset: () => void;
  isResetting: boolean;
}

/**
 * 商品設計一覧画面のロジック。
 * - BackendのGET /product-blueprintsを正とする
 * - フィルタ・ソート・表示用日時整形・画面遷移のみ担当する
 */
export function useProductBlueprintManagement(): UseProductBlueprintManagementResult {
  const navigate = useNavigate();

  const [allRows, setAllRows] = useState<ProductBlueprintListRow[]>([]);
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [assigneeFilter, setAssigneeFilter] = useState<string[]>([]);
  const [printedFilter, setPrintedFilter] = useState<string[]>([]);
  const [sortedKey, setSortedKey] = useState<ProductBlueprintSortKey>(null);
  const [sortedDir, setSortedDir] = useState<SortDirection>(null);
  const [isResetting, setIsResetting] = useState(false);

  const load = useCallback(async (): Promise<void> => {
    setIsResetting(true);

    try {
      const rows = await fetchProductBlueprintManagementRows();
      setAllRows(rows);
    } catch {
      setAllRows([]);
    } finally {
      setIsResetting(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const filteredSortedRows = useMemo<ProductBlueprintListRow[]>(
    () =>
      filterAndSortProductBlueprintRows({
        allRows,
        brandFilter,
        assigneeFilter,
        printedFilter,
        sortedKey,
        sortedDir,
      }),
    [allRows, brandFilter, assigneeFilter, printedFilter, sortedKey, sortedDir],
  );

  const rows = useMemo<ProductBlueprintListRow[]>(
    () =>
      filteredSortedRows.map((row) => ({
        ...row,
        createdAt: safeDateTimeLabelJa(row.createdAt, ""),
        updatedAt: safeDateTimeLabelJa(row.updatedAt, ""),
      })),
    [filteredSortedRows],
  );

  const handleBrandFilterChange = useCallback((values: string[]) => {
    setBrandFilter(values);
  }, []);

  const handleAssigneeFilterChange = useCallback((values: string[]) => {
    setAssigneeFilter(values);
  }, []);

  const handlePrintedFilterChange = useCallback((values: string[]) => {
    setPrintedFilter(values);
  }, []);

  const handleSortChange = useCallback(
    (key: string | null, dir: "asc" | "desc" | null) => {
      const nextKey: ProductBlueprintSortKey =
        key === "createdAt" || key === "updatedAt" ? key : null;

      setSortedKey(nextKey);
      setSortedDir(dir);
    },
    [],
  );

  const handleRowClick = useCallback(
    (row: ProductBlueprintListRow) => {
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
    setPrintedFilter([]);
    setSortedKey(null);
    setSortedDir(null);
    void load();
  }, [load]);

  return {
    rows,
    brandFilter,
    assigneeFilter,
    printedFilter,
    handleBrandFilterChange,
    handleAssigneeFilterChange,
    handlePrintedFilterChange,
    handleSortChange,
    handleRowClick,
    handleCreate,
    handleReset,
    isResetting,
  };
}