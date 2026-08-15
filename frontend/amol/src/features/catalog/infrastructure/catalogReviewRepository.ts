// frontend/amol/src/features/catalog/infrastructure/catalogReviewRepository.ts

import { DEFAULT_REVIEW_PAGE, DEFAULT_REVIEW_PER_PAGE } from "../constants";
import type { ProductBlueprintReviewPage } from "../../shared/types/review";

export async function fetchCatalogReviews(
  apiBaseUrl: string,
  productBlueprintId: string,
): Promise<ProductBlueprintReviewPage> {
  const searchParams = new URLSearchParams({
    page: String(DEFAULT_REVIEW_PAGE),
    perPage: String(DEFAULT_REVIEW_PER_PAGE),
  });

  const response = await fetch(
    `${apiBaseUrl}/mall/catalog/product-blueprints/${encodeURIComponent(productBlueprintId)}/reviews?${searchParams.toString()}`,
    {
      method: "GET",
      headers: { Accept: "application/json" },
      credentials: "include",
    },
  );

  const contentType = response.headers.get("content-type") ?? "";
  if (!contentType.includes("application/json")) {
    throw new Error("レビュー一覧APIがJSON以外を返しました。");
  }

  const data = (await response.json()) as ProductBlueprintReviewPage;

  if (!response.ok) {
    throw new Error("レビュー一覧の取得に失敗しました。");
  }

  return data;
}