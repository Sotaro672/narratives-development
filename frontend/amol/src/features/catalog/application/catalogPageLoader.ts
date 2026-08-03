// frontend/amol/src/features/catalog/application/catalogPageLoader.ts

import {
  fetchCatalogDetail,
} from "../infrastructure/catalogRepository";

import {
  fetchCatalogReviews,
} from "../infrastructure/catalogReviewRepository";

import type {
  CatalogResponse,
} from "../../shared/types/catalog";

import type {
  ProductBlueprintReviewPage,
} from "../../shared/types/review";

export type LoadCatalogPageResult = {
  catalog: CatalogResponse;
  reviews: ProductBlueprintReviewPage | null;
  reviewErrorMessage: string;
};

export async function loadCatalogPage(args: {
  apiBaseUrl: string;
  listId: string;
}): Promise<LoadCatalogPageResult> {
  const catalog = await fetchCatalogDetail({
    apiBaseUrl: args.apiBaseUrl,
    listId: args.listId,
  });

  try {
    const reviews = await fetchCatalogReviews(
      args.apiBaseUrl,
      catalog.productBlueprint.id,
    );

    return {
      catalog,
      reviews,
      reviewErrorMessage: "",
    };
  } catch (error) {
    return {
      catalog,
      reviews: null,
      reviewErrorMessage:
        error instanceof Error
          ? error.message
          : "レビュー一覧の取得中にエラーが発生しました。",
    };
  }
}