// frontend\console\inventory\src\presentation\hook\useInventoryDetail.tsx

import * as React from "react";
import type { InventoryRow } from "../../application/inventoryTypes";

// ✅ fetcher（inventoryRepositoryHTTP.fetchers.ts）を経由しない：Raw API を直接叩く
import {
  getListCreateRaw,
  getInventoryDetailRaw,
  getTokenBlueprintPatchRaw,
} from "../../infrastructure/api/inventoryApi";

// ✅ DTO 型は types.ts から直接 import（barrel/fetcher 依存を避ける）
import type {
  InventoryDetailDTO,
  TokenBlueprintPatchDTO,
  ProductBlueprintPatchDTO,
  ProductBlueprintModelRefDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP.types";

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

function asInt(v: any): number | undefined {
  const n = Number(v);
  if (!Number.isFinite(n)) return undefined;
  const i = Math.trunc(n);
  return i > 0 ? i : undefined;
}

function normalizeTokenBlueprintPatch(raw: any): TokenBlueprintPatchDTO | undefined {
  if (!raw) return undefined;

  const tokenName = asString(raw?.tokenName) || undefined;
  const symbol = asString(raw?.symbol) || undefined;
  const brandId = asString(raw?.brandId) || undefined;
  const brandName = asString(raw?.brandName) || undefined;
  const description = asString(raw?.description) || undefined;
  const iconUrl = asString(raw?.iconUrl) || undefined;

  return {
    tokenName: tokenName ?? null,
    symbol: symbol ?? null,
    brandId: brandId ?? null,
    brandName: brandName ?? null,
    description: description ?? null,
    iconUrl: iconUrl ?? null,
  };
}

function buildModelDisplayOrderMap(patch: ProductBlueprintPatchDTO | undefined): Record<string, number> {
  const refs = (patch as any)?.modelRefs as ProductBlueprintModelRefDTO[] | null | undefined;
  if (!Array.isArray(refs)) return {};

  const out: Record<string, number> = {};
  for (const r of refs) {
    const modelId = asString((r as any)?.modelId);
    const displayOrder = asInt((r as any)?.displayOrder);
    if (!modelId || displayOrder === undefined) continue;
    out[modelId] = displayOrder;
  }
  return out;
}

function dtoRowsToInventoryRows(dto: InventoryDetailDTO, modelOrderById: Record<string, number>): InventoryRow[] {
  const rowsRaw: any[] = Array.isArray((dto as any)?.rows) ? ((dto as any).rows as any[]) : [];

  return rowsRaw.map((r: any) => {
    const modelId = asString(r?.modelId); // ✅ backend が返す想定（なければ ""）
    const displayOrder = modelId ? modelOrderById[modelId] : undefined;

    return {
      // InventoryCard は token を使わないが、InventoryRow 型都合で空文字を入れておく
      token: asString(r?.token) || "",
      modelNumber: asString(r?.modelNumber),
      size: asString(r?.size),
      color: asString(r?.color),
      rgb: (r?.rgb ?? null) as any,
      stock: asNumber(r?.stock),

      // ✅ NEW: displayOrder を InventoryCard に渡せるように付与
      // ※ InventoryRow 側に displayOrder?: number を追加して受ける前提
      displayOrder,
    } as InventoryRow;
  });
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

        // ① pbId + tbId → inventoryId を Raw API（list-create）から解決
        const listCreateRaw = await getListCreateRaw({
          productBlueprintId: pbId,
          tokenBlueprintId: tbId,
        });

        const inventoryId = asString((listCreateRaw as any)?.inventoryId);
        if (!inventoryId) {
          throw new Error("inventoryId is empty (failed to resolve inventoryId from list-create)");
        }

        // ② inventory detail を Raw API から取得
        const detailRaw = (await getInventoryDetailRaw(inventoryId)) as any;
        const detail: InventoryDetailDTO = detailRaw as InventoryDetailDTO;
        if (cancelled) return;

        // ③ tokenBlueprintPatch（主に iconUrl 用）を Raw API から取得（失敗しても続行）
        let tokenBlueprintPatch: TokenBlueprintPatchDTO | undefined = undefined;
        try {
          const patchRaw = await getTokenBlueprintPatchRaw(tbId);
          tokenBlueprintPatch = normalizeTokenBlueprintPatch(patchRaw);
        } catch {
          tokenBlueprintPatch = undefined;
        }
        if (cancelled) return;

        // ✅ productBlueprintPatch から modelRefs(displayOrder) を拾う
        const productBlueprintPatch = ((detail as any)?.productBlueprintPatch ?? {}) as ProductBlueprintPatchDTO;
        const modelOrderById = buildModelDisplayOrderMap(productBlueprintPatch);

        // ④ InventoryCard 用 rows を DTO（detail.rows）から直接生成（displayOrder 付与）
        const nextRows: InventoryRow[] = dtoRowsToInventoryRows(detail, modelOrderById);

        // ⑤ vm も DTO（detail）から直接、最小構成で生成
        const totalStock =
          (detail as any)?.totalStock !== undefined && (detail as any)?.totalStock !== null
            ? asNumber((detail as any).totalStock)
            : nextRows.reduce((sum, r) => sum + asNumber((r as any).stock), 0);

        const nextVm: InventoryDetailViewModel = {
          inventoryKey: `${pbId}__${tbId}`,
          inventoryId,

          tokenBlueprintId: asString((detail as any)?.tokenBlueprintId) || tbId,
          productBlueprintId: asString((detail as any)?.productBlueprintId) || pbId,

          productBlueprintPatch,
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
