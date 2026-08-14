// frontend/console/shell/src/features/list/application/listDetail/listDetailMapper.ts

import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";
import type { ListDetailDTO } from "../../infrastructure/dto/listDetailDto";

export type ListDetailPriceRowVM = {
  modelId: string;
  kind: string;
  modelNumber: string;
  displayOrder?: number | null;
  stock: number;
  price?: number;
  size?: string;
  color?: string;
  rgb?: number | null;
  volumeValue?: number | null;
  volumeUnit?: string;
};

function formatYMDHM(value: string | undefined): string {
  return value ? safeDateTimeLabelJa(value, "") : "";
}

function buildPriceRows(dto: ListDetailDTO): ListDetailPriceRowVM[] {
  return dto.priceRows.map((row) => ({
    modelId: row.modelId,
    kind: row.kind,
    modelNumber: row.modelNumber,
    displayOrder: row.displayOrder,
    stock: row.stock,
    ...(row.price == null ? {} : { price: row.price }),
    ...(row.size === undefined ? {} : { size: row.size }),
    ...(row.color === undefined ? {} : { color: row.color }),
    ...(row.rgb === undefined ? {} : { rgb: row.rgb }),
    ...(row.volumeValue === undefined ? {} : { volumeValue: row.volumeValue }),
    ...(row.volumeUnit === undefined ? {} : { volumeUnit: row.volumeUnit }),
  }));
}

export function updatePriceRowPrice<TRow extends object>(
  rows: readonly TRow[] | null | undefined,
  index: number,
  price: number | undefined,
): TRow[] {
  const source = rows ?? [];

  return source.map((row, rowIndex) => {
    if (rowIndex !== index) {
      return row;
    }

    if (price === undefined) {
      const nextRow = { ...row } as TRow & { price?: number };
      delete nextRow.price;
      return nextRow;
    }

    return { ...row, price } as TRow;
  });
}

export function deriveListDetail<TRow extends object = ListDetailPriceRowVM>(
  dto: ListDetailDTO | null | undefined,
) {
  if (!dto) {
    return {
      listingTitle: "",
      description: "",
      status: "" as const,
      productBrandId: "",
      productBrandName: "",
      productName: "",
      tokenBrandId: "",
      tokenBrandName: "",
      tokenName: "",
      images: [],
      imageUrls: [],
      primaryImageId: "",
      priceRows: [] as TRow[],
      assigneeId: "",
      assigneeName: "",
      createdByName: "",
      createdAt: "",
      updatedByName: "",
      updatedAt: "",
    };
  }

  const images = dto.images;
  const priceRows = buildPriceRows(dto) as TRow[];

  return {
    listingTitle: dto.title,
    description: dto.description,
    status: dto.status,
    productBrandId: dto.productBrandId,
    productBrandName: dto.productBrandName,
    productName: dto.productName,
    tokenBrandId: dto.tokenBrandId,
    tokenBrandName: dto.tokenBrandName,
    tokenName: dto.tokenName,
    images,
    imageUrls: images.map((image) => image.url),
    primaryImageId: dto.primaryImageId ?? "",
    priceRows,
    assigneeId: dto.assigneeId,
    assigneeName: dto.assigneeName,
    createdByName: dto.createdByName,
    createdAt: formatYMDHM(dto.createdAt),
    updatedByName: dto.updatedByName ?? "",
    updatedAt: formatYMDHM(dto.updatedAt),
  };
}

export function computeListDetailPageTitle(args: {
  listId?: string;
  listingTitle?: string;
}): string {
  const id = args.listId ?? "";
  const title = args.listingTitle || "出品詳細";
  return id ? `${title}（listId: ${id}）` : title;
}