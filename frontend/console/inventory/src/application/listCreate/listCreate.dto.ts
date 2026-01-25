// frontend/console/inventory/src/application/listCreate/listCreate.dto.ts

import type { ListCreateDTO } from "../../infrastructure/http/inventoryRepositoryHTTP";
import type { PriceRow, PriceRowEx } from "./listCreate.types";
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

export function attachModelIdsFromDTO(dto: any, baseRows: PriceRow[]): PriceRowEx[] {
  const dtoRows: any[] = Array.isArray(dto?.priceRows) ? dto.priceRows : [];

  const keyToModelId = new Map<string, string>();
  for (const dr of dtoRows) {
    const size = s(dr?.size);
    const color = s(dr?.color);
    const modelId = s(dr?.modelId);
    if (!size || !color || !modelId) continue;
    keyToModelId.set(`${size}__${color}`, modelId);
  }

  return baseRows.map((r, idx) => {
    const size = s((r as any)?.size);
    const color = s((r as any)?.color);
    const byKey = keyToModelId.get(`${size}__${color}`) ?? "";
    const byIndex = s(dtoRows[idx]?.modelId);
    const modelId = byKey || byIndex;

    return {
      ...(r as any),
      modelId,
    } as PriceRowEx;
  });
}

export function initPriceRowsFromDTO(dto: ListCreateDTO | null): PriceRowEx[] {
  const base = mapDTOToPriceRows(dto);
  return attachModelIdsFromDTO(dto as any, base);
}
