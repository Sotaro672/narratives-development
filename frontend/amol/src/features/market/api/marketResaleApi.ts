// frontend/amol/src/features/market/api/marketResaleApi.ts

import {
  MARKET_RESALES_PATH,
} from "../constants/marketPaths";

import {
  getMarketApiBaseUrl,
  readMarketJsonResponse,
} from "../infrastructure/marketHttpClient";

import {
  buildMarketResaleSearchParams,
} from "../infrastructure/marketResaleQueryBuilder";

import type {
  FetchMarketResalesParams,
  MarketResaleDetailResponse,
  MarketResaleListing,
  MarketResaleListResponse,
} from "../../shared/types/marketResale";

export async function fetchMarketResales(
  params: FetchMarketResalesParams = {},
): Promise<MarketResaleListResponse> {
  const apiBaseUrl =
    getMarketApiBaseUrl();

  const searchParams =
    buildMarketResaleSearchParams(
      params,
    );

  const query =
    searchParams.toString();

  const url =
    `${apiBaseUrl}${MARKET_RESALES_PATH}` +
    `${query ? `?${query}` : ""}`;

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

  return readMarketJsonResponse<MarketResaleListResponse>(
    response,
    "マーケット一覧APIがJSON以外を返しました。",
  );
}

export async function fetchMarketResaleById(
  resaleId: string,
): Promise<MarketResaleListing> {
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
    )}`;

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
    await readMarketJsonResponse<MarketResaleDetailResponse>(
      response,
      "マーケット詳細APIがJSON以外を返しました。",
    );

  return result.data;
}