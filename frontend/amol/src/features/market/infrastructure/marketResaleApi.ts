// frontend/amol/src/features/market/api/marketResaleApi.ts

import { requestJson } from "../../../lib/http";
import { MARKET_RESALES_PATH } from "../constants/marketPaths";
import { buildMarketResaleSearchParams } from "../application/marketResaleQueryBuilder";

import type {
  FetchMarketResalesParams,
  MarketResaleDetailResponse,
  MarketResaleListing,
  MarketResaleListResponse,
} from "../../shared/types/marketResale";

export async function fetchMarketResales(
  params: FetchMarketResalesParams = {},
): Promise<MarketResaleListResponse> {
  const searchParams = buildMarketResaleSearchParams(params);

  return requestJson<MarketResaleListResponse>(MARKET_RESALES_PATH, {
    method: "GET",
    auth: "required",
    query: Object.fromEntries(searchParams.entries()),
    credentials: "include",
    messages: {
      requestErrorMessage: "マーケット情報の取得に失敗しました。",
      nonJsonErrorMessage: "マーケット一覧APIがJSON以外を返しました。",
    },
  });
}

export async function fetchMarketResaleById(
  resaleId: string,
): Promise<MarketResaleListing> {
  const normalizedResaleId = resaleId.trim();

  if (!normalizedResaleId) {
    throw new Error("マーケット出品IDが未指定です。");
  }

  const result = await requestJson<MarketResaleDetailResponse>(
    `${MARKET_RESALES_PATH}/${encodeURIComponent(normalizedResaleId)}`,
    {
      method: "GET",
      auth: "required",
      credentials: "include",
      messages: {
        requestErrorMessage: "マーケット情報の取得に失敗しました。",
        nonJsonErrorMessage: "マーケット詳細APIがJSON以外を返しました。",
      },
    },
  );

  return result.data;
}