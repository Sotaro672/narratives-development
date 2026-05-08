// frontend/console/inventory/src/presentation/hook/useInventoryDetail.tsx

import * as React from "react";
import type { InventoryRow } from "../../application/inventoryTypes";

// ✅ fetcher を経由しない：Raw API を直接叩く
import {
  getInventoryDetailRaw,
  getTokenBlueprintPatchRaw,
} from "../../infrastructure/api/inventoryApi";

// ✅ DTO 型は types.ts から直接 import
import type {
  InventoryDetailDTO,
  TokenBlueprintPatchDTO,
  ProductBlueprintPatchDTO,
  ProductBlueprintModelRefDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP.types";

export type InventoryDetailViewModel = {
  // ✅ inventory docId を正とする
  inventoryId: string;

  // ✅ inventory テーブルに両方記載されている前提で、そこから取得する（split/合成しない）
  productBlueprintId: string;
  tokenBlueprintId: string;

  // ✅ Header 表示用（productName / tokenName のみ）
  productName: string;
  tokenName: string;
  headerTitle: string; // `${productName} / ${tokenName}`

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

/**
 * ✅ Firestore productBlueprint 実データを正とするため、名揺れ吸収は削除する
 * - productIdTagType は "QRコード" が正（そのまま使う）
 * - 画面側の patch には余計な変換を掛けない
 */
function normalizeProductBlueprintPatch(raw: ProductBlueprintPatchDTO): ProductBlueprintPatchDTO {
  // ここでは “何もしない” を正とする（名揺れ吸収ロジックを全廃）
  return raw;
}

function buildModelDisplayOrderMap(
  patch: ProductBlueprintPatchDTO | undefined,
): Record<string, number> {
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

function dtoRowsToInventoryRows(
  dto: InventoryDetailDTO,
  modelOrderById: Record<string, number>,
): InventoryRow[] {
  const rowsRaw: any[] = Array.isArray((dto as any)?.rows) ? ((dto as any).rows as any[]) : [];

  return rowsRaw.map((r: any) => {
    const modelId = asString(r?.modelId);
    const displayOrder = modelId ? modelOrderById[modelId] : undefined;

    return {
      token: asString(r?.token) || "",
      modelNumber: asString(r?.modelNumber),
      size: asString(r?.size),
      color: asString(r?.color),
      rgb: (r?.rgb ?? null) as any,
      stock: asNumber(r?.stock),

      // ✅ displayOrder を InventoryCard に渡す
      displayOrder,
    } as InventoryRow;
  });
}

export function useInventoryDetail(inventoryId: string | undefined): UseInventoryDetailResult {
  const [vm, setVm] = React.useState<InventoryDetailViewModel | null>(null);
  const [rows, setRows] = React.useState<InventoryRow[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const invId = React.useMemo(() => asString(inventoryId), [inventoryId]);

  React.useEffect(() => {
    if (!invId) {
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

        // ① inventory detail を Raw API から取得（inventoryId(docId) を正とする）
        const detailRaw = (await getInventoryDetailRaw(invId)) as any;
        const detail: InventoryDetailDTO = detailRaw as InventoryDetailDTO;
        if (cancelled) return;

        // ② inventory テーブルにある pbId/tbId を「直接」取得（split/合成しない）
        const pbId = asString((detail as any)?.productBlueprintId);
        const tbId = asString((detail as any)?.tokenBlueprintId);

        if (!pbId || !tbId) {
          throw new Error("inventory_detail_missing_product_or_token_blueprint_id");
        }

        // ③ tokenBlueprintPatch（iconUrl / tokenName の補強）
        let tokenBlueprintPatch: TokenBlueprintPatchDTO | undefined = undefined;
        try {
          const patchRaw = await getTokenBlueprintPatchRaw(tbId);
          tokenBlueprintPatch = normalizeTokenBlueprintPatch(patchRaw);
        } catch {
          tokenBlueprintPatch = undefined;
        }
        if (cancelled) return;

        // ④ productBlueprintPatch（名揺れ吸収は全廃：実データを正としてそのまま使う）
        const rawProductBlueprintPatch = ((detail as any)?.productBlueprintPatch ?? {}) as ProductBlueprintPatchDTO;
        const productBlueprintPatch: ProductBlueprintPatchDTO =
          normalizeProductBlueprintPatch(rawProductBlueprintPatch);

        // ⑤ modelRefs(displayOrder) を拾う
        const modelOrderById = buildModelDisplayOrderMap(productBlueprintPatch);

        // ⑥ InventoryCard 用 rows を DTO（detail.rows）から生成（displayOrder 付与）
        const nextRows: InventoryRow[] = dtoRowsToInventoryRows(detail, modelOrderById);

        // ⑦ totalStock
        const totalStock =
          (detail as any)?.totalStock !== undefined && (detail as any)?.totalStock !== null
            ? asNumber((detail as any).totalStock)
            : nextRows.reduce((sum, r) => sum + asNumber((r as any).stock), 0);

        // ⑧ Header 表示（productName/tokenName のみ）
        const productName =
          asString((productBlueprintPatch as any)?.productName) ||
          asString((detail as any)?.productName) ||
          "-";

        const tokenName =
          asString((tokenBlueprintPatch as any)?.tokenName) ||
          asString((detail as any)?.tokenName) ||
          tbId ||
          "-";

        const nextVm: InventoryDetailViewModel = {
          inventoryId: invId,

          productBlueprintId: pbId,
          tokenBlueprintId: tbId,

          productName,
          tokenName,
          headerTitle: `${productName} / ${tokenName}`,

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
  }, [invId]);

  return { vm, rows, loading, error };
}
