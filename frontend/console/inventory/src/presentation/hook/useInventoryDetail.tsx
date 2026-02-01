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

function normalizeTokenBlueprintPatch(raw: any): TokenBlueprintPatchDTO | undefined {
  if (!raw) return undefined;

  const mintedRaw = raw?.minted;
  const minted: boolean | null | undefined =
    mintedRaw === undefined
      ? undefined
      : mintedRaw === null
        ? null
        : typeof mintedRaw === "boolean"
          ? mintedRaw
          : String(mintedRaw).trim().toLowerCase() === "true";

  const tokenName = asString(raw?.tokenName) || undefined;
  const symbol = asString(raw?.symbol) || undefined;
  const brandId = asString(raw?.brandId) || undefined;
  const brandName = asString(raw?.brandName) || undefined;
  const description = asString(raw?.description) || undefined;
  const metadataUri = asString(raw?.metadataUri) || undefined;
  const iconUrl = asString(raw?.iconUrl) || undefined;

  return {
    tokenName: tokenName ?? null,
    symbol: symbol ?? null,
    brandId: brandId ?? null,
    brandName: brandName ?? null,
    description: description ?? null,
    minted: minted ?? null,
    metadataUri: metadataUri ?? null,
    iconUrl: iconUrl ?? null,
  };
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

        // ① pbId + tbId → inventoryId を Raw API（list-create）から解決
        const listCreateRaw = await getListCreateRaw({
          productBlueprintId: pbId,
          tokenBlueprintId: tbId,
        });

        const inventoryId = asString((listCreateRaw as any)?.inventoryId);
        if (!inventoryId) {
          throw new Error(
            "inventoryId is empty (failed to resolve inventoryId from list-create)",
          );
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

        // ④ InventoryCard 用 rows を DTO（detail.rows）から直接生成
        const nextRows: InventoryRow[] = dtoRowsToInventoryRows(detail);

        // ⑤ vm も DTO（detail）から直接、最小構成で生成
        const totalStock =
          (detail as any)?.totalStock !== undefined && (detail as any)?.totalStock !== null
            ? asNumber((detail as any).totalStock)
            : nextRows.reduce((sum, r) => sum + asNumber(r.stock), 0);

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
