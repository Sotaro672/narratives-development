// frontend/amol/src/features/market/api/marketResaleImageApi.ts

import { requestJson } from "../../../lib/http";
import { MARKET_RESALES_PATH } from "../constants/marketPaths";
import type { ResaleConditionImage } from "../../shared/types/resale";

type MarketResaleImagesResponse = {
  items: ResaleConditionImage[];
};

export async function fetchMarketResaleConditionImages(
  resaleId: string,
): Promise<ResaleConditionImage[]> {
  const normalizedResaleId = resaleId.trim();

  if (!normalizedResaleId) {
    throw new Error("マーケット出品IDが未指定です。");
  }

  const result = await requestJson<MarketResaleImagesResponse>(
    `${MARKET_RESALES_PATH}/${encodeURIComponent(normalizedResaleId)}/images`,
    {
      method: "GET",
      auth: "none",
      credentials: "include",
      messages: {
        requestErrorMessage: "マーケット出品画像の取得に失敗しました。",
        nonJsonErrorMessage: "マーケット出品画像APIがJSON以外を返しました。",
      },
    },
  );

  return result.items;
}