// frontend/amol/src/features/market/api/marketResaleImageApi.ts

import {
  MARKET_RESALES_PATH,
} from "../constants/marketPaths";

import {
  getMarketApiBaseUrl,
  readMarketJsonResponse,
} from "../infrastructure/marketHttpClient";

import {
  normalizeMarketResaleConditionImagesResponse,
} from "../infrastructure/marketResaleImageMapper";

import type {
  MarketResaleConditionImage,
  MarketResaleConditionImagesResponse,
} from "../types/marketResaleImage";

export async function fetchMarketResaleConditionImages(
  resaleId: string,
): Promise<MarketResaleConditionImage[]> {
  const apiBaseUrl =
    getMarketApiBaseUrl();

  const normalizedResaleId =
    resaleId.trim();

  if (!normalizedResaleId) {
    throw new Error(
      "マーケット出品IDが未指定です。",
    );
  }

  const url =
    `${apiBaseUrl}${MARKET_RESALES_PATH}/${encodeURIComponent(
      normalizedResaleId,
    )}/images`;

  const response = await fetch(
    url,
    {
      method: "GET",
      headers: {
        Accept: "application/json",
      },
      credentials: "include",
    },
  );

  const result =
    await readMarketJsonResponse<MarketResaleConditionImagesResponse>(
      response,
      "マーケット出品画像APIがJSON以外を返しました。",
    );

  return normalizeMarketResaleConditionImagesResponse(
    result,
  );
}