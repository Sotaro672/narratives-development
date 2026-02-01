// frontend\console\inventory\src\presentation\hook\useInventoryDetail.tsx

import * as React from "react";
import type { InventoryRow } from "../../application/inventoryTypes";

// ✅ fetcher を経由しない：Raw API を直接叩く
import { getInventoryDetailRaw, getTokenBlueprintPatchRaw } from "../../infrastructure/api/inventoryApi";

// ✅ ListCreate Raw API は分離したため別 import
import { getListCreateRaw } from "../../infrastructure/api/listCreateApi";

// ✅ DTO 型は types.ts から直接 import
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

  // ✅ Header 表示用（productName / tokenName）
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
 * ✅ 商品IDタグを「QRコード」のみに正規化する
 * - {"Type":"QRコード"} のような JSON文字列/オブジェクトが来ても「QRコード」だけにする
 * - QR 以外なら null（表示しない前提）
 */
function normalizeProductIdTagToQRCodeOnly(v: any): string | null | undefined {
  if (v === undefined) return undefined;
  if (v === null) return null;

  const toTag = (x: any): string => {
    if (x === undefined || x === null) return "";
    if (typeof x === "string") return x.trim();
    if (typeof x === "number" || typeof x === "boolean") return String(x).trim();
    if (typeof x === "object") {
      const o: any = x;
      const cand =
        o?.Type ??
        o?.type ??
        o?.label ??
        o?.Label ??
        o?.value ??
        o?.Value ??
        o?.name ??
        o?.Name;
      return typeof cand === "string" ? cand.trim() : "";
    }
    return "";
  };

  // 1) 文字列
  if (typeof v === "string") {
    const s = v.trim();
    if (!s) return null;

    // JSONっぽい文字列なら parse して Type を取る
    if (s.startsWith("{") && s.endsWith("}")) {
      try {
        const obj = JSON.parse(s);
        const inner = toTag(obj);
        const low = inner.toLowerCase();
        if (inner === "QRコード" || low === "qr" || low.includes("qr")) return "QRコード";
        return null;
      } catch {
        // parse できないならそのまま判定
        const low = s.toLowerCase();
        if (s === "QRコード" || low === "qr" || low.includes("qr")) return "QRコード";
        return null;
      }
    }

    const low = s.toLowerCase();
    if (s === "QRコード" || low === "qr" || low.includes("qr")) return "QRコード";
    return null;
  }

  // 2) オブジェクト
  if (typeof v === "object") {
    const t = toTag(v);
    const low = t.toLowerCase();
    if (t === "QRコード" || low === "qr" || low.includes("qr")) return "QRコード";
    return null;
  }

  // 3) その他（数値/boolean等）
  const t = toTag(v);
  const low = t.toLowerCase();
  if (t === "QRコード" || low === "qr" || low.includes("qr")) return "QRコード";
  return null;
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
    const modelId = asString(r?.modelId); // ✅ backend が返す想定（なければ ""）
    const displayOrder = modelId ? modelOrderById[modelId] : undefined;

    return {
      token: asString(r?.token) || "",
      modelNumber: asString(r?.modelNumber),
      size: asString(r?.size),
      color: asString(r?.color),
      rgb: (r?.rgb ?? null) as any,
      stock: asNumber(r?.stock),

      // ✅ NEW: displayOrder を InventoryCard に渡せるように付与
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

        // ✅ Header 用：list-create から productName / tokenName を拾う（最優先）
        const productNameFromLC = asString((listCreateRaw as any)?.productName) || "-";
        const tokenNameFromLC = asString((listCreateRaw as any)?.tokenName) || tbId;

        // ② inventory detail を Raw API から取得
        const detailRaw = (await getInventoryDetailRaw(inventoryId)) as any;
        const detail: InventoryDetailDTO = detailRaw as InventoryDetailDTO;
        if (cancelled) return;

        // ③ tokenBlueprintPatch（主に iconUrl、tokenName の補強にも使える）
        let tokenBlueprintPatch: TokenBlueprintPatchDTO | undefined = undefined;
        try {
          const patchRaw = await getTokenBlueprintPatchRaw(tbId);
          tokenBlueprintPatch = normalizeTokenBlueprintPatch(patchRaw);
        } catch {
          tokenBlueprintPatch = undefined;
        }
        if (cancelled) return;

        // ✅ productBlueprintPatch を取得し、商品IDタグを「QRコード」だけに正規化
        const rawProductBlueprintPatch = ((detail as any)?.productBlueprintPatch ?? {}) as ProductBlueprintPatchDTO;

        const normalizedProductIdTag =
          normalizeProductIdTagToQRCodeOnly((rawProductBlueprintPatch as any)?.productIdTag) ?? null;

        const productBlueprintPatch: ProductBlueprintPatchDTO = {
          ...rawProductBlueprintPatch,
          productIdTag: normalizedProductIdTag,
        };

        // ✅ modelRefs(displayOrder) を拾う
        const modelOrderById = buildModelDisplayOrderMap(productBlueprintPatch);

        // ④ InventoryCard 用 rows を DTO（detail.rows）から直接生成（displayOrder 付与）
        const nextRows: InventoryRow[] = dtoRowsToInventoryRows(detail, modelOrderById);

        // ⑤ totalStock
        const totalStock =
          (detail as any)?.totalStock !== undefined && (detail as any)?.totalStock !== null
            ? asNumber((detail as any).totalStock)
            : nextRows.reduce((sum, r) => sum + asNumber((r as any).stock), 0);

        // ✅ Header 表示用の productName / tokenName を確定
        // - productName: list-create を正（detail側が空でも）
        // - tokenName: patch の tokenName が取れればそれ、なければ list-create
        const productName = productNameFromLC;
        const tokenName = asString((tokenBlueprintPatch as any)?.tokenName) || tokenNameFromLC || tbId;

        const nextVm: InventoryDetailViewModel = {
          inventoryKey: `${pbId}__${tbId}`,
          inventoryId,

          tokenBlueprintId: asString((detail as any)?.tokenBlueprintId) || tbId,
          productBlueprintId: asString((detail as any)?.productBlueprintId) || pbId,

          // ✅ header
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
  }, [pbId, tbId]);

  return { vm, rows, loading, error };
}
