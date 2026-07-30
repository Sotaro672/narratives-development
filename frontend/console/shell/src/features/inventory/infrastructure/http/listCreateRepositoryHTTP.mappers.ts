// frontend/console/shell/src/features/inventory/infrastructure/http/listCreateRepositoryHTTP.mappers.ts

import type {
  ListCreateDTO,
  ListCreateModelRefDTO,
  ListCreatePriceRowDTO,
} from "./listCreateRepositoryHTTP.types";

function toNullableNumber(value: unknown): number | null {
  if (value === undefined || value === null) {
    return null;
  }

  const numberValue = Number(value);

  return Number.isFinite(numberValue)
    ? numberValue
    : null;
}

function toOptionalNumber(value: unknown): number | undefined {
  if (value === undefined || value === null) {
    return undefined;
  }

  const numberValue = Number(value);

  return Number.isFinite(numberValue)
    ? numberValue
    : undefined;
}

function toNullableString(value: unknown): string | null {
  if (value === undefined || value === null) {
    return null;
  }

  const stringValue = String(value).trim();

  return stringValue || null;
}

function mapListCreateModelRefs(
  data: any,
): ListCreateModelRefDTO[] {
  const rawRefs: any[] =
    Array.isArray(data?.modelRefs)
      ? data.modelRefs
      : [];

  return rawRefs.flatMap((rawRef: any) => {
    const modelId =
      toNullableString(rawRef?.modelId);

    if (!modelId) {
      return [];
    }

    return [
      {
        modelId,
        displayOrder:
          toNullableNumber(rawRef?.displayOrder),
      },
    ];
  });
}

function mapListCreatePriceRows(
  data: any,
): ListCreatePriceRowDTO[] {
  const rawRows: any[] =
    Array.isArray(data?.priceRows)
      ? data.priceRows
      : [];

  return rawRows.flatMap((rawRow: any) => {
    const modelId =
      toNullableString(rawRow?.modelId);

    if (!modelId) {
      return [];
    }

    const row: ListCreatePriceRowDTO = {
      modelId,

      kind:
        toNullableString(rawRow?.kind),

      modelNumber:
        toNullableString(rawRow?.modelNumber),

      displayOrder:
        toNullableNumber(rawRow?.displayOrder),

      stock:
        toOptionalNumber(rawRow?.stock) ?? 0,

      size:
        toNullableString(rawRow?.size),

      color:
        toNullableString(rawRow?.color),

      rgb:
        toNullableNumber(rawRow?.rgb),

      volumeValue:
        toNullableNumber(rawRow?.volumeValue),

      volumeUnit:
        toNullableString(rawRow?.volumeUnit),

      ...(
        rawRow?.price === undefined ||
        rawRow?.price === null
          ? {}
          : {
              price:
                rawRow.price,
            }
      ),
    };

    return [row];
  });
}

export function mapListCreateDTO(
  data: any,
): ListCreateDTO {
  const totalStockRaw =
    data?.totalStock;

  return {
    inventoryId:
      data?.inventoryId,

    productBlueprintId:
      data?.productBlueprintId,

    tokenBlueprintId:
      data?.tokenBlueprintId,

    productBrandName:
      data?.productBrandName,

    productName:
      data?.productName,

    tokenBrandName:
      data?.tokenBrandName,

    tokenName:
      data?.tokenName,

    listImageUrl:
      data?.listImageUrl ?? null,

    modelRefs:
      mapListCreateModelRefs(data),

    priceRows:
      mapListCreatePriceRows(data),

    totalStock:
      totalStockRaw === undefined ||
      totalStockRaw === null
        ? undefined
        : Number(totalStockRaw),

    priceNote:
      data?.priceNote ?? null,

    currencyJpy:
      Boolean(data?.currencyJpy),
  };
}