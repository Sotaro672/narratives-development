// frontend/amol/src/features/list/application/loadListPage.ts

import {
  fetchListCatalog,
  fetchMallLists,
} from "../infrastructure/listApi";

import type {
  LoadListPageResult,
  MallCatalogResponse,
  MallListCardItem,
  MallListItem,
} from "../../shared/types/list";

type LoadListPageArgs = {
  page: number;
  perPage: number;
};

function asOptionalString(value: unknown): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }

  const normalizedValue = value.trim();
  return normalizedValue || undefined;
}

function asOptionalNumber(value: unknown): number | undefined {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return undefined;
  }

  return value;
}

function mapListItemToCardItem(
  item: MallListItem,
  catalog: MallCatalogResponse | null,
): MallListCardItem {
  const productBlueprint = catalog?.productBlueprint;
  const productReviewSummary = catalog?.productReviewSummary;

  return {
    ...item,
    productName: asOptionalString(productBlueprint?.productName),
    brandName: asOptionalString(productBlueprint?.brandName),
    reviewAverage: asOptionalNumber(productReviewSummary?.averageRating),
    reviewCount: asOptionalNumber(productReviewSummary?.totalCount),
  };
}

async function attachCatalogToListItem(
  item: MallListItem,
): Promise<MallListCardItem> {
  try {
    const catalog = await fetchListCatalog(item.id);

    return mapListItemToCardItem(
      item,
      catalog,
    );
  } catch {
    return mapListItemToCardItem(
      item,
      null,
    );
  }
}

export async function loadListPage({
  page,
  perPage,
}: LoadListPageArgs): Promise<LoadListPageResult> {
  const response = await fetchMallLists({
    page,
    perPage,
  });

  const items = await Promise.all(
    response.items.map(attachCatalogToListItem),
  );

  return {
    items,
    page: response.page > 0 ? response.page : page,
    totalPages: response.totalPages > 0 ? response.totalPages : 1,
  };
}