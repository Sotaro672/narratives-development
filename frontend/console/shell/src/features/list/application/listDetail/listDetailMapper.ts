// frontend/console/shell/src/features/list/application/listDetail/listDetailMapper.ts

import {
  isValidListStatus,
  type ListStatus,
} from "../../../../shared/types/list";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";
import type { ListDetailDTO } from "../../infrastructure/dto/listDetailDto";

type ListDetailPriceRowSource = {
  modelId?: unknown;

  kind?: unknown;
  modelNumber?: unknown;

  displayOrder?: unknown;
  stock?: unknown;
  price?: unknown;

  size?: unknown;
  color?: unknown;
  rgb?: unknown;

  volumeValue?: unknown;
  volumeUnit?: unknown;
};

type ListDetailSource = Omit<
  Partial<ListDetailDTO>,
  "imageUrls" | "priceRows"
> & {
  imageUrls?: readonly unknown[] | null;
  priceRows?: readonly ListDetailPriceRowSource[] | null;
};

export type NormalizedListDetailPriceRow = {
  id: string;
  modelId: string;

  kind: string | null;
  modelNumber: string | null;

  displayOrder: number | null;
  stock: number;
  price: number | null;

  size: string | null;
  color: string | null;
  rgb: number | null;

  volumeValue: number | null;
  volumeUnit: string | null;
};

function dedupeUrlsKeepOrder(
  urls: readonly unknown[],
): string[] {
  const seen = new Set<string>();
  const result: string[] = [];

  for (const value of urls) {
    const url =
      typeof value === "string"
        ? value
        : "";

    if (!url || seen.has(url)) {
      continue;
    }

    seen.add(url);
    result.push(url);
  }

  return result;
}

function toInt(value: unknown): number {
  const numberValue = Number(value);

  if (!Number.isFinite(numberValue)) {
    return 0;
  }

  return Math.trunc(numberValue);
}

function toNumberOrNull(
  value: unknown,
): number | null {
  if (value === null || value === undefined) {
    return null;
  }

  const numberValue = Number(value);

  if (!Number.isFinite(numberValue)) {
    return null;
  }

  return numberValue;
}

function toDisplayOrderOrNull(
  value: unknown,
): number | null {
  const numberValue = toNumberOrNull(value);

  if (numberValue === null) {
    return null;
  }

  return Math.trunc(numberValue);
}

function toStringOrNull(
  value: unknown,
): string | null {
  if (value === null || value === undefined) {
    return null;
  }

  if (typeof value === "string") {
    return value || null;
  }

  return String(value) || null;
}

export function normalizeStatus(
  value: unknown,
): ListStatus | "" {
  return isValidListStatus(value) ? value : "";
}

export function formatYMDHM(
  value: unknown,
): string {
  if (typeof value !== "string") {
    return "";
  }

  return safeDateTimeLabelJa(value, "");
}

export function normalizeImageUrls(
  dto:
    | {
        imageUrls?: readonly unknown[] | null;
      }
    | null
    | undefined,
): string[] {
  return dedupeUrlsKeepOrder(
    dto?.imageUrls ?? [],
  );
}

export function normalizePriceRows<
  TRow extends object = NormalizedListDetailPriceRow,
>(
  dto:
    | {
        priceRows?:
          | readonly ListDetailPriceRowSource[]
          | null;
      }
    | null
    | undefined,
): TRow[] {
  const rows = dto?.priceRows ?? [];

  return rows.map((row, index) => {
    const modelId =
      typeof row.modelId === "string"
        ? row.modelId
        : "";

    const normalizedRow: NormalizedListDetailPriceRow = {
      id: modelId || String(index),
      modelId,

      kind: toStringOrNull(row.kind),
      modelNumber: toStringOrNull(
        row.modelNumber,
      ),

      displayOrder: toDisplayOrderOrNull(
        row.displayOrder,
      ),
      stock: toInt(row.stock),
      price: toNumberOrNull(row.price),

      size: toStringOrNull(row.size),
      color: toStringOrNull(row.color),
      rgb: toNumberOrNull(row.rgb),

      volumeValue: toNumberOrNull(
        row.volumeValue,
      ),
      volumeUnit: toStringOrNull(
        row.volumeUnit,
      ),
    };

    return normalizedRow as unknown as TRow;
  });
}

export function updatePriceRowPrice<
  TRow extends object,
>(
  rows: readonly TRow[] | null | undefined,
  index: number,
  price: number | null,
): TRow[] {
  const source = rows ?? [];

  return source.map((row, rowIndex) => {
    if (rowIndex !== index) {
      return row;
    }

    return {
      ...row,
      price,
    };
  });
}

export function deriveListDetail<
  TRow extends object = NormalizedListDetailPriceRow,
>(
  dto: ListDetailSource | null | undefined,
) {
  const listingTitle = dto?.title ?? "";
  const description = dto?.description ?? "";
  const status = normalizeStatus(dto?.status);

  const productBrandId =
    dto?.productBrandId ?? "";
  const productBrandName =
    dto?.productBrandName ?? "";
  const productName = dto?.productName ?? "";

  const tokenBrandId =
    dto?.tokenBrandId ?? "";
  const tokenBrandName =
    dto?.tokenBrandName ?? "";
  const tokenName = dto?.tokenName ?? "";

  const assigneeId = dto?.assigneeId ?? "";
  const assigneeName =
    dto?.assigneeName || "未設定";

  const createdByName =
    dto?.createdByName ?? "";
  const createdAt = formatYMDHM(
    dto?.createdAt,
  );

  const updatedByName =
    dto?.updatedByName ?? "";
  const updatedAt = formatYMDHM(
    dto?.updatedAt,
  );

  const imageUrls = normalizeImageUrls(dto);
  const priceRows =
    normalizePriceRows<TRow>(dto);

  return {
    listingTitle,
    description,
    status,

    productBrandId,
    productBrandName,
    productName,

    tokenBrandId,
    tokenBrandName,
    tokenName,

    imageUrls,
    priceRows,

    assigneeId,
    assigneeName,

    createdByName,
    createdAt,

    updatedByName,
    updatedAt,
  };
}

export function computeListDetailPageTitle(
  args: {
    listId?: string;
    listingTitle?: string;
  },
): string {
  const id = args.listId ?? "";
  const title =
    args.listingTitle || "出品詳細";

  return id
    ? `${title}（listId: ${id}）`
    : title;
}