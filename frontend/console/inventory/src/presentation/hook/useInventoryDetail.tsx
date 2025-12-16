// frontend/console/inventory/src/presentation/hook/useInventoryDetail.tsx
import * as React from "react";
import { fetchInventoryDetailDTO } from "../../infrastructure/http/inventoryRepositoryHTTP";
import type { InventoryRow } from "../components/inventoryCard";

export type UseInventoryDetailResult = {
  rows: InventoryRow[];
  loading: boolean;
  error: string | null;
};

export function useInventoryDetail(
  inventoryId: string | undefined,
): UseInventoryDetailResult {
  const [rows, setRows] = React.useState<InventoryRow[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  // 画面マウント確認ログ（遷移できてるか）
  React.useEffect(() => {
    console.log("[inventory/useInventoryDetail] mounted", { inventoryId });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  React.useEffect(() => {
    if (!inventoryId) return;

    let cancelled = false;

    (async () => {
      try {
        setLoading(true);
        setError(null);

        console.log("[inventory/useInventoryDetail] fetch start", { inventoryId });

        const dto = await fetchInventoryDetailDTO(inventoryId);

        if (cancelled) return;

        console.log("[inventory/useInventoryDetail] fetch ok", {
          inventoryId,
          rowsCount: dto.rows?.length ?? 0,
          totalStock: dto.totalStock,
          dto,
        });

        setRows(dto.rows as unknown as InventoryRow[]);
      } catch (e: any) {
        if (cancelled) return;

        console.error("[inventory/useInventoryDetail] fetch error", {
          inventoryId,
          error: String(e?.message ?? e),
          raw: e,
        });

        setError(String(e?.message ?? e));
        setRows([]);
      } finally {
        if (cancelled) return;
        setLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [inventoryId]);

  return { rows, loading, error };
}
