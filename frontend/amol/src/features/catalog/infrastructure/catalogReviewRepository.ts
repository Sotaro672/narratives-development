// frontend/amol/src/features/catalog/infrastructure/catalogReviewRepository.ts

import {
  DEFAULT_REVIEW_PAGE,
  DEFAULT_REVIEW_PER_PAGE,
} from "../constants";
import type { CatalogProductBlueprintReviewPage } from "../types";

export async function fetchCatalogReviews(
  apiBaseUrl: string,
  productBlueprintId: string,
): Promise<CatalogProductBlueprintReviewPage> {
  const searchParams = new URLSearchParams({
    page: String(DEFAULT_REVIEW_PAGE),
    perPage: String(DEFAULT_REVIEW_PER_PAGE),
  });

  const response = await fetch(
    `${apiBaseUrl}/mall/catalog/product-blueprints/${encodeURIComponent(
      productBlueprintId,
    )}/reviews?${searchParams.toString()}`,
    {
      method: "GET",
      headers: {
        Accept: "application/json",
      },
      credentials: "include",
    },
  );

  const contentType = response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    throw new Error("レビュー一覧APIがJSON以外を返しました。");
  }

  const data =
    (await response.json()) as Partial<CatalogProductBlueprintReviewPage>;

  if (!response.ok) {
    throw new Error("レビュー一覧の取得に失敗しました。");
  }

  return {
    items: Array.isArray(data.items) ? data.items : [],
    page:
      typeof data.page === "number" && data.page > 0
        ? data.page
        : DEFAULT_REVIEW_PAGE,
    perPage:
      typeof data.perPage === "number" && data.perPage > 0
        ? data.perPage
        : DEFAULT_REVIEW_PER_PAGE,
    total: typeof data.total === "number" && data.total > 0 ? data.total : 0,
    hasNext: Boolean(data.hasNext),
  };
}