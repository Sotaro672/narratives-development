// frontend/console/inventory/src/application/listCreate/listCreate.dto.ts

import type { ListCreateDTO } from "../../infrastructure/http/inventoryRepositoryHTTP";
import type { PriceRow } from "./listCreate.types";

export function extractDisplayStrings(dto: ListCreateDTO | null): {
  productBrandName: string;
  productName: string;
  tokenBrandName: string;
  tokenName: string;
} {
  return {
    productBrandName: dto?.productBrandName ?? "",
    productName: dto?.productName ?? "",
    tokenBrandName: dto?.tokenBrandName ?? "",
    tokenName: dto?.tokenName ?? "",
  };
}

/**
 * backend の ListCreateDTO.priceRows を PriceCard 用 PriceRow[] に変換
 * - 期待値: inventory/application の PriceRow（listCreate.types.ts）を正とする
 * - 識別子は modelId を正とする
 * - 並び順は displayOrder（未設定は null を保持）
 * - 名揺れ補完はしない
 */
export function mapDTOToPriceRows(dto: ListCreateDTO | null): PriceRow[] {
  const rows = Array.isArray(dto?.priceRows) ? dto.priceRows : [];

  return rows.map((r: any) => {
    const displayOrderRaw = r.displayOrder;
    const displayOrder =
      displayOrderRaw === null || displayOrderRaw === undefined
        ? null
        : Number(displayOrderRaw);

    const stockRaw = Number(r.stock ?? 0);
    const stock = Number.isFinite(stockRaw) ? stockRaw : 0;

    const row: PriceRow = {
      modelId: r.modelId,
      displayOrder,
      size: r.size,
      color: r.color,
      stock,
      rgb: r.rgb as any,
      price: r.price === undefined ? null : (r.price as any),
    };

    return row;
  });
}

/**
 * 初期表示用: PriceRow[] を返す
 */
export function initPriceRowsFromDTO(dto: ListCreateDTO | null): PriceRow[] {
  return mapDTOToPriceRows(dto);
}