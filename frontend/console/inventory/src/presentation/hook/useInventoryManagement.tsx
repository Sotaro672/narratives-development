// frontend/console/inventory/src/presentation/hook/useInventoryManagement.tsx

import {
  useMemo,
  useState,
  useCallback,
  useEffect,
} from "react";
import { useNavigate } from "react-router-dom";
import {
  fetchPrintedInventorySummaries,
  type InventoryProductSummary,
} from "../../infrastructure/http/inventoryRepositoryHTTP";

export type InventorySortKey = "totalQuantity" | null;
export type SortDirection = "asc" | "desc" | null;

/**
 * InventoryRow は BE/API から取得する在庫データの行。
 * 現状は printed な ProductBlueprint 一覧を 1 行ずつマッピングする。
 */
export type InventoryRow = {
  id: string;
  productBlueprintId: string;

  productName: string;
  brandName: string;     // brandId → brandName

  assigneeName?: string; // assigneeId → Member 名
  totalQuantity: number; // 将来拡張
};

/** フックの返却型 */
export type UseInventoryManagementResult = {
  rows: InventoryRow[];
  options: {
    productOptions: Array<{ value: string; label: string }>;
    brandOptions: Array<{ value: string; label: string }>;
    assigneeOptions: Array<{ value: string; label: string }>;
  };
  state: {
    productFilter: string[];
    brandFilter: string[];
    assigneeFilter: string[];
    sortKey: InventorySortKey;
    sortDir: SortDirection;
  };
  handlers: {
    setProductFilter: (v: string[]) => void;
    setBrandFilter: (v: string[]) => void;
    setAssigneeFilter: (v: string[]) => void;
    setSortKey: (k: InventorySortKey) => void;
    setSortDir: (d: SortDirection) => void;
    handleRowClick: (row: InventoryRow) => void;
    handleReset: () => void;
  };
};

/** 在庫管理ページ用 ロジックフック */
export function useInventoryManagement(): UseInventoryManagementResult {
  const navigate = useNavigate();

  // ===== クライアント側状態 =====
  const [inventoryRows, setInventoryRows] = useState<InventoryRow[]>([]);

  // ===== フィルタ状態 =====
  const [productFilter, setProductFilter] = useState<string[]>([]);
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [assigneeFilter, setAssigneeFilter] = useState<string[]>([]);

  // ===== ソート状態 =====
  const [sortKey, setSortKey] = useState<InventorySortKey>(null);
  const [sortDir, setSortDir] = useState<SortDirection>(null);

  /* ---------------------------------------------------------
   * printed ProductBlueprint 一覧の取得
   * --------------------------------------------------------- */
  useEffect(() => {
    (async () => {
      try {
        const summaries: InventoryProductSummary[] =
          await fetchPrintedInventorySummaries();

        const rows: InventoryRow[] = summaries.map((s) => ({
          id: s.id,
          productBlueprintId: s.id,
          productName: s.productName,
          brandName:
            s.brandName && s.brandName.trim().length > 0
              ? s.brandName
              : s.brandId,
          assigneeName:
            (s.assigneeName && s.assigneeName.trim().length > 0
              ? s.assigneeName
              : s.assigneeId || "-") || "-",
          totalQuantity: 0,
        }));

        setInventoryRows(rows);
      } catch (e) {
        setInventoryRows([]);
      }
    })();
  }, []);

  /* ---------------------------------------------------------
   * フィルタ → ソートの処理
   * --------------------------------------------------------- */
  const filteredSortedRows = useMemo(() => {
    let data = inventoryRows.filter((r) => {
      const productOk =
        productFilter.length === 0 ||
        productFilter.includes(r.productName);

      const brandOk =
        brandFilter.length === 0 ||
        brandFilter.includes(r.brandName);

      const assigneeOk =
        assigneeFilter.length === 0 ||
        (r.assigneeName != null &&
          assigneeFilter.includes(r.assigneeName));

      return productOk && brandOk && assigneeOk;
    });

    if (sortKey && sortDir) {
      data = [...data].sort((a, b) => {
        const av = a.totalQuantity;
        const bv = b.totalQuantity;
        return sortDir === "asc" ? av - bv : bv - av;
      });
    }

    return data;
  }, [
    inventoryRows,
    productFilter,
    brandFilter,
    assigneeFilter,
    sortKey,
    sortDir,
  ]);

  /* ---------------------------------------------------------
   * UI ハンドラ
   * --------------------------------------------------------- */
  const handleRowClick = useCallback(
    (row: InventoryRow) => {
      navigate(`/inventory/detail/${encodeURIComponent(row.id)}`);
    },
    [navigate],
  );

  const handleReset = useCallback(() => {
    setProductFilter([]);
    setBrandFilter([]);
    setAssigneeFilter([]);
    setSortKey(null);
    setSortDir(null);
  }, []);

  /* ---------------------------------------------------------
   * フィルタ項目（オプション）
   * --------------------------------------------------------- */
  const productOptions = useMemo(
    () =>
      Array.from(
        new Set(filteredSortedRows.map((r) => r.productName)),
      ).map((v) => ({ value: v, label: v })),
    [filteredSortedRows],
  );

  const brandOptions = useMemo(
    () =>
      Array.from(
        new Set(filteredSortedRows.map((r) => r.brandName)),
      ).map((v) => ({ value: v, label: v })),
    [filteredSortedRows],
  );

  const assigneeOptions = useMemo(
    () =>
      Array.from(
        new Set(
          filteredSortedRows
            .map((r) => r.assigneeName)
            .filter((x): x is string => !!x),
        ),
      ).map((v) => ({ value: v, label: v })),
    [filteredSortedRows],
  );

  return {
    rows: filteredSortedRows,
    options: {
      productOptions,
      brandOptions,
      assigneeOptions,
    },
    state: {
      productFilter,
      brandFilter,
      assigneeFilter,
      sortKey,
      sortDir,
    },
    handlers: {
      setProductFilter,
      setBrandFilter,
      setAssigneeFilter,
      setSortKey,
      setSortDir,
      handleRowClick,
      handleReset,
    },
  };
}

