// frontend/amol/src/features/market/marketApi.ts
import { fetchCurrentAvatarId } from "../cart/api/cartApi";
import { getApiBaseUrl } from "../../lib/apiBaseUrl";

export type MarketResaleStatus = "listing" | "suspended";

export type MarketResaleCondition =
  | "新品・未使用"
  | "未使用に近い"
  | "目立った傷や汚れなし"
  | "やや傷や汚れあり"
  | "傷や汚れあり";

export type MarketResaleListing = {
  id: string;
  status?: MarketResaleStatus;
  mintAddress?: string;
  tokenBlueprintId?: string;
  productId?: string;
  brandId?: string;
  productBlueprintId?: string;
  avatarId?: string;
  price?: number;
  condition?: MarketResaleCondition;
  description?: string;
  imageId?: string;
  imageUrl?: string;

  productName?: string;
  tokenName?: string;
  brandName?: string;

  createdBy?: string;
  createdAt?: string;
  updatedBy?: string | null;
  updatedAt?: string | null;
};

export type MarketResaleListResponse = {
  items: MarketResaleListing[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

export type MarketResaleCursorListResponse = {
  items: MarketResaleListing[];
  nextCursor: string | null;
  limit: number;
};

export type MarketResaleDetailResponse = {
  data: MarketResaleListing;
};

export type MarketResaleSortOrder = "asc" | "desc";

export type FetchMarketResalesParams = {
  page?: number;
  perPage?: number;
  q?: string;
  search?: string;
  searchQuery?: string;
  ids?: string[];
  mintAddresses?: string[];
  tokenBlueprintIds?: string[];
  productIds?: string[];
  brandIds?: string[];
  productBlueprintIds?: string[];

  /**
   * NOTE:
   * backend の MarketQuery では avatarIds は「閲覧者 avatarId の除外用」として扱う。
   * 新規呼び出しでは viewerAvatarId / viewerAvatarIds を優先する。
   */
  avatarIds?: string[];
  avatarId?: string;
  viewerAvatarId?: string;
  viewerAvatarIds?: string[];

  status?: MarketResaleStatus;
  statuses?: MarketResaleStatus[];
  condition?: MarketResaleCondition;
  conditions?: MarketResaleCondition[];
  minPrice?: number;
  maxPrice?: number;
  sort?: string;
  sortBy?: string;
  orderBy?: string;
  order?: MarketResaleSortOrder;
  sortOrder?: MarketResaleSortOrder;
  direction?: MarketResaleSortOrder;
};

export type FetchMarketResalesByCursorParams = Omit<
  FetchMarketResalesParams,
  "page" | "perPage"
> & {
  after?: string;
  cursor?: string;
  limit?: number;
};

const MARKET_RESALES_PATH = "/mall/market/resales";

function normalizeApiBaseUrl(): string {
  const baseUrl = getApiBaseUrl();

  if (typeof baseUrl === "string" && baseUrl.trim() !== "") {
    return baseUrl.replace(/\/$/, "");
  }

  return "";
}

function appendString(
  searchParams: URLSearchParams,
  key: string,
  value: unknown,
) {
  if (typeof value !== "string") {
    return;
  }

  const trimmed = value.trim();

  if (trimmed === "") {
    return;
  }

  searchParams.set(key, trimmed);
}

function appendNumber(
  searchParams: URLSearchParams,
  key: string,
  value: unknown,
) {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return;
  }

  searchParams.set(key, String(value));
}

function appendStringList(
  searchParams: URLSearchParams,
  key: string,
  values: unknown,
) {
  if (!Array.isArray(values)) {
    return;
  }

  const cleaned = values
    .filter((value): value is string => typeof value === "string")
    .map((value) => value.trim())
    .filter(Boolean);

  if (cleaned.length === 0) {
    return;
  }

  searchParams.set(key, cleaned.join(","));
}

function buildMarketResaleSearchParams(
  params: FetchMarketResalesParams | FetchMarketResalesByCursorParams = {},
): URLSearchParams {
  const searchParams = new URLSearchParams();

  appendNumber(searchParams, "page", "page" in params ? params.page : undefined);
  appendNumber(
    searchParams,
    "perPage",
    "perPage" in params ? params.perPage : undefined,
  );

  appendString(searchParams, "q", params.q);
  appendString(searchParams, "search", params.search);
  appendString(searchParams, "searchQuery", params.searchQuery);

  appendStringList(searchParams, "ids", params.ids);
  appendStringList(searchParams, "mintAddresses", params.mintAddresses);
  appendStringList(searchParams, "tokenBlueprintIds", params.tokenBlueprintIds);
  appendStringList(searchParams, "productIds", params.productIds);
  appendStringList(searchParams, "brandIds", params.brandIds);
  appendStringList(
    searchParams,
    "productBlueprintIds",
    params.productBlueprintIds,
  );

  appendStringList(searchParams, "avatarIds", params.avatarIds);
  appendString(searchParams, "avatarId", params.avatarId);
  appendString(searchParams, "viewerAvatarId", params.viewerAvatarId);
  appendStringList(searchParams, "viewerAvatarIds", params.viewerAvatarIds);

  appendString(searchParams, "status", params.status);
  appendStringList(searchParams, "statuses", params.statuses);

  appendString(searchParams, "condition", params.condition);
  appendStringList(searchParams, "conditions", params.conditions);

  appendNumber(searchParams, "minPrice", params.minPrice);
  appendNumber(searchParams, "maxPrice", params.maxPrice);

  appendString(searchParams, "sort", params.sort);
  appendString(searchParams, "sortBy", params.sortBy);
  appendString(searchParams, "orderBy", params.orderBy);
  appendString(searchParams, "order", params.order);
  appendString(searchParams, "sortOrder", params.sortOrder);
  appendString(searchParams, "direction", params.direction);

  if ("after" in params) {
    appendString(searchParams, "after", params.after);
  }

  if ("cursor" in params) {
    appendString(searchParams, "cursor", params.cursor);
  }

  if ("limit" in params) {
    appendNumber(searchParams, "limit", params.limit);
  }

  return searchParams;
}

function getErrorMessage(status: number): string {
  if (status === 400) {
    return "マーケット一覧の取得条件が不正です。";
  }

  if (status === 401) {
    return "ログインが必要です。";
  }

  if (status === 403) {
    return "マーケット情報を取得する権限がありません。";
  }

  if (status === 404) {
    return "マーケット情報が見つかりません。";
  }

  if (status >= 500) {
    return "サーバー側でエラーが発生しました。";
  }

  return "マーケット情報の取得に失敗しました。";
}

async function readJsonResponse<T>(
  response: Response,
  fallbackMessage: string,
): Promise<T> {
  const contentType = response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    throw new Error(fallbackMessage);
  }

  const data = (await response.json()) as T;

  if (!response.ok) {
    throw new Error(getErrorMessage(response.status));
  }

  return data;
}

