// frontend/amol/src/features/market/infrastructure/marketResaleQueryBuilder.ts

import {
  isFiniteNumber,
} from "../../../components/utils/typeGuards";

import type {
  FetchMarketResalesParams,
} from "../../shared/types/marketResale";

function appendString(
  searchParams: URLSearchParams,
  key: string,
  value: unknown,
): void {
  if (typeof value !== "string") {
    return;
  }

  const trimmed = value.trim();

  if (!trimmed) {
    return;
  }

  searchParams.set(
    key,
    trimmed,
  );
}

function appendNumber(
  searchParams: URLSearchParams,
  key: string,
  value: unknown,
): void {
  if (!isFiniteNumber(value)) {
    return;
  }

  searchParams.set(
    key,
    String(value),
  );
}

function appendStringList(
  searchParams: URLSearchParams,
  key: string,
  values: unknown,
): void {
  if (!Array.isArray(values)) {
    return;
  }

  const cleaned = values
    .filter(
      (value): value is string =>
        typeof value === "string",
    )
    .map((value) => value.trim())
    .filter(Boolean);

  if (cleaned.length === 0) {
    return;
  }

  searchParams.set(
    key,
    cleaned.join(","),
  );
}

export function buildMarketResaleSearchParams(
  params: FetchMarketResalesParams = {},
): URLSearchParams {
  const searchParams =
    new URLSearchParams();

  appendNumber(
    searchParams,
    "page",
    params.page,
  );

  appendNumber(
    searchParams,
    "perPage",
    params.perPage,
  );

  appendString(
    searchParams,
    "q",
    params.q,
  );

  appendString(
    searchParams,
    "search",
    params.search,
  );

  appendString(
    searchParams,
    "searchQuery",
    params.searchQuery,
  );

  appendStringList(
    searchParams,
    "ids",
    params.ids,
  );

  appendStringList(
    searchParams,
    "mintAddresses",
    params.mintAddresses,
  );

  appendStringList(
    searchParams,
    "tokenBlueprintIds",
    params.tokenBlueprintIds,
  );

  appendStringList(
    searchParams,
    "productIds",
    params.productIds,
  );

  appendStringList(
    searchParams,
    "brandIds",
    params.brandIds,
  );

  appendStringList(
    searchParams,
    "productBlueprintIds",
    params.productBlueprintIds,
  );

  appendStringList(
    searchParams,
    "avatarIds",
    params.avatarIds,
  );

  appendString(
    searchParams,
    "avatarId",
    params.avatarId,
  );

  appendString(
    searchParams,
    "viewerAvatarId",
    params.viewerAvatarId,
  );

  appendStringList(
    searchParams,
    "viewerAvatarIds",
    params.viewerAvatarIds,
  );

  appendString(
    searchParams,
    "status",
    params.status,
  );

  appendStringList(
    searchParams,
    "statuses",
    params.statuses,
  );

  appendString(
    searchParams,
    "condition",
    params.condition,
  );

  appendStringList(
    searchParams,
    "conditions",
    params.conditions,
  );

  appendNumber(
    searchParams,
    "minPrice",
    params.minPrice,
  );

  appendNumber(
    searchParams,
    "maxPrice",
    params.maxPrice,
  );

  appendString(
    searchParams,
    "sort",
    params.sort,
  );

  appendString(
    searchParams,
    "sortBy",
    params.sortBy,
  );

  appendString(
    searchParams,
    "orderBy",
    params.orderBy,
  );

  appendString(
    searchParams,
    "order",
    params.order,
  );

  appendString(
    searchParams,
    "sortOrder",
    params.sortOrder,
  );

  appendString(
    searchParams,
    "direction",
    params.direction,
  );

  return searchParams;
}