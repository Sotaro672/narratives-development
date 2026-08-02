// frontend/amol/src/features/market/marketApi.ts

import { getApiBaseUrl } from "../../lib/apiBaseUrl";
import {
  isFiniteNumber,
  isRecord,
} from "../../components/utils/typeGuards";

export type MarketResaleStatus = "listing" | "suspended";

export type MarketResaleCondition =
  | "新品・未使用"
  | "未使用に近い"
  | "目立った傷や汚れなし"
  | "やや傷や汚れあり"
  | "傷や汚れあり";

export type MarketResaleConditionImage = {
  id: string;
  resaleId?: string;
  url: string;
  objectPath?: string;
  fileName?: string;
  fileSize?: number;
  mimeType?: string;
  type?: string;
  displayOrder?: number;
  createdAt?: string;
  createdBy?: string;
  updatedAt?: string | null;
  updatedBy?: string | null;
};

export type MarketProductBlueprintReview = {
  id: string;
  productBlueprintId?: string;
  avatarId?: string;
  avatarName?: string;
  avatarIcon?: string;
  rating?: number;
  title?: string;
  body?: string;
  helpfulVotes?: number;
  totalVotes?: number;
  reviewedAt?: string;
  status?: string;
};

export type MarketProductBlueprintReviewPage = {
  items: MarketProductBlueprintReview[];
  page: number;
  perPage: number;
  total: number;
  hasNext: boolean;
};

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
  tokenIcon?: string;
  brandName?: string;
  avatarName?: string;
  avatarIcon?: string;

  images?: MarketResaleConditionImage[];
  conditionImages?: MarketResaleConditionImage[];

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

export type MarketResaleDetailResponse = {
  data: MarketResaleListing;
};

export type MarketResaleConditionImagesResponse =
  | MarketResaleConditionImage[]
  | {
      data?: MarketResaleConditionImage[];
      items?: MarketResaleConditionImage[];
    };

