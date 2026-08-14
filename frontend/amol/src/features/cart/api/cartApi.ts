// frontend/amol/src/features/cart/api/cartApi.ts

import { HttpError, requestJson } from "../../../lib/http";
import type { CartCatalogSnapshot, CartDTO, CartDisplayItem } from "../types/cart";

/**
 * 現在のカートを取得します。
 *
 * GET /mall/me/cart
 */
export async function fetchCart(): Promise<CartDTO> {
  return requestJson<CartDTO>("/mall/me/cart", {
    method: "GET",
    auth: "required",
    credentials: "include",
    messages: {
      requestErrorMessage: "カートの取得に失敗しました。",
      nonJsonErrorMessage: "カート取得APIがJSON以外を返しました。",
      invalidJsonErrorMessage: "カート取得APIのJSON形式が不正です。",
    },
  });
}

/**
 * 二次流通商品をカートへ追加します。
 *
 * POST /mall/me/cart/resales
 */
export async function addResaleCartItem(args: {
  resaleId: string;
  productId: string;
}): Promise<void> {
  const resaleId = args.resaleId.trim();
  const productId = args.productId.trim();

  if (!resaleId || !productId) {
    throw new Error("出品情報が不足しています。");
  }

  await requestJson<CartDTO>("/mall/me/cart/resales", {
    method: "POST",
    auth: "required",
    credentials: "include",
    json: {
      resaleId,
      productId,
    },
    messages: {
      requestErrorMessage: "カートへの追加に失敗しました。",
      nonJsonErrorMessage: "カート追加APIがJSON以外を返しました。",
      invalidJsonErrorMessage: "カート追加APIのJSON形式が不正です。",
    },
  });
}

/**
 * カートから商品を削除します。
 *
 * 通常販売商品:
 * DELETE /mall/me/cart/items
 *
 * 二次流通商品:
 * DELETE /mall/me/cart/resales
 */
export async function removeCartItem(args: {
  item: CartDisplayItem;
}): Promise<CartDTO> {
  const { item } = args;
  const isResale = item.type === "resale";

  const path = isResale
    ? "/mall/me/cart/resales"
    : "/mall/me/cart/items";

  const body = isResale
    ? {
        resaleId: item.resaleId,
        productId: item.productId,
      }
    : {
        inventoryId: item.inventoryId,
        listId: item.listId,
        modelId: item.modelId,
      };

  return requestJson<CartDTO>(path, {
    method: "DELETE",
    auth: "required",
    credentials: "include",
    json: body,
    messages: {
      requestErrorMessage: "カート商品の削除に失敗しました。",
      nonJsonErrorMessage: "カート商品削除APIがJSON以外を返しました。",
      invalidJsonErrorMessage: "カート商品削除APIのJSON形式が不正です。",
    },
  });
}

/**
 * 通常販売商品のカタログ情報を取得します。
 *
 * GET /mall/catalog/:listId
 *
 * カタログが取得できない場合はnullを返します。
 */
export async function fetchCatalog(
  listId: string,
): Promise<CartCatalogSnapshot | null> {
  const normalizedListId = listId.trim();

  if (!normalizedListId) {
    return null;
  }

  try {
    return await requestJson<CartCatalogSnapshot>(
      `/mall/catalog/${encodeURIComponent(normalizedListId)}`,
      {
        method: "GET",
        auth: "required",
        credentials: "include",
        messages: {
          requestErrorMessage: "カート用カタログの取得に失敗しました。",
          nonJsonErrorMessage: "カート用カタログAPIがJSON以外を返しました。",
          invalidJsonErrorMessage: "カート用カタログAPIのJSON形式が不正です。",
        },
      },
    );
  } catch (error) {
    if (
      error instanceof HttpError ||
      (error instanceof Error &&
        (error.message === "カート用カタログAPIがJSON以外を返しました。" ||
          error.message === "カート用カタログAPIのJSON形式が不正です。"))
    ) {
      return null;
    }

    throw error;
  }
}