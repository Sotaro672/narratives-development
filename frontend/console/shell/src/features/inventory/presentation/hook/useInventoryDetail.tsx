// frontend/console/shell/src/features/inventory/presentation/hook/useInventoryDetail.tsx

import * as React from "react";

import type {
  InventoryDetailRowDTO,
  InventoryDetailViewModel,
} from "../../../../shared/types/inventory";
import { loadInventoryDetailViewModel } from "../../application/inventoryDetailService";

export type UseInventoryDetailResult = {
  vm: InventoryDetailViewModel | null;
  rows: InventoryDetailRowDTO[];
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

    async function load() {
      try {
        setLoading(true);
        setError(null);

        const nextVm = await loadInventoryDetailViewModel(invId);

        if (cancelled) {
          return;
        }

        setVm(nextVm);
      } catch (error) {
        if (cancelled) {
          return;
        }

        setError(error instanceof Error ? error.message : String(error));
        setVm(null);
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [invId]);

  const rows = React.useMemo<InventoryDetailRowDTO[]>(
    () => vm?.rows ?? [],
    [vm],
  );

  return {
    vm,
    rows,
    loading,
    error,
  };
}