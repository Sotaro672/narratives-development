// frontend/amol/src/features/brand/mappers/brandMapper.ts

import {
  isRecord,
} from "../../../components/utils/typeGuards";

import {
  textOrEmpty,
} from "../../../components/utils/textOrEmpty";

import type {
  BrandDetail,
  BrandListItem,
  ListPriceRow,
} from "../types/brand";

function numberValue(
  value: unknown,
): number | undefined {
  if (
    typeof value === "number" &&
    Number.isFinite(value)
  ) {
    return value;
  }

  if (typeof value === "string") {
    const normalizedValue =
      value.trim();

    if (!normalizedValue) {
      return undefined;
    }

    const parsedValue =
      Number(normalizedValue);

    if (
      Number.isFinite(
        parsedValue,
      )
    ) {
      return parsedValue;
    }
  }

  return undefined;
}

function stringArrayValue(
  value: unknown,
): string[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .map((item) =>
      textOrEmpty(item).trim(),
    )
    .filter(Boolean);
}

function unwrapData(
  value: unknown,
): Record<string, unknown> {
  if (
    !isRecord(value) ||
    Array.isArray(value)
  ) {
    throw new Error(
      "APIレスポンスの形式が不正です。",
    );
  }

  const data =
    value.data;

  if (
    isRecord(data) &&
    !Array.isArray(data)
  ) {
    return unwrapData(data);
  }

  return value;
}

function unwrapListItem(
  value: unknown,
): Record<string, unknown> {
  const root =
    unwrapData(value);

  if (
    isRecord(root.item) &&
    !Array.isArray(root.item)
  ) {
    return root.item;
  }

  if (
    isRecord(root.list) &&
    !Array.isArray(root.list)
  ) {
    return root.list;
  }

  return root;
}

function priceRowsValue(
  value: unknown,
): ListPriceRow[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .filter(
      (
        row,
      ): row is Record<string, unknown> =>
        isRecord(row) &&
        !Array.isArray(row),
    )
    .map((row) => {
      return {
        ...row,

        currency:
          textOrEmpty(
            row.currency,
          ),

        amount:
          numberValue(
            row.amount,
          ),

        price:
          numberValue(
            row.price,
          ),
      };
    });
}

export function brandDetailFromJson(
  raw: unknown,
): BrandDetail {
  const json =
    unwrapData(raw);

  return {
    brandId:
      textOrEmpty(
        json.brandId,
      ),

    brandName:
      textOrEmpty(
        json.brandName,
      ),

    websiteUrl:
      textOrEmpty(
        json.websiteUrl ??
          json.url,
      ),

    brandIcon:
      textOrEmpty(
        json.brandIcon,
      ),

    brandBackgroundImage:
      textOrEmpty(
        json.brandBackgroundImage,
      ),

    description:
      textOrEmpty(
        json.description,
      ),

    companyId:
      textOrEmpty(
        json.companyId,
      ),

    companyName:
      textOrEmpty(
        json.companyName,
      ),

    inventoryIds:
      stringArrayValue(
        json.inventoryIds,
      ),

    listIds:
      stringArrayValue(
        json.listIds,
      ),
  };
}

export function brandListItemFromJson(
  raw: unknown,
  fallbackId: string,
): BrandListItem {
  const json =
    unwrapListItem(raw);

  const normalizedFallbackId =
    fallbackId.trim();

  return {
    id:
      textOrEmpty(
        json.id,
      ) ||
      normalizedFallbackId,

    title:
      textOrEmpty(
        json.title,
      ),

    description:
      textOrEmpty(
        json.description,
      ),

    image:
      textOrEmpty(
        json.image ??
          json.imageUrl ??
          json.thumbnailUrl,
      ),

    prices:
      priceRowsValue(
        json.prices,
      ),

    inventoryId:
      textOrEmpty(
        json.inventoryId,
      ) || undefined,

    productBlueprintId:
      textOrEmpty(
        json.productBlueprintId,
      ) || undefined,

    tokenBlueprintId:
      textOrEmpty(
        json.tokenBlueprintId,
      ) || undefined,
  };
}