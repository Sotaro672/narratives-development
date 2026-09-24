// frontend/mall/src/features/market/infrastructure/marketResaleReviewContextApi.ts

import { requestJson } from "../../../lib/http";
import { MARKET_RESALES_PATH } from "../constants/marketPaths";

import type { MarketResaleListing } from "../../shared/types/marketResale";
import type { ResaleConditionImage } from "../../shared/types/resale";

export type MarketResaleReviewContext = {
  data: MarketResaleListing;
  images: ResaleConditionImage[];
};

function requireResaleId(resaleId: string): string {
  const normalizedResaleId = resaleId.trim();

  if (!normalizedResaleId) {
    throw new Error("マーケット出品IDが未指定です。");
  }

  return normalizedResaleId;
}

function buildMarketResaleReviewContextPath(
  resaleId: string,
): string {
  const normalizedResaleId = requireResaleId(resaleId);

  return `${MARKET_RESALES_PATH}/${encodeURIComponent(normalizedResaleId)}/review-context`;
}

export async function fetchMarketResaleReviewContext(
  resaleId: string,
): Promise<MarketResaleReviewContext> {
  return requestJson<MarketResaleReviewContext>(
    buildMarketResaleReviewContextPath(resaleId),
    {
      method: "GET",
      auth: "required",
      credentials: "include",
      messages: {
        requestErrorMessage:
          "再販レビュー情報の取得に失敗しました。",
        nonJsonErrorMessage:
          "再販レビュー情報APIがJSON以外を返しました。",
        invalidJsonErrorMessage:
          "再販レビュー情報APIのレスポンスが不正です。",
      },
    },
  );
}