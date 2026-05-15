// frontend/console/inventory/src/infrastructure/http/listCreateRepositoryHTTP.mappers.ts

import type {
  ListCreateDTO,
  ListCreatePriceRowDTO,
} from "./listCreateRepositoryHTTP.types";

export function mapListCreateDTO(data: any): ListCreateDTO {
  const rawRows: any[] = Array.isArray(data?.priceRows) ? data.priceRows : [];

  const priceRows: ListCreatePriceRowDTO[] = rawRows.flatMap((r: any) => {
    const modelId = r?.modelId;
    if (!modelId) return [];

    const hasPriceField = r?.price !== undefined;
    const rawPrice = r?.price;
    const price: number | null | undefined =
      !hasPriceField ? undefined : rawPrice === null ? null : Number(rawPrice);

    const row: ListCreatePriceRowDTO = {
      modelId,
      size: r?.size || "-",
      color: r?.color || "-",
      stock: Number(r?.stock ?? 0),
      ...(r?.rgb === undefined ? {} : { rgb: r.rgb }),
      ...(price === undefined ? {} : { price }),
    };

    return [row];
  });

  const totalStockRaw = data?.totalStock;

  return {
    inventoryId: data?.inventoryId,
    productBlueprintId: data?.productBlueprintId,
    tokenBlueprintId: data?.tokenBlueprintId,

    productBrandName: data?.productBrandName,
    productName: data?.productName,

    tokenBrandName: data?.tokenBrandName,
    tokenName: data?.tokenName,

    priceRows,
    totalStock:
      totalStockRaw === undefined || totalStockRaw === null
        ? undefined
        : Number(totalStockRaw),
  };
}