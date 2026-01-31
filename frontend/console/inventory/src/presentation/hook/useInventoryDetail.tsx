// frontend/console/inventory/src/presentation/hook/useInventoryDetail.tsx

import * as React from "react";
import type { InventoryRow } from "../../application/inventoryTypes";
import {
  queryInventoryDetailByProductAndToken,
  type InventoryDetailViewModel,
} from "../../application/inventoryDetail/inventoryDetailService";

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

  const pbId = React.useMemo(() => asString(productBlueprintId), [productBlueprintId]);
  const tbId = React.useMemo(() => asString(tokenBlueprintId), [tokenBlueprintId]);

  React.useEffect(() => {
    if (!pbId || !tbId) {
      setVm(null);
      setRows([]);
      setError(null);
      setLoading(false);
      return;
    }

    let cancelled = false;

    (async () => {
      try {
        setLoading(true);
        setError(null);

        const merged = await queryInventoryDetailByProductAndToken(pbId, tbId);
        if (cancelled) return;

        const nextRows = Array.isArray((merged as any)?.rows) ? (merged as any).rows : [];
        setVm(merged);
        setRows(nextRows);
      } catch (e: any) {
        if (cancelled) return;
        const msg = String(e?.message ?? e);
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
