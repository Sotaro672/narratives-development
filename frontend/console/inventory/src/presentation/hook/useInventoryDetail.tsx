// frontend\console\inventory\src\presentation\hook\useInventoryDetail.tsx

import * as React from "react";
import type { InventoryRow } from "../../application/inventoryTypes";

import {
  fetchListCreateDTO,
  fetchInventoryDetailDTO,
  fetchTokenBlueprintPatchDTO,
  type InventoryDetailDTO,
  type TokenBlueprintPatchDTO,
  type ProductBlueprintPatchDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP";

export type InventoryDetailViewModel = {
  inventoryKey: string;
  inventoryId: string;

  tokenBlueprintId: string;
  productBlueprintId: string;

  productBlueprintPatch: ProductBlueprintPatchDTO;
  tokenBlueprintPatch?: TokenBlueprintPatchDTO;

  updatedAt?: string;
  totalStock: number;

  // InventoryCard に渡す最小
  rows: InventoryRow[];
};

export type UseInventoryDetailResult = {
  vm: InventoryDetailViewModel | null;
  rows: InventoryRow[];
  loading: boolean;
  error: string | null;
};

function asString(v: any): string {
  return String(v ?? "").trim();
}

function asNumber(v: any): number {
  const n = Number(v ?? 0);
  return Number.isFinite(n) ? n : 0;
}

function dtoRowsToInventoryRows(dto: InventoryDetailDTO): InventoryRow[] {
  const rowsRaw: any[] = Array.isArray((dto as any)?.rows) ? ((dto as any).rows as any[]) : [];

  return rowsRaw.map((r: any) => ({
    // InventoryCard は token を使わないが、InventoryRow 型都合で空文字を入れておく
    token: asString(r?.token) || "",
    modelNumber: asString(r?.modelNumber),
    size: asString(r?.size),
    color: asString(r?.color),
    rgb: (r?.rgb ?? null) as any,
    stock: asNumber(r?.stock),
  }));
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

        // ② inventory detail 取得（rows を直接 InventoryCard に渡すための元データ）
        const detail: InventoryDetailDTO = await fetchInventoryDetailDTO(inventoryId);
        if (cancelled) return;

        // ③ tokenBlueprintPatch（主に iconUrl 用）を取得（失敗しても続行）
        let tokenBlueprintPatch: TokenBlueprintPatchDTO | undefined = undefined;
        try {
          tokenBlueprintPatch = await fetchTokenBlueprintPatchDTO(tbId);
        } catch {
          tokenBlueprintPatch = undefined;
        }
        if (cancelled) return;

        // ④ DTO から InventoryCard 用 rows を直接生成（mergeDetailDTOs 不要）
        const nextRows: InventoryRow[] = dtoRowsToInventoryRows(detail);

        // ⑤ vm も「DTOから直接」最小構成で作る
        const totalStock = nextRows.reduce((sum, r) => sum + asNumber(r.stock), 0);

        const nextVm: InventoryDetailViewModel = {
          inventoryKey: `${pbId}__${tbId}`,
          inventoryId,

          tokenBlueprintId: asString((detail as any)?.tokenBlueprintId) || tbId,
          productBlueprintId: asString((detail as any)?.productBlueprintId) || pbId,

          productBlueprintPatch: ((detail as any)?.productBlueprintPatch ?? {}) as ProductBlueprintPatchDTO,
          tokenBlueprintPatch,

          updatedAt: (detail as any)?.updatedAt ? String((detail as any).updatedAt) : undefined,
          totalStock,

          rows: nextRows,
        };

        setVm(nextVm);
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
