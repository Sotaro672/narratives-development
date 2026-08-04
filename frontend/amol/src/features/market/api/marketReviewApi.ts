// frontend/amol/src/features/market/api/marketReviewApi.ts

import {
  isFiniteNumber,
} from "../../../components/utils/typeGuards";

import {
  MARKET_CATALOG_PRODUCT_BLUEPRINTS_PATH,
} from "../constants/marketPaths";

import {
  getMarketApiBaseUrl,
  readMarketJsonResponse,
} from "../infrastructure/marketHttpClient";

import {
  normalizeMarketProductBlueprintReviewsResponse,
} from "../infrastructure/marketReviewMapper";

import type {
  ProductBlueprintReviewPage,
} from "../../shared/types/review";

import type {
  MarketProductBlueprintReviewsResponse,
} from "../../shared/types/marketReview";

export async function fetchMarketProductBlueprintReviews(
  args: {
    productBlueprintId: string;
    page?: number;
    perPage?: number;
  },
): Promise<ProductBlueprintReviewPage> {
  const apiBaseUrl =
    getMarketApiBaseUrl();

  const productBlueprintId =
    args.productBlueprintId.trim();

  const page =
    isFiniteNumber(args.page) &&
    args.page > 0
      ? args.page
      : 1;

  const perPage =
    isFiniteNumber(args.perPage) &&
    args.perPage > 0
      ? args.perPage
      : 20;

  if (!productBlueprintId) {
    throw new Error(
      "商品IDが未指定です。",
    );
  }

  const url = new URL(
    `${apiBaseUrl}${MARKET_CATALOG_PRODUCT_BLUEPRINTS_PATH}/${encodeURIComponent(
      productBlueprintId,
    )}/reviews`,
  );

  url.searchParams.set(
    "page",
    String(page),
  );

  url.searchParams.set(
    "perPage",
    String(perPage),
  );

  const response = await fetch(
    url.toString(),
    {
      method: "GET",
      headers: {
        Accept: "application/json",
      },
      credentials: "include",
    },
  );

  const result =
    await readMarketJsonResponse<MarketProductBlueprintReviewsResponse>(
      response,
      "商品レビューAPIがJSON以外を返しました。",
    );

  return normalizeMarketProductBlueprintReviewsResponse(
    result,
    page,
    perPage,
  );
}