// frontend/console/shell/src/features/list/application/listDetail/listDetailMapper.ts

import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";
import type { ListDetailDTO } from "../../infrastructure/dto/listDetailDto";
import type { ListDetailPriceRowDTO } from "../../infrastructure/dto/listPriceRowDto";

export type ListDetailPriceRowVM = Omit<
  ListDetailPriceRowDTO,
  "price"
> & {
  price?: number;
};

export type ListDetailVM = Omit<
  ListDetailDTO,
  "priceRows"
> & {
  priceRows: ListDetailPriceRowVM[];
  createdAtLabel: string;
  updatedAtLabel: string;
};

function buildPriceRows(
  rows: readonly ListDetailPriceRowDTO[],
): ListDetailPriceRowVM[] {
  return rows.map(({ price, ...row }) => ({
    ...row,
    ...(price == null ? {} : { price }),
  }));
}

export function updatePriceRowPrice<TRow extends object>(
  rows: readonly TRow[],
  index: number,
  price: number | undefined,
): TRow[] {
  return rows.map((row, rowIndex) => {
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

export function deriveListDetail(
  dto: ListDetailDTO,
): ListDetailVM {
  return {
    ...dto,
    priceRows: buildPriceRows(dto.priceRows),
    createdAtLabel: safeDateTimeLabelJa(dto.createdAt, ""),
    updatedAtLabel: dto.updatedAt
      ? safeDateTimeLabelJa(dto.updatedAt, "")
      : "",
  };
}

export function computeListDetailPageTitle(args: {
  listId?: string;
  title?: string;
}): string {
  const title = args.title || "出品詳細";

  return args.listId
    ? `${title}（listId: ${args.listId}）`
    : title;
}