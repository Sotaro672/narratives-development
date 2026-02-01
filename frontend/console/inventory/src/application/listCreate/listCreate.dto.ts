// frontend/console/inventory/src/application/listCreate/listCreate.dto.ts

import type { ListCreateDTO } from "../../infrastructure/http/inventoryRepositoryHTTP";
import type { PriceRow } from "./listCreate.types";
import { s } from "./listCreate.utils";

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
 * - 名揺れは modelId のみを正とする（Size/Color/Index での補完はしない）
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
    const rgb = r?.rgb ?? r?.RGB;
    const price = r?.price ?? r?.Price;

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
 * ✅ dto.priceRows[].modelId を取り出す（名揺れ補完はしない）
 * - PriceRowEx は削除したため「modelId 配列」を返す
 */
export function extractModelIdsFromDTO(dto: any): string[] {
  const dtoRows: any[] = Array.isArray(dto?.priceRows) ? dto.priceRows : [];
  return dtoRows.map((dr) => s(dr?.modelId));
}

/**
 * ✅ 初期表示用: PriceRow[] を返す（modelId は別途 extractModelIdsFromDTO で扱う）
 */
export function initPriceRowsFromDTO(dto: ListCreateDTO | null): PriceRow[] {
  return mapDTOToPriceRows(dto);
}
