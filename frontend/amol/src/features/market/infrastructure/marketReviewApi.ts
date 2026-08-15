// frontend/amol/src/features/market/api/marketReviewApi.ts

import { requestJson } from "../../../lib/http";
import { MARKET_CATALOG_PRODUCT_BLUEPRINTS_PATH } from "../constants/marketPaths";
import type { ProductBlueprintReviewPage } from "../../shared/types/review";

export async function fetchMarketProductBlueprintReviews(args: {
  productBlueprintId: string;
  page?: number;
  perPage?: number;
}): Promise<ProductBlueprintReviewPage> {
  const productBlueprintId = args.productBlueprintId.trim();
  const page = args.page ?? 1;
  const perPage = args.perPage ?? 20;

  if (!productBlueprintId) {
    throw new Error("商品IDが未指定です。");
  }

  return requestJson<ProductBlueprintReviewPage>(
    `${MARKET_CATALOG_PRODUCT_BLUEPRINTS_PATH}/${encodeURIComponent(productBlueprintId)}/reviews`,
    {
      method: "GET",
      auth: "none",
      query: {
        page: String(page),
        perPage: String(perPage),
      },
      credentials: "include",
      messages: {
        requestErrorMessage: "レビューの取得に失敗しました。",
        nonJsonErrorMessage: "商品レビューAPIがJSON以外を返しました。",
      },
    },
  );
}