// frontend/console/inventory/src/application/listCreateService.tsx

import type * as React from "react";
import type { PriceRow } from "../../../list/src/presentation/hook/usePriceCard";

import {
  fetchListCreateDTO,
  type ListCreateDTO,
} from "../infrastructure/http/inventoryRepositoryHTTP";

function s(v: unknown): string {
  return String(v ?? "").trim();
}

// ✅ NEW: RefObject は null を含むのが正しい（useRef(initial=null) のため）
export type ImageInputRef = React.RefObject<HTMLInputElement | null>;

export type ListCreateRouteParams = {
  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
};

export type ResolvedListCreateParams = {
  inventoryId: string;
  productBlueprintId: string;
  tokenBlueprintId: string;
  raw: ListCreateRouteParams;
};

export function resolveListCreateParams(
  raw: ListCreateRouteParams,
): ResolvedListCreateParams {
  return {
    inventoryId: s(raw?.inventoryId),
    productBlueprintId: s(raw?.productBlueprintId),
    tokenBlueprintId: s(raw?.tokenBlueprintId),
    raw,
  };
}

export function computeListCreateTitle(inventoryId: string): string {
  return inventoryId ? `出品作成（inventoryId: ${inventoryId}）` : "出品作成";
}

export function canFetchListCreate(p: ResolvedListCreateParams): boolean {
  return Boolean(p.inventoryId) || (Boolean(p.productBlueprintId) && Boolean(p.tokenBlueprintId));
}

export function buildListCreateFetchInput(p: ResolvedListCreateParams): {
  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
} {
  return {
    inventoryId: p.inventoryId || undefined,
    productBlueprintId: p.productBlueprintId || undefined,
    tokenBlueprintId: p.tokenBlueprintId || undefined,
  };
}

export function getInventoryIdFromDTO(dto: ListCreateDTO | null | undefined): string {
  return s((dto as any)?.inventoryId ?? (dto as any)?.InventoryID);
}

export function shouldRedirectToInventoryIdRoute(args: {
  currentInventoryId: string;
  gotInventoryId: string;
  alreadyRedirected: boolean;
}): boolean {
  return !args.alreadyRedirected && !args.currentInventoryId && Boolean(args.gotInventoryId);
}

export function buildInventoryDetailPath(pbId: string, tbId: string): string {
  const pb = s(pbId);
  const tb = s(tbId);
  if (!pb || !tb) return "/inventory";
  return `/inventory/detail/${encodeURIComponent(pb)}/${encodeURIComponent(tb)}`;
}

export function buildInventoryListCreatePath(inventoryId: string): string {
  const id = s(inventoryId);
  if (!id) return "/inventory/list/create";
  return `/inventory/list/create/${encodeURIComponent(id)}`;
}

export function buildBackPath(p: ResolvedListCreateParams): string {
  // ✅ 詳細へは pb/tb で戻す
  if (p.productBlueprintId && p.tokenBlueprintId) {
    return buildInventoryDetailPath(p.productBlueprintId, p.tokenBlueprintId);
  }
  return "/inventory";
}

export function buildAfterCreatePath(p: ResolvedListCreateParams): string {
  // ✅ 作成後も pb/tb があれば detail へ
  if (p.productBlueprintId && p.tokenBlueprintId) {
    return buildInventoryDetailPath(p.productBlueprintId, p.tokenBlueprintId);
  }
  return "/inventory";
}

export function extractDisplayStrings(dto: ListCreateDTO | null): {
  productBrandName: string;
  productName: string;
  tokenBrandName: string;
  tokenName: string;
} {
  return {
    productBrandName: s(dto?.productBrandName),
    productName: s(dto?.productName),
    tokenBrandName: s(dto?.tokenBrandName),
    tokenName: s(dto?.tokenName),
  };
}

/**
 * ✅ backend の ListCreateDTO.priceRows を PriceCard 用 PriceRow[] に変換
 * - dto 側に priceRows が無ければ []
 */
export function mapDTOToPriceRows(dto: ListCreateDTO | null): PriceRow[] {
  const rowsAny: any[] = Array.isArray((dto as any)?.priceRows)
    ? ((dto as any).priceRows as any[])
    : Array.isArray((dto as any)?.PriceRows)
      ? ((dto as any).PriceRows as any[])
      : [];

  return rowsAny.flatMap((r: any) => {
    const size = s(r?.size ?? r?.Size) || "-";
    const color = s(r?.color ?? r?.Color) || "-";
    const stock = Number(r?.stock ?? r?.Stock ?? 0);
    const rgb = r?.rgb ?? r?.RGB; // number|null|undefined 想定
    const price = r?.price ?? r?.Price;

    // stock が数値でない場合の防御
    const safeStock = Number.isFinite(stock) ? stock : 0;

    const row: PriceRow = {
      size,
      color,
      stock: safeStock,
      rgb: rgb as any,
      price: price === undefined ? null : (price as any),
    };

    return [row];
  });
}

/**
 * ✅ ListCreateDTO を取得する（Hook からはこれだけ呼ぶ）
 */
export async function loadListCreateDTOFromParams(
  p: ResolvedListCreateParams,
): Promise<ListCreateDTO> {
  const input = buildListCreateFetchInput(p);
  return await fetchListCreateDTO(input);
}
