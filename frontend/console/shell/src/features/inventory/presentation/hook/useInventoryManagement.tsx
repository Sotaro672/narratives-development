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
 * InventoryRow は inventory_query.go の結果を元にした一覧用の行。
 * 列順: [プロダクト名, トークン名, 保管場所, 在庫数, 注文数]
 */
export type InventoryRow = {
  id: string;
  productBlueprintId: string;
  productName: string;
  tokenBlueprintId: string;
  tokenName: string;
  shippingAddressName: string;
  availableStock: number;
  reservedCount: number;
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
  isResetting: boolean;
};

function mapToRows(items: InventoryManagementRow[]): InventoryRow[] {
  return items.map((x, i) => ({
    id: `${x.productBlueprintId}__${x.tokenBlueprintId}__${i}`,
    productBlueprintId: x.productBlueprintId,
    productName: x.productName,
    tokenBlueprintId: x.tokenBlueprintId,
    tokenName: x.tokenName,
    shippingAddressName: x.shippingAddressName,
    availableStock: x.availableStock,
    reservedCount: x.reservedCount,
  }));
}

function normalizeId(v: unknown): string {
  return String(v ?? "");
}

/** 在庫管理ページ用 ロジックフック */
export function useInventoryManagement(): UseInventoryManagementResult {
  const navigate = useNavigate();

  const [inventoryRows, setInventoryRows] = useState<InventoryRow[]>([]);
  const [productFilter, setProductFilter] = useState<string[]>([]);
  const [tokenFilter, setTokenFilter] = useState<string[]>([]);
  const [sortKey, setSortKey] = useState<InventorySortKey>("productName");
  const [sortDir, setSortDir] = useState<SortDirection>("asc");
  const [isResetting, setIsResetting] = useState(false);

  const reload = useCallback(async () => {
    setIsResetting(true);

    try {
      const vmRows = await loadInventoryRowsFromBackend();
      const mapped = mapToRows(vmRows);
      setInventoryRows(mapped);
    } catch (_e: any) {
      setInventoryRows([]);
    } finally {
      setIsResetting(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const filteredSortedRows = useMemo(() => {
    let data = inventoryRows.filter((r) => {
      const productOk = productFilter.length === 0 || productFilter.includes(r.productName);
      const tokenOk = tokenFilter.length === 0 || tokenFilter.includes(r.tokenName);
      return productOk && tokenOk;
    });

    if (sortKey && sortDir) {
      data = [...data].sort((a, b) => {
        const dir = sortDir === "asc" ? 1 : -1;
        const as = (v: any) => String(v ?? "");
        const an = (v: any) => Number(v ?? 0);

        if (sortKey === "productName") return dir * as(a.productName).localeCompare(as(b.productName));
        if (sortKey === "tokenName") return dir * as(a.tokenName).localeCompare(as(b.tokenName));
        if (sortKey === "availableStock") return dir * (an(a.availableStock) - an(b.availableStock));
        if (sortKey === "reservedCount") return dir * (an(a.reservedCount) - an(b.reservedCount));
        return 0;
      });
    }

    return data;
  }, [inventoryRows, productFilter, tokenFilter, sortKey, sortDir]);

  const options = useMemo(() => {
    const asServiceRows: InventoryManagementRow[] = filteredSortedRows.map((r) => ({
      productBlueprintId: r.productBlueprintId,
      productName: r.productName,
      tokenBlueprintId: r.tokenBlueprintId,
      tokenName: r.tokenName,
      shippingAddressName: r.shippingAddressName,
      availableStock: r.availableStock,
      reservedCount: r.reservedCount,
    }));

    const base = buildInventoryFilterOptionsFromRows(asServiceRows);

    return {
      productOptions: base.productOptions,
      tokenOptions: base.tokenOptions,
    };
  }, [filteredSortedRows]);

  const handleRowClick = useCallback(
    (row: InventoryRow) => {
      const pbId = normalizeId(row.productBlueprintId);
      const tbId = normalizeId(row.tokenBlueprintId);

      if (!pbId || !tbId || tbId === "-") return;

      const inventoryId = `${pbId}__${tbId}`;
      navigate(`/inventory/detail/${encodeURIComponent(inventoryId)}`);
    },
    [navigate],
  );

  const handleReset = useCallback(() => {
    setProductFilter([]);
    setTokenFilter([]);
    setSortKey("productName");
    setSortDir("asc");
    void reload();
  }, [reload]);

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
    isResetting,
  };
}