async function resolveViewerAvatarId(
  apiBaseUrl: string,
  params: FetchMarketResalesParams | FetchMarketResalesByCursorParams,
): Promise<string> {
  const explicitViewerAvatarId = normalizeString(params.viewerAvatarId);
  if (explicitViewerAvatarId) {
    return explicitViewerAvatarId;
  }

  const firstViewerAvatarId = firstNonEmptyString(params.viewerAvatarIds);
  if (firstViewerAvatarId) {
    return firstViewerAvatarId;
  }

  const explicitAvatarId = normalizeString(params.avatarId);
  if (explicitAvatarId) {
    return explicitAvatarId;
  }

  const firstAvatarId = firstNonEmptyString(params.avatarIds);
  if (firstAvatarId) {
    return firstAvatarId;
  }

  try {
    return await fetchCurrentAvatarId(apiBaseUrl);
  } catch {
    return "";
  }
}

async function withResolvedViewerAvatarId<
  T extends FetchMarketResalesParams | FetchMarketResalesByCursorParams,
>(apiBaseUrl: string, params: T): Promise<T> {
  const viewerAvatarId = await resolveViewerAvatarId(apiBaseUrl, params);

  if (!viewerAvatarId) {
    return params;
  }

  return {
    ...params,
    viewerAvatarId,
  };
}

