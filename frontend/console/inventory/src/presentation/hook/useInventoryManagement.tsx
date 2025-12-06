// frontend/console/inventory/src/presentation/hook/useInventoryManagement.tsx

import {
  useMemo,
  useState,
  useCallback,
  useEffect,
} from "react";
import { useNavigate } from "react-router-dom";
import type { ProductBlueprint } from "../../domain/entity/productBlueprint";
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";
import { API_BASE as PRODUCT_BLUEPRINT_API_BASE } from "../../../../productBlueprint/src/infrastructure/repository/productBlueprintRepositoryHTTP";

export type InventorySortKey = "totalQuantity" | null;
export type SortDirection = "asc" | "desc" | null;

/**
 * InventoryRow は BE/API から取得する在庫データの行。
 * mock_data は廃止したので、外部 API から取得する前提。
 */
export type InventoryRow = {
  id: string;
  productBlueprintId: string;

  productName: string;
  brandName: string;

  assigneeName?: string; // ★ ProductBlueprint から補完
  totalQuantity: number;
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
  const [printedBlueprints, setPrintedBlueprints] = useState<ProductBlueprint[]>([]);

  // ===== フィルタ状態 =====
  const [productFilter, setProductFilter] = useState<string[]>([]);
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [assigneeFilter, setAssigneeFilter] = useState<string[]>([]);

  // ===== ソート状態 =====
  const [sortKey, setSortKey] = useState<InventorySortKey>(null);
  const [sortDir, setSortDir] = useState<SortDirection>(null);

  /* ---------------------------------------------------------
   * API①：在庫一覧の取得（本来は BE に合わせて API を実装）
   * --------------------------------------------------------- */
  useEffect(() => {
    (async () => {
      try {
        const user = auth.currentUser;
        if (!user) return;

        const token = await user.getIdToken();

        const res = await fetch(`/api/inventory`, {
          method: "GET",
          headers: {
            Authorization: `Bearer ${token}`,
          },
        });

        if (!res.ok) return;

        const rows = (await res.json()) as InventoryRow[];
        setInventoryRows(rows ?? []);
      } catch {
        // 無視
      }
    })();
  }, []);

  /* ---------------------------------------------------------
   * API②：printed ProductBlueprint 一覧の取得
   * backend/internal/application/usecase/productBlueprint_usecase.go の
   * ListIDsByCompany → ListPrinted を呼ぶ
   * --------------------------------------------------------- */
  useEffect(() => {
    (async () => {
      try {
        const user = auth.currentUser;
        if (!user) return;

        const token = await user.getIdToken();

        const res = await fetch(
          `${PRODUCT_BLUEPRINT_API_BASE}/product-blueprints/printed`,
          {
            method: "GET",
            headers: { Authorization: `Bearer ${token}` },
          },
        );

        if (!res.ok) return;

        const data = (await res.json()) as ProductBlueprint[];
        setPrintedBlueprints(data ?? []);
      } catch {
        // UI 保護のため無視
      }
    })();
  }, []);

  /* ---------------------------------------------------------
   * printedBlueprints と inventoryRows をマージ
   * --------------------------------------------------------- */
  const mergedRows = useMemo(() => {
    const index = new Map<string, ProductBlueprint>();
    printedBlueprints.forEach((pb) => index.set(pb.id, pb));

    return inventoryRows.map((row) => {
      const pb = index.get(row.productBlueprintId);

      return {
        ...row,
        assigneeName: pb?.createdBy ?? row.assigneeName ?? "-", // 適宜調整
      };
    });
  }, [inventoryRows, printedBlueprints]);

  /* ---------------------------------------------------------
   * フィルタ → ソートの処理
   * --------------------------------------------------------- */
  const filteredSortedRows = useMemo(() => {
    let data = mergedRows.filter((r) => {
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
    mergedRows,
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
      navigate(`/inventory/${encodeURIComponent(row.id)}`);
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
   * フィルタ項目（オプション）の作成
   * --------------------------------------------------------- */
  const productOptions = useMemo(
    () =>
      Array.from(
        new Set(mergedRows.map((r) => r.productName)),
      ).map((v) => ({ value: v, label: v })),
    [mergedRows],
  );

  const brandOptions = useMemo(
    () =>
      Array.from(
        new Set(mergedRows.map((r) => r.brandName)),
      ).map((v) => ({ value: v, label: v })),
    [mergedRows],
  );

  const assigneeOptions = useMemo(
    () =>
      Array.from(
        new Set(
          mergedRows
            .map((r) => r.assigneeName)
            .filter((x): x is string => !!x),
        ),
      ).map((v) => ({ value: v, label: v })),
    [mergedRows],
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
