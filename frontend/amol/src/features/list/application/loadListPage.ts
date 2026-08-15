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

function asOptionalString(
  value: unknown,
): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }

  const normalizedValue =
    value.trim();

  return normalizedValue ||
    undefined;
}

function mapListItemToCardItem(
  item: MallListItem,
  catalog: MallCatalogResponse | null,
): MallListCardItem {
  const productBlueprint =
    catalog?.productBlueprint;

  return {
    ...item,
    productName:
      asOptionalString(
        productBlueprint?.productName,
      ),
    brandName:
      asOptionalString(
        productBlueprint?.brandName,
      ),
  };
}

async function attachCatalogToListItem(
  item: MallListItem,
): Promise<MallListCardItem> {
  try {
    const catalog =
      await fetchListCatalog(
        item.id,
      );

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
  const response =
    await fetchMallLists({
      page,
      perPage,
    });

  const items =
    await Promise.all(
      response.items.map(
        attachCatalogToListItem,
      ),
    );

  return {
    items,
    page:
      response.page > 0
        ? response.page
        : page,
    totalPages:
      response.totalPages > 0
        ? response.totalPages
        : 1,
  };
}