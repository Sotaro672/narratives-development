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

function asString(v: any): string {
  return String(v ?? "").trim();
}

export function useInventoryDetail(
  productBlueprintId: string | undefined,
  tokenBlueprintId: string | undefined,
): UseInventoryDetailResult {
  const [vm, setVm] = React.useState<InventoryDetailViewModel | null>(null);
  const [rows, setRows] = React.useState<InventoryRow[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  // ✅ サービス層（queryInventoryDetailByProductAndToken）を正として、入力正規化は hook 側で最小限のみ
  const pbId = React.useMemo(() => asString(productBlueprintId), [productBlueprintId]);
  const tbId = React.useMemo(() => asString(tokenBlueprintId), [tokenBlueprintId]);

  // 画面マウント確認ログ（遷移できてるか）
  React.useEffect(() => {
    console.log("[inventory/useInventoryDetail] mounted", {
      productBlueprintId,
      tokenBlueprintId,
      normalized: { pbId, tbId },
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  React.useEffect(() => {
    // 入力不足なら何もしない（サービス層が正なので、ここでの補完ロジックは持たない）
    if (!pbId || !tbId) {
      setVm(null);
      setRows([]);
      setLoading(false);
      setError(null);
      return;
    }

    let cancelled = false;

    (async () => {
      try {
        setLoading(true);
        setError(null);

        console.log("[inventory/useInventoryDetail] fetch start", { pbId, tbId });

        // ✅ 方針A: pbId + tbId -> inventoryIds -> details -> merge(viewModel)
        // ✅ 取得・集計・揺れ吸収はサービス層を正として任せる
        const merged = await queryInventoryDetailByProductAndToken(pbId, tbId);

        if (cancelled) return;

        const nextRows = Array.isArray(merged.rows) ? merged.rows : [];

        console.log("[inventory/useInventoryDetail] fetch ok", {
          inventoryKey: merged.inventoryKey,
          inventoryIds: merged.inventoryIds?.length ?? 0,
          rowsCount: nextRows.length,
          totalStock: merged.totalStock,
          updatedAt: merged.updatedAt,
        });

        setVm(merged);
        setRows(nextRows);
      } catch (e: any) {
        if (cancelled) return;

        const msg = String(e?.message ?? e);

        console.error("[inventory/useInventoryDetail] fetch error", {
          pbId,
          tbId,
          error: msg,
          raw: e,
        });

        setError(msg);
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
  }, [pbId, tbId]);

  return { vm, rows, loading, error };
}
