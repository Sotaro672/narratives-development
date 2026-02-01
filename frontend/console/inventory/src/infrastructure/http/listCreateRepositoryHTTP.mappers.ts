// frontend/console/inventory/src/infrastructure/http/listCreateRepositoryHTTP.mappers.ts

import type {
  ListCreateDTO,
  ListCreatePriceRowDTO,
} from "./listCreateRepositoryHTTP.types";

import { s, n, toRgbNumberOrNull } from "./inventoryRepositoryHTTP.utils";

// ---------------------------------------------------------
// ListCreate mapper（縮小）
// 互換吸収を削除し、想定キーのみ読む
// ---------------------------------------------------------
export function mapListCreateDTO(data: any): ListCreateDTO {
  const rawRows: any[] = Array.isArray(data?.priceRows) ? data.priceRows : [];

  const priceRows: ListCreatePriceRowDTO[] = rawRows.flatMap((r: any) => {
    const modelId = s(r?.modelId);
    if (!modelId) return [];

    const rgbVal = toRgbNumberOrNull(r?.rgb);
    const stock = n(r?.stock);

    const hasPriceField = r?.price !== undefined;
    const rawPrice = r?.price;
    const price: number | null | undefined =
      !hasPriceField ? undefined : rawPrice === null ? null : n(rawPrice);

    const row: ListCreatePriceRowDTO = {
      modelId,
      size: s(r?.size) || "-",
      color: s(r?.color) || "-",
      stock,
      ...(rgbVal === undefined ? {} : { rgb: rgbVal }),
      ...(price === undefined ? {} : { price }),
    };

    return [row];
  });

  const totalStockRaw = data?.totalStock;

  return {
    inventoryId: data?.inventoryId ? s(data.inventoryId) : undefined,
    productBlueprintId: data?.productBlueprintId ? s(data.productBlueprintId) : undefined,
    tokenBlueprintId: data?.tokenBlueprintId ? s(data.tokenBlueprintId) : undefined,

    productBrandName: s(data?.productBrandName),
    productName: s(data?.productName),

    tokenBrandName: s(data?.tokenBrandName),
    tokenName: s(data?.tokenName),

    priceRows,
    totalStock:
      totalStockRaw === undefined || totalStockRaw === null ? undefined : n(totalStockRaw),
  };
}
