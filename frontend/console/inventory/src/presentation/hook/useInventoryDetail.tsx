// frontend/console/inventory/src/presentation/hook/useInventoryDetail.tsx

import * as React from "react";
import type { InventoryRow } from "../../application/inventoryTypes";
import type { InventoryDetailViewModel } from "../../application/inventoryDetail/inventoryDetail.types";
import { loadInventoryDetailViewModel } from "../../application/inventoryDetail/inventoryDetail.usecase";

export type UseInventoryDetailResult = {
  vm: InventoryDetailViewModel | null;
  rows: InventoryRow[];
  loading: boolean;
  error: string | null;
};

export function useInventoryDetail(
  inventoryId: string | undefined,
): UseInventoryDetailResult {
  const [vm, setVm] = React.useState<InventoryDetailViewModel | null>(null);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const invId = React.useMemo(() => inventoryId ?? "", [inventoryId]);

  React.useEffect(() => {
    if (!invId) {
      setVm(null);
      setError(null);
      setLoading(false);
      return;
    }

    let cancelled = false;

    (async () => {
      try {
        setLoading(true);
        setError(null);

        const nextVm = await loadInventoryDetailViewModel(invId);

        if (cancelled) return;

        setVm(nextVm);
      } catch (e: any) {
        if (cancelled) return;

        setError(String(e?.message ?? e));
        setVm(null);
      } finally {
        if (cancelled) return;
        setLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [invId]);

  const rows = React.useMemo<InventoryRow[]>(() => vm?.rows ?? [], [vm]);

  return {
    vm,
    rows,
    loading,
    error,
  };
}