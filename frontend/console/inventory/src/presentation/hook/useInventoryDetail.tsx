// frontend/console/inventory/src/presentation/hook/useInventoryDetail.tsx

import * as React from "react";
import type { InventoryRow } from "../components/inventoryCard";
import {
  queryInventoryDetailByProductAndToken,
  type InventoryDetailViewModel,
} from "../../application/inventoryDetailService";

export type UseInventoryDetailResult = {
  vm: InventoryDetailViewModel | null;
  rows: InventoryRow[];
  loading: boolean;
  error: string | null;
};

export function useInventoryDetail(
  productBlueprintId: string | undefined,
  tokenBlueprintId: string | undefined,
): UseInventoryDetailResult {
  const [vm, setVm] = React.useState<InventoryDetailViewModel | null>(null);
  const [rows, setRows] = React.useState<InventoryRow[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  // 画面マウント確認ログ（遷移できてるか）
  React.useEffect(() => {
    console.log("[inventory/useInventoryDetail] mounted", {
      productBlueprintId,
      tokenBlueprintId,
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  React.useEffect(() => {
    const pbId = String(productBlueprintId ?? "").trim();
    const tbId = String(tokenBlueprintId ?? "").trim();

    if (!pbId || !tbId) return;

    let cancelled = false;

    (async () => {
      try {
        setLoading(true);
        setError(null);

        console.log("[inventory/useInventoryDetail] fetch start", { pbId, tbId });

        // ✅ 方針A: pbId + tbId -> inventoryIds -> details -> merge(viewModel)
        const merged = await queryInventoryDetailByProductAndToken(pbId, tbId);

        if (cancelled) return;

        console.log("[inventory/useInventoryDetail] fetch ok", {
          inventoryKey: merged.inventoryKey,
          inventoryIds: merged.inventoryIds?.length ?? 0,
          rowsCount: merged.rows?.length ?? 0,
          totalStock: merged.totalStock,
          updatedAt: merged.updatedAt,
          vm: merged,
        });

        setVm(merged);
        setRows(Array.isArray(merged.rows) ? merged.rows : []);
      } catch (e: any) {
        if (cancelled) return;

        console.error("[inventory/useInventoryDetail] fetch error", {
          productBlueprintId,
          tokenBlueprintId,
          error: String(e?.message ?? e),
          raw: e,
        });

        setError(String(e?.message ?? e));
        setVm(null);
        setRows([]);
      } finally {
        if (cancelled) return;
        setLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [productBlueprintId, tokenBlueprintId]);

  return { vm, rows, loading, error };
}