function normalizeString(value: unknown): string {
  if (typeof value !== "string") {
    return "";
  }

  return value.trim();
}

function firstNonEmptyString(values: unknown): string {
  if (!Array.isArray(values)) {
    return "";
  }

  for (const value of values) {
    const normalized = normalizeString(value);
    if (normalized) {
      return normalized;
    }
  }

  return "";
}

export async function fetchMarketResales(
  params: FetchMarketResalesParams = {},
): Promise<MarketResaleListResponse> {
  const apiBaseUrl = normalizeApiBaseUrl();

  if (!apiBaseUrl) {
    throw new Error("API Base URLが未設定です。");
  }

  const resolvedParams = await withResolvedViewerAvatarId(apiBaseUrl, params);
  const searchParams = buildMarketResaleSearchParams(resolvedParams);
  const query = searchParams.toString();
  const url = `${apiBaseUrl}${MARKET_RESALES_PATH}${query ? `?${query}` : ""}`;

  const response = await fetch(url, {
    method: "GET",
    headers: {
      Accept: "application/json",
    },
    credentials: "include",
  });

  return readJsonResponse<MarketResaleListResponse>(
    response,
    "マーケット一覧APIがJSON以外を返しました。",
  );
}

export async function fetchMarketResalesByCursor(
  params: FetchMarketResalesByCursorParams = {},
): Promise<MarketResaleCursorListResponse> {
  const apiBaseUrl = normalizeApiBaseUrl();

  if (!apiBaseUrl) {
    throw new Error("API Base URLが未設定です。");
  }

  const resolvedParams = await withResolvedViewerAvatarId(apiBaseUrl, params);
  const searchParams = buildMarketResaleSearchParams(resolvedParams);
  searchParams.set("mode", "cursor");

  const query = searchParams.toString();
  const url = `${apiBaseUrl}${MARKET_RESALES_PATH}${query ? `?${query}` : ""}`;

  const response = await fetch(url, {
    method: "GET",
    headers: {
      Accept: "application/json",
    },
    credentials: "include",
  });

  return readJsonResponse<MarketResaleCursorListResponse>(
    response,
    "マーケット一覧APIがJSON以外を返しました。",
  );
}

export async function fetchMarketResalesCursorEndpoint(
  params: FetchMarketResalesByCursorParams = {},
): Promise<MarketResaleCursorListResponse> {
  const apiBaseUrl = normalizeApiBaseUrl();

  if (!apiBaseUrl) {
    throw new Error("API Base URLが未設定です。");
  }

  const resolvedParams = await withResolvedViewerAvatarId(apiBaseUrl, params);
  const searchParams = buildMarketResaleSearchParams(resolvedParams);
  const query = searchParams.toString();
  const url = `${apiBaseUrl}${MARKET_RESALES_PATH}/cursor${
    query ? `?${query}` : ""
  }`;

  const response = await fetch(url, {
    method: "GET",
    headers: {
      Accept: "application/json",
    },
    credentials: "include",
  });

  return readJsonResponse<MarketResaleCursorListResponse>(
    response,
    "マーケット一覧APIがJSON以外を返しました。",
  );
}

export async function fetchMarketResaleById(
  resaleId: string,
): Promise<MarketResaleListing> {
  const apiBaseUrl = normalizeApiBaseUrl();
  const id = resaleId.trim();

  if (!apiBaseUrl) {
    throw new Error("API Base URLが未設定です。");
  }

  if (!id) {
    throw new Error("マーケット出品IDが未指定です。");
  }

  const response = await fetch(
    `${apiBaseUrl}${MARKET_RESALES_PATH}/${encodeURIComponent(id)}`,
    {
      method: "GET",
      headers: {
        Accept: "application/json",
      },
      credentials: "include",
    },
  );

  const result = await readJsonResponse<MarketResaleDetailResponse>(
    response,
    "マーケット詳細APIがJSON以外を返しました。",
  );

  return result.data;
}