export type MarketProductBlueprintReviewsResponse =
  | MarketProductBlueprintReviewPage
  | {
      data?: unknown;
      items?: unknown;
      reviews?: unknown;
      page?: unknown;
      perPage?: unknown;
      total?: unknown;
      totalCount?: unknown;
      hasNext?: unknown;
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
   * backendのMarketQueryへ明示的な検索条件として渡す場合に使用する。
   * 現在のavatarIdをフロントエンド側で自動取得する処理は行わない。
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

const MARKET_RESALES_PATH = "/mall/market/resales";

const MARKET_CATALOG_PRODUCT_BLUEPRINTS_PATH =
  "/mall/catalog/product-blueprints";

function normalizeApiBaseUrl(): string {
  const baseUrl = getApiBaseUrl();

  if (
    typeof baseUrl === "string" &&
    baseUrl.trim() !== ""
  ) {
    return baseUrl.replace(/\/+$/, "");
  }

  return "";
}

function normalizeString(value: unknown): string {
  if (typeof value !== "string") {
    return "";
  }

  return value.trim();
}

function toFiniteNumber(
  value: unknown,
  fallback = 0,
): number {
  if (isFiniteNumber(value)) {
    return value;
  }

  if (
    typeof value === "string" &&
    value.trim() !== ""
  ) {
    const parsed = Number(value);

    return isFiniteNumber(parsed)
      ? parsed
      : fallback;
  }

  return fallback;
}

function toBoolean(value: unknown): boolean {
  return value === true;
}

function appendString(
  searchParams: URLSearchParams,
  key: string,
  value: unknown,
): void {
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
): void {
  if (!isFiniteNumber(value)) {
    return;
  }

  searchParams.set(key, String(value));
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

function buildMarketResaleSearchParams(
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

function getErrorMessage(
  status: number,
): string {
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
  const contentType =
    response.headers.get(
      "content-type",
    ) ?? "";

  if (
    !contentType.includes(
      "application/json",
    )
  ) {
    throw new Error(
      fallbackMessage,
    );
  }

  const data =
    (await response.json()) as T;

  if (!response.ok) {
    throw new Error(
      getErrorMessage(
        response.status,
      ),
    );
  }

  return data;
}

function isMarketResaleConditionImage(
  value: unknown,
): value is MarketResaleConditionImage {
  if (
    !isRecord(value) ||
    Array.isArray(value)
  ) {
    return false;
  }

  return (
    normalizeString(value.id) !== "" &&
    normalizeString(value.url) !== ""
  );
}

function normalizeMarketResaleConditionImagesResponse(
  response: MarketResaleConditionImagesResponse,
): MarketResaleConditionImage[] {
  if (Array.isArray(response)) {
    return response.filter(
      isMarketResaleConditionImage,
    );
  }

  if (Array.isArray(response.data)) {
    return response.data.filter(
      isMarketResaleConditionImage,
    );
  }

  if (Array.isArray(response.items)) {
    return response.items.filter(
      isMarketResaleConditionImage,
    );
  }

  return [];
}

function normalizeReview(
  value: unknown,
): MarketProductBlueprintReview | null {
  if (
    !isRecord(value) ||
    Array.isArray(value)
  ) {
    return null;
  }

  const id =
    normalizeString(value.id);

  if (!id) {
    return null;
  }

  return {
    id,

    productBlueprintId:
      normalizeString(
        value.productBlueprintId,
      ),

    avatarId:
      normalizeString(
        value.avatarId,
      ),

    avatarName:
      normalizeString(
        value.avatarName,
      ),

    avatarIcon:
      normalizeString(
        value.avatarIcon,
      ),

    rating:
      toFiniteNumber(
        value.rating,
      ),

    title:
      normalizeString(
        value.title,
      ),

    body:
      normalizeString(
        value.body,
      ),

    helpfulVotes:
      toFiniteNumber(
        value.helpfulVotes,
      ),

    totalVotes:
      toFiniteNumber(
        value.totalVotes,
      ),

    reviewedAt:
      normalizeString(
        value.reviewedAt ||
          value.createdAt,
      ),

    status:
      normalizeString(
        value.status,
      ),
  };
}

function normalizeMarketProductBlueprintReviewsResponse(
  response: MarketProductBlueprintReviewsResponse,
  fallbackPage: number,
  fallbackPerPage: number,
): MarketProductBlueprintReviewPage {
  const root:
    Record<string, unknown> =
    isRecord(response) &&
    !Array.isArray(response)
      ? response
      : {};

  const rawData =
    root["data"];

  const data:
    Record<string, unknown> | null =
    isRecord(rawData) &&
    !Array.isArray(rawData)
      ? rawData
      : null;

  const source:
    Record<string, unknown> =
    data ?? root;

  const rawItems:
    unknown[] =
    Array.isArray(
      source["items"],
    )
      ? source["items"]
      : Array.isArray(
            source["reviews"],
          )
        ? source["reviews"]
        : [];

  const items =
    rawItems
      .map(
        (item: unknown) =>
          normalizeReview(item),
      )
      .filter(
        (
          item:
            | MarketProductBlueprintReview
            | null,
        ): item is MarketProductBlueprintReview =>
          item !== null,
      );

  return {
    items,

    page:
      toFiniteNumber(
        source["page"],
        fallbackPage,
      ) ||
      fallbackPage,

    perPage:
      toFiniteNumber(
        source["perPage"],
        fallbackPerPage,
      ) ||
      fallbackPerPage,

    total:
      toFiniteNumber(
        source["total"] ??
          source["totalCount"],
        items.length,
      ),

    hasNext:
      toBoolean(
        source["hasNext"],
      ),
  };
}

export async function fetchMarketResales(
  params: FetchMarketResalesParams = {},
): Promise<MarketResaleListResponse> {
  const apiBaseUrl =
    normalizeApiBaseUrl();

  if (!apiBaseUrl) {
    throw new Error(
      "API Base URLが未設定です。",
    );
  }

  const searchParams =
    buildMarketResaleSearchParams(
      params,
    );

  const query =
    searchParams.toString();

  const url =
    `${apiBaseUrl}${MARKET_RESALES_PATH}` +
    `${query ? `?${query}` : ""}`;

  const response =
    await fetch(url, {
      method: "GET",

      headers: {
        Accept:
          "application/json",
      },

      credentials:
        "include",
    });

  return readJsonResponse<MarketResaleListResponse>(
    response,
    "マーケット一覧APIがJSON以外を返しました。",
  );
}

export async function fetchMarketResaleById(
  resaleId: string,
): Promise<MarketResaleListing> {
  const apiBaseUrl =
    normalizeApiBaseUrl();

  const id =
    resaleId.trim();

  if (!apiBaseUrl) {
    throw new Error(
      "API Base URLが未設定です。",
    );
  }

  if (!id) {
    throw new Error(
      "マーケット出品IDが未指定です。",
    );
  }

  const response =
    await fetch(
      `${apiBaseUrl}${MARKET_RESALES_PATH}/${encodeURIComponent(
        id,
      )}`,
      {
        method: "GET",

        headers: {
          Accept:
            "application/json",
        },

        credentials:
          "include",
      },
    );

  const result =
    await readJsonResponse<MarketResaleDetailResponse>(
      response,
      "マーケット詳細APIがJSON以外を返しました。",
    );

  return result.data;
}

export async function fetchMarketResaleConditionImages(
  resaleId: string,
): Promise<MarketResaleConditionImage[]> {
  const apiBaseUrl =
    normalizeApiBaseUrl();

  const id =
    resaleId.trim();

  if (!apiBaseUrl) {
    throw new Error(
      "API Base URLが未設定です。",
    );
  }

  if (!id) {
    throw new Error(
      "マーケット出品IDが未指定です。",
    );
  }

  const response =
    await fetch(
      `${apiBaseUrl}${MARKET_RESALES_PATH}/${encodeURIComponent(
        id,
      )}/images`,
      {
        method: "GET",

        headers: {
          Accept:
            "application/json",
        },

        credentials:
          "include",
      },
    );

  const result =
    await readJsonResponse<MarketResaleConditionImagesResponse>(
      response,
      "マーケット出品画像APIがJSON以外を返しました。",
    );

  return normalizeMarketResaleConditionImagesResponse(
    result,
  );
}

export async function fetchMarketProductBlueprintReviews(
  args: {
    productBlueprintId: string;
    page?: number;
    perPage?: number;
  },
): Promise<MarketProductBlueprintReviewPage> {
  const apiBaseUrl =
    normalizeApiBaseUrl();

  const productBlueprintId =
    args.productBlueprintId.trim();

  const page =
    isFiniteNumber(
      args.page,
    ) &&
    args.page > 0
      ? args.page
      : 1;

  const perPage =
    isFiniteNumber(
      args.perPage,
    ) &&
    args.perPage > 0
      ? args.perPage
      : 20;

  if (!apiBaseUrl) {
    throw new Error(
      "API Base URLが未設定です。",
    );
  }

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

  const response =
    await fetch(
      url.toString(),
      {
        method: "GET",

        headers: {
          Accept:
            "application/json",
        },

        credentials:
          "include",
      },
    );

  const result =
    await readJsonResponse<MarketProductBlueprintReviewsResponse>(
      response,
      "商品レビューAPIがJSON以外を返しました。",
    );

  return normalizeMarketProductBlueprintReviewsResponse(
    result,
    page,
    perPage,
  );
}