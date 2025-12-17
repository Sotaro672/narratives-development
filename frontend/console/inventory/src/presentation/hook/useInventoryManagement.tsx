// frontend/console/inventory/src/presentation/hook/useInventoryManagement.tsx

import { useMemo, useState, useCallback, useEffect } from "react";
import { useNavigate } from "react-router-dom";

import {
  loadInventoryRowsFromBackend,
  buildInventoryFilterOptionsFromRows,
  type InventoryManagementRow,
  type InventorySortKey,
} from "../../application/inventoryManagementService";

export type SortDirection = "asc" | "desc" | null;

/**
 * ✅ InventoryRow は inventory_query.go の結果を元にした一覧用の行。
 * 列順: [プロダクト名, トークン名, 在庫数]
 */
export type InventoryRow = {
  id: string; // 一覧の主キー（UI用）
  productBlueprintId: string;

  productName: string;

  tokenBlueprintId: string; // ★追加: 集計キー
  tokenName: string;

  stock: number;
};

/** フックの返却型 */
export type UseInventoryManagementResult = {
  rows: InventoryRow[];
  options: {
    productOptions: Array<{ value: string; label: string }>;
    tokenOptions: Array<{ value: string; label: string }>;
  };
  state: {
    productFilter: string[];
    tokenFilter: string[];
    sortKey: InventorySortKey;
    sortDir: SortDirection;
  };
  handlers: {
    setProductFilter: (v: string[]) => void;
    setTokenFilter: (v: string[]) => void;
    setSortKey: (k: InventorySortKey) => void;
    setSortDir: (d: SortDirection) => void;
    handleRowClick: (row: InventoryRow) => void;
    handleReset: () => void;
  };
};

function mapToRows(items: InventoryManagementRow[]): InventoryRow[] {
  return items.map((x, i) => ({
    // ★ productBlueprintId + tokenBlueprintId で一意になる想定（念のため i も付与）
    id: `${x.productBlueprintId}__${x.tokenBlueprintId}__${i}`,
    productBlueprintId: x.productBlueprintId,

    productName: x.productName,

    tokenBlueprintId: x.tokenBlueprintId,
    tokenName: x.tokenName,

    stock: x.stock,
  }));
}

/** 在庫管理ページ用 ロジックフック */
export function useInventoryManagement(): UseInventoryManagementResult {
  const navigate = useNavigate();

  // ===== rows (raw) =====
  const [inventoryRows, setInventoryRows] = useState<InventoryRow[]>([]);

  // ===== filters =====
  const [productFilter, setProductFilter] = useState<string[]>([]);
  const [tokenFilter, setTokenFilter] = useState<string[]>([]);

  // ===== sort =====
  const [sortKey, setSortKey] = useState<InventorySortKey>("productName");
  const [sortDir, setSortDir] = useState<SortDirection>("asc");

  /* ---------------------------------------------------------
   * ✅ inventory_query.go の結果をロード
   * --------------------------------------------------------- */
  useEffect(() => {
    (async () => {
      try {
        console.log("[inventory/useInventoryManagement] load start");

        const vmRows = await loadInventoryRowsFromBackend();
        const mapped = mapToRows(vmRows);

        console.log("[inventory/useInventoryManagement] load ok", {
          rows: mapped.length,
          sample: mapped.slice(0, 10),
        });

        setInventoryRows(mapped);
      } catch (e: any) {
        console.warn("[inventory/useInventoryManagement] load failed", {
          error: String(e?.message ?? e),
        });
        setInventoryRows([]);
      }
    })();
  }, []);

  /* ---------------------------------------------------------
   * フィルタ → ソート
   * --------------------------------------------------------- */
  const filteredSortedRows = useMemo(() => {
    let data = inventoryRows.filter((r) => {
      const productOk =
        productFilter.length === 0 || productFilter.includes(r.productName);

      const tokenOk = tokenFilter.length === 0 || tokenFilter.includes(r.tokenName);

      return productOk && tokenOk;
    });

    if (sortKey && sortDir) {
      data = [...data].sort((a, b) => {
        const dir = sortDir === "asc" ? 1 : -1;

        const as = (v: any) => String(v ?? "");
        const an = (v: any) => Number(v ?? 0);

        if (sortKey === "productName")
          return dir * as(a.productName).localeCompare(as(b.productName));
        if (sortKey === "tokenName")
          return dir * as(a.tokenName).localeCompare(as(b.tokenName));
        if (sortKey === "stock") return dir * (an(a.stock) - an(b.stock));

        return 0;
      });
    }

    return data;
  }, [inventoryRows, productFilter, tokenFilter, sortKey, sortDir]);

  /* ---------------------------------------------------------
   * options（フィルタ選択肢）
   * ※ product/token は Service helper を利用
   * --------------------------------------------------------- */
  const options = useMemo(() => {
    const asServiceRows: InventoryManagementRow[] = filteredSortedRows.map((r) => ({
      productBlueprintId: r.productBlueprintId,
      productName: r.productName,
      tokenBlueprintId: r.tokenBlueprintId,
      tokenName: r.tokenName,
      stock: r.stock,
    }));

    const base = buildInventoryFilterOptionsFromRows(asServiceRows);

    return {
      productOptions: base.productOptions,
      tokenOptions: base.tokenOptions,
    };
  }, [filteredSortedRows]);

  /* ---------------------------------------------------------
   * UI handlers
   * --------------------------------------------------------- */
  const handleRowClick = useCallback(
    (row: InventoryRow) => {
      // ✅ 詳細は pbId を渡して query させる（期待値）
      navigate(`/inventory/detail/${encodeURIComponent(row.productBlueprintId)}`);
    },
    [navigate],
  );

  const handleReset = useCallback(() => {
    setProductFilter([]);
    setTokenFilter([]);
    setSortKey("productName");
    setSortDir("asc");
  }, []);

  return {
    rows: filteredSortedRows,
    options: {
      productOptions: options.productOptions,
      tokenOptions: options.tokenOptions,
    },
    state: {
      productFilter,
      tokenFilter,
      sortKey,
      sortDir,
    },
    handlers: {
      setProductFilter,
      setTokenFilter,
      setSortKey,
      setSortDir,
      handleRowClick,
      handleReset,
    },
  };
}
