import * as React from "react";
import type { InventoryRow } from "../../application/inventoryTypes";

import {
  fetchListCreateDTO,
  fetchInventoryDetailDTO,
  fetchTokenBlueprintPatchDTO,
  type InventoryDetailDTO,
  type TokenBlueprintPatchDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP";

import { mergeDetailDTOs } from "../../application/inventoryDetail/inventoryDetail.mapper";

export type InventoryDetailViewModel = ReturnType<typeof mergeDetailDTOs>;

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

        // ① pbId + tbId → inventoryId を list-create から解決
        const listCreate = await fetchListCreateDTO({
          productBlueprintId: pbId,
          tokenBlueprintId: tbId,
        });

        const inventoryId = asString((listCreate as any)?.inventoryId);
        if (!inventoryId) {
          throw new Error(
            "inventoryId is empty (failed to resolve inventoryId from list-create)",
          );
        }

        // ② inventory detail 取得
        const detail: InventoryDetailDTO = await fetchInventoryDetailDTO(inventoryId);
        if (cancelled) return;

        // ③ tokenBlueprintPatch（主に iconUrl 用）を取得（失敗しても続行）
        let tokenBlueprintPatch: TokenBlueprintPatchDTO | null = null;
        try {
          tokenBlueprintPatch = await fetchTokenBlueprintPatchDTO(tbId);
        } catch {
          tokenBlueprintPatch = null;
        }
        if (cancelled) return;

        // ④ merge（従来の ViewModel 形に寄せる）
        const merged = mergeDetailDTOs(pbId, tbId, [inventoryId], [detail], tokenBlueprintPatch);
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
