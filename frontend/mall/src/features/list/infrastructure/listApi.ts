// frontend/amol/src/features/list/infrastructure/listApi.ts

import {
  requestJson,
} from "../../../lib/http";

import type {
  MallCatalogResponse,
  MallListIndexResponse,
} from "../../shared/types/list";

type FetchMallListsArgs = {
  page: number;
  perPage: number;
};

/**
 * 商品一覧を取得します。
 *
 * GET /mall/lists
 */
export async function fetchMallLists({
  page,
  perPage,
}: FetchMallListsArgs): Promise<MallListIndexResponse> {
  const data =
    await requestJson<
      Partial<MallListIndexResponse>
    >(
      "/mall/lists",
      {
        method: "GET",
        auth: "none",
        credentials:
          "include",
        query: {
          page,
          perPage,
        },
        messages: {
          requestErrorMessage:
            "商品一覧の取得に失敗しました。",
          nonJsonErrorMessage:
            "商品一覧APIがJSON以外を返しました。",
          invalidJsonErrorMessage:
            "商品一覧APIのJSON形式が不正です。",
        },
      },
    );

  if (
    !Array.isArray(
      data.items,
    )
  ) {
    throw new Error(
      "商品一覧APIのitemsが配列ではありません。",
    );
  }

  return {
    ...data,
    items:
      data.items,
    page:
      typeof data.page ===
        "number" &&
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

  const encodedListId =
    encodeURIComponent(
      normalizedListId,
    );

  try {
    return await requestJson<
      MallCatalogResponse
    >(
      `/mall/catalog/${encodedListId}`,
      {
        method: "GET",
        auth: "none",
        credentials:
          "include",
        messages: {
          requestErrorMessage:
            "カタログの取得に失敗しました。",
          nonJsonErrorMessage:
            "カタログAPIがJSON以外を返しました。",
          invalidJsonErrorMessage:
            "カタログAPIのJSON形式が不正です。",
        },
      },
    );
  } catch {
    return null;
  }
}