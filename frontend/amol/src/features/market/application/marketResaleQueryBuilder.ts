// frontend/amol/src/features/market/infrastructure/marketResaleQueryBuilder.ts

import type { FetchMarketResalesParams } from "../../shared/types/marketResale";

function appendString(
  searchParams: URLSearchParams,
  key: string,
  value?: string,
): void {
  if (!value) {
    return;
  }

  searchParams.set(key, value);
}

function appendNumber(
  searchParams: URLSearchParams,
  key: string,
  value?: number,
): void {
  if (value === undefined) {
    return;
  }

  searchParams.set(key, String(value));
}

function appendStringList(
  searchParams: URLSearchParams,
  key: string,
  values?: readonly string[],
): void {
  if (!values || values.length === 0) {
    return;
  }

  searchParams.set(key, values.join(","));
}

export function buildMarketResaleSearchParams(
  params: FetchMarketResalesParams = {},
): URLSearchParams {
  const searchParams = new URLSearchParams();

  appendNumber(searchParams, "page", params.page);
  appendNumber(searchParams, "perPage", params.perPage);
  appendString(searchParams, "q", params.q);
  appendStringList(searchParams, "ids", params.ids);
  appendStringList(searchParams, "assetIds", params.assetIds);
  appendStringList(
    searchParams,
    "tokenBlueprintIds",
    params.tokenBlueprintIds,
  );
  appendStringList(searchParams, "productIds", params.productIds);
  appendStringList(searchParams, "brandIds", params.brandIds);
  appendStringList(
    searchParams,
    "productBlueprintIds",
    params.productBlueprintIds,
  );
  appendStringList(searchParams, "avatarIds", params.avatarIds);
  appendStringList(searchParams, "conditions", params.conditions);
  appendNumber(searchParams, "minPrice", params.minPrice);
  appendNumber(searchParams, "maxPrice", params.maxPrice);
  appendString(searchParams, "sort", params.sort);
  appendString(searchParams, "order", params.order);

  return searchParams;
}