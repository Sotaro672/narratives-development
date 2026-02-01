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
 * - 期待値: inventory/application の PriceRow（priceCard.types.ts）を正とする
 * - 識別子は modelId のみ（= PriceRow.id）を正とする
 * - 並び順は displayOrder（未設定は null を保持）
 * - 名揺れ補完はしない（Size/Color/Index での補完はしない）
 */
export function mapDTOToPriceRows(dto: ListCreateDTO | null): PriceRow[] {
  const rowsAny: any[] = Array.isArray((dto as any)?.priceRows)
    ? ((dto as any).priceRows as any[])
    : Array.isArray((dto as any)?.PriceRows)
      ? ((dto as any).PriceRows as any[])
      : [];

  return rowsAny.flatMap((r: any) => {
    const id = s(r?.modelId ?? r?.ModelId);
    if (!id) return []; // modelId が無い行は捨てる（識別子が正）

    const displayOrderRaw = r?.displayOrder ?? r?.DisplayOrder;
    const displayOrder =
      displayOrderRaw === null || displayOrderRaw === undefined
        ? null
        : (Number(displayOrderRaw) as number);

    const size = s(r?.size ?? r?.Size) || "-";
    const color = s(r?.color ?? r?.Color) || "-";

    const stock0 = Number(r?.stock ?? r?.Stock ?? 0);
    const safeStock = Number.isFinite(stock0) ? stock0 : 0;

    const rgb = r?.rgb ?? r?.RGB;
    const price = r?.price ?? r?.Price;

    const row: PriceRow = {
      id, // ✅ modelId -> id
      displayOrder, // ✅ displayOrder を保持（未設定は null）
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
 * ✅ 初期表示用: PriceRow[] を返す
 */
export function initPriceRowsFromDTO(dto: ListCreateDTO | null): PriceRow[] {
  return mapDTOToPriceRows(dto);
}
