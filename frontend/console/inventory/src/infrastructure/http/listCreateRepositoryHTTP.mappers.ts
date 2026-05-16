// frontend/console/inventory/src/infrastructure/http/listCreateRepositoryHTTP.mappers.ts

import type {
  ListCreateDTO,
  ListCreateModelRefDTO,
  ListCreatePriceRowDTO,
} from "./listCreateRepositoryHTTP.types";

function toNullableNumber(value: unknown): number | null {
  if (value === undefined || value === null) return null;

  const n = Number(value);
  return Number.isFinite(n) ? n : null;
}

function toOptionalNumber(value: unknown): number | undefined {
  if (value === undefined || value === null) return undefined;

  const n = Number(value);
  return Number.isFinite(n) ? n : undefined;
}

function toNullableString(value: unknown): string | null {
  if (value === undefined || value === null) return null;

  const s = String(value).trim();
  return s || null;
}

function mapListCreateModelRefs(data: any): ListCreateModelRefDTO[] {
  const rawRefs: any[] = Array.isArray(data?.modelRefs) ? data.modelRefs : [];

  return rawRefs.flatMap((r: any) => {
    const modelId = toNullableString(r?.modelId);
    if (!modelId) return [];

    return [
      {
        modelId,
        displayOrder: toNullableNumber(r?.displayOrder),
      },
    ];
  });
}

function mapListCreatePriceRows(data: any): ListCreatePriceRowDTO[] {
  const rawRows: any[] = Array.isArray(data?.priceRows) ? data.priceRows : [];

  return rawRows.flatMap((r: any) => {
    const modelId = toNullableString(r?.modelId);
    if (!modelId) return [];

    const hasPriceField = r?.price !== undefined;
    const rawPrice = r?.price;
    const price: number | null | undefined =
      !hasPriceField ? undefined : rawPrice === null ? null : Number(rawPrice);

    const row: ListCreatePriceRowDTO = {
      modelId,
      kind: toNullableString(r?.kind),
      modelNumber: toNullableString(r?.modelNumber),
      displayOrder: toNullableNumber(r?.displayOrder),
      stock: toOptionalNumber(r?.stock) ?? 0,

      size: toNullableString(r?.size),
      color: toNullableString(r?.color),
      rgb: toNullableNumber(r?.rgb),

      volumeValue: toNullableNumber(r?.volumeValue),
      volumeUnit: toNullableString(r?.volumeUnit),

      ...(price === undefined ? {} : { price }),
    };

    return [row];
  });
}

export function mapListCreateDTO(data: any): ListCreateDTO {
  const totalStockRaw = data?.totalStock;

  return {
    inventoryId: data?.inventoryId,
    productBlueprintId: data?.productBlueprintId,
    tokenBlueprintId: data?.tokenBlueprintId,

    productBrandName: data?.productBrandName,
    productName: data?.productName,

    tokenBrandName: data?.tokenBrandName,
    tokenName: data?.tokenName,

    listImageUrl: data?.listImageUrl ?? null,

    modelRefs: mapListCreateModelRefs(data),

    priceRows: mapListCreatePriceRows(data),

    totalStock:
      totalStockRaw === undefined || totalStockRaw === null
        ? undefined
        : Number(totalStockRaw),

    priceNote: data?.priceNote ?? null,
    currencyJpy: Boolean(data?.currencyJpy),
  };
}