// frontend/amol/src/features/list/infrastructure/listApi.ts

import {
  getApiBaseUrl,
} from "../../../lib/apiBaseUrl";

import type {
  MallCatalogResponse,
  MallListIndexResponse,
} from "../types/list";

type FetchMallListsArgs = {
  page: number;
  perPage: number;
};

function isJsonResponse(
  response: Response,
): boolean {
  const contentType =
    response.headers.get(
      "content-type",
    ) ?? "";

  return contentType.includes(
    "application/json",
  );
}

/**
 * 商品一覧を取得します。
 *
 * GET /mall/lists
 */
export async function fetchMallLists({
  page,
  perPage,
}: FetchMallListsArgs): Promise<MallListIndexResponse> {
  const apiBaseUrl =
    getApiBaseUrl();

  const searchParams =
    new URLSearchParams({
      page: String(page),
      perPage: String(perPage),
    });

  const response =
    await fetch(
      `${apiBaseUrl}/mall/lists?${searchParams.toString()}`,
      {
        method: "GET",
        headers: {
          Accept: "application/json",
        },
        credentials: "include",
      },
    );

  if (!isJsonResponse(response)) {
    throw new Error(
      "商品一覧APIがJSON以外を返しました。",
    );
  }

  const data =
    (await response.json()) as Partial<MallListIndexResponse>;

  if (!response.ok) {
    throw new Error(
      "商品一覧の取得に失敗しました。",
    );
  }

  if (!Array.isArray(data.items)) {
    throw new Error(
      "商品一覧APIのitemsが配列ではありません。",
    );
  }

  return {
    ...data,
    items: data.items,
    page:
      typeof data.page === "number" &&
      data.page > 0
        ? data.page
        : page,
    totalPages:
      typeof data.totalPages ===
        "number" &&
      data.totalPages > 0
        ? data.totalPages
        : 1,
  } as MallListIndexResponse;
}

/**
 * 商品一覧カードの補完に使用する
 * カタログ情報を取得します。
 *
 * GET /mall/catalog/:listId
 */
export async function fetchListCatalog(
  listId: string,
): Promise<MallCatalogResponse | null> {
  const normalizedListId =
    listId.trim();

  if (!normalizedListId) {
    return null;
  }

  const apiBaseUrl =
    getApiBaseUrl();

  try {
    const response =
      await fetch(
        `${apiBaseUrl}/mall/catalog/${encodeURIComponent(
          normalizedListId,
        )}`,
        {
          method: "GET",
          headers: {
            Accept: "application/json",
          },
          credentials: "include",
        },
      );

    if (
      !response.ok ||
      !isJsonResponse(response)
    ) {
      return null;
    }

    return (
      await response.json()
    ) as MallCatalogResponse;
  } catch {
    return null;
  }
}