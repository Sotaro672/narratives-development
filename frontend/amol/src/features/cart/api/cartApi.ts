// frontend/amol/src/features/cart/api/cartApi.ts

import {
  HttpError,
  requestJson,
} from "../../../lib/http";
import {
  isFiniteNumber,
  isRecord,
} from "../../../components/utils/typeGuards";

import type {
  CartCatalogSnapshot,
  CartDTO,
  CartDisplayItem,
  CartItemDTO,
} from "../../shared/types/cart";

function normalizeCartDTO(
  data: Partial<CartDTO>,
): CartDTO {
  return {
    avatarId:
      typeof data.avatarId === "string"
        ? data.avatarId.trim()
        : "",

    items:
      isRecord(data.items) &&
      !Array.isArray(data.items)
        ? (data.items as Record<
            string,
            CartItemDTO
          >)
        : {},

    createdAt:
      data.createdAt ?? null,

    updatedAt:
      data.updatedAt ?? null,

    expiresAt:
      data.expiresAt ?? null,
  };
}

export async function fetchCart(
  apiBaseUrl: string,
): Promise<CartDTO> {
  void apiBaseUrl;

  const data =
    await requestJson<Partial<CartDTO>>(
      "/mall/me/cart",
      {
        method: "GET",
        auth: "required",
        credentials: "include",
        messages: {
          requestErrorMessage:
            "カートの取得に失敗しました。",
          nonJsonErrorMessage:
            "カート取得APIがJSON以外を返しました。",
        },
      },
    );

  return normalizeCartDTO(
    data,
  );
}

export async function addResaleCartItem(
  args: {
    resaleId: string;
    productId: string;
  },
): Promise<void> {
  const resaleId =
    args.resaleId.trim();

  const productId =
    args.productId.trim();

  if (
    !resaleId ||
    !productId
  ) {
    throw new Error(
      "出品情報が不足しています。",
    );
  }

  await requestJson<Partial<CartDTO>>(
    "/mall/me/cart/resales",
    {
      method: "POST",
      auth: "required",
      credentials: "include",
      json: {
        resaleId,
        productId,
      },
      messages: {
        requestErrorMessage:
          "カートへの追加に失敗しました。",
        nonJsonErrorMessage:
          "カート追加APIがJSON以外を返しました。",
      },
    },
  );
}

export async function removeCartItem(
  args: {
    apiBaseUrl: string;
    item: CartDisplayItem;
  },
): Promise<CartDTO> {
  const {
    apiBaseUrl,
    item,
  } = args;

  void apiBaseUrl;

  const isResale =
    item.type === "resale";

  const path =
    isResale
      ? "/mall/me/cart/resales"
      : "/mall/me/cart/items";

  const body =
    isResale
      ? {
          resaleId:
            item.resaleId,

          productId:
            item.productId,
        }
      : {
          inventoryId:
            item.inventoryId,

          listId:
            item.listId,

          modelId:
            item.modelId,
        };

  const data =
    await requestJson<Partial<CartDTO>>(
      path,
      {
        method: "DELETE",
        auth: "required",
        credentials: "include",
        json: body,
        messages: {
          requestErrorMessage:
            "カート商品の削除に失敗しました。",
          nonJsonErrorMessage:
            "カート商品削除APIがJSON以外を返しました。",
        },
      },
    );

  return normalizeCartDTO(
    data,
  );
}

export async function fetchCatalog(
  apiBaseUrl: string,
  listId: string,
): Promise<CartCatalogSnapshot | null> {
  void apiBaseUrl;

  try {
    return await requestJson<CartCatalogSnapshot>(
      `/mall/catalog/${encodeURIComponent(
        listId,
      )}`,
      {
        method: "GET",
        auth: "required",
        credentials: "include",
        messages: {
          nonJsonErrorMessage:
            "カート用カタログAPIがJSON以外を返しました。",
        },
      },
    );
  } catch (error) {
    if (
      error instanceof HttpError ||
      (
        error instanceof Error &&
        error.message ===
          "カート用カタログAPIがJSON以外を返しました。"
      )
    ) {
      return null;
    }

    throw error;
  }
}

export async function fetchCartItemsWithCatalog(
  args: {
    apiBaseUrl: string;
  },
): Promise<CartDisplayItem[]> {
  const {
    apiBaseUrl,
  } = args;

  const cart =
    await fetchCart(
      apiBaseUrl,
    );

  const baseItems =
    cartDTOToDisplayItems(
      cart,
    );

  return Promise.all(
    baseItems.map(
      async (
        item,
      ): Promise<CartDisplayItem> => {
        if (
          isResaleDisplayItem(
            item,
          )
        ) {
          return {
            ...item,
            catalog: null,
          };
        }

        try {
          const catalog =
            item.listId
              ? await fetchCatalog(
                  apiBaseUrl,
                  item.listId,
                )
              : null;

          return {
            ...item,
            catalog,
          };
        } catch {
          return {
            ...item,
            catalog: null,
          };
        }
      },
    ),
  );
}

function cartDTOToDisplayItems(
  cart: CartDTO,
): CartDisplayItem[] {
  const avatarId =
    cart.avatarId;

  const rawItems =
    cart.items;

  if (
    !isRecord(rawItems) ||
    Array.isArray(rawItems)
  ) {
    return [];
  }

  return Object.entries(
    rawItems,
  )
    .map(
      ([
        itemKey,
        item,
      ]) =>
        cartItemToDisplayItem({
          avatarId,
          itemKey,
          item:
            item as CartItemDTO,
        }),
    )
    .filter(
      (
        item,
      ): item is CartDisplayItem =>
        item !== null,
    );
}

function cartItemToDisplayItem(
  args: {
    avatarId: string;
    itemKey: string;
    item: CartItemDTO;
  },
): CartDisplayItem | null {
  const {
    avatarId,
    itemKey,
    item,
  } = args;

  if (
    item.type === "resale"
  ) {
    return resaleCartItemToDisplayItem({
      avatarId,
      itemKey,
      item,
    });
  }

  if (
    item.type === "list"
  ) {
    return listCartItemToDisplayItem({
      avatarId,
      itemKey,
      item,
    });
  }

  return null;
}

function listCartItemToDisplayItem(
  args: {
    avatarId: string;
    itemKey: string;
    item: CartItemDTO;
  },
): CartDisplayItem | null {
  const {
    avatarId,
    itemKey,
    item,
  } = args;

  const inventoryId =
    asNonEmptyString(
      item.inventoryId,
    );

  const listId =
    asNonEmptyString(
      item.listId,
    );

  const modelId =
    asNonEmptyString(
      item.modelId,
    );

  const qty =
    normalizeQty(
      item.qty,
    );

  if (
    !inventoryId ||
    !listId ||
    !modelId ||
    qty <= 0
  ) {
    return null;
  }

  return {
    ...item,

    avatarId,
    itemKey,

    type:
      "list",

    inventoryId,
    listId,
    modelId,
    qty,

    catalog:
      null,
  };
}

function resaleCartItemToDisplayItem(
  args: {
    avatarId: string;
    itemKey: string;
    item: CartItemDTO;
  },
): CartDisplayItem | null {
  const {
    avatarId,
    itemKey,
    item,
  } = args;

  const resaleId =
    asNonEmptyString(
      item.resaleId,
    );

  const productId =
    asNonEmptyString(
      item.productId,
    );

  if (
    !resaleId ||
    !productId
  ) {
    return null;
  }

  return {
    ...item,

    avatarId,
    itemKey,

    type:
      "resale",

    resaleId,
    productId,

    productBlueprintId:
      asNonEmptyString(
        item.productBlueprintId,
      ),

    tokenBlueprintId:
      asNonEmptyString(
        item.tokenBlueprintId,
      ),

    brandId:
      asNonEmptyString(
        item.brandId,
      ),

    title:
      asNonEmptyString(
        item.title,
      ),

    productName:
      asNonEmptyString(
        item.productName,
      ),

    brandName:
      asNonEmptyString(
        item.brandName,
      ),

    listImage:
      asNonEmptyString(
        item.listImage,
      ),

    imageUrl:
      asNonEmptyString(
        item.imageUrl,
      ),

    price:
      normalizePrice(
        item.price,
      ),

    qty:
      1,

    catalog:
      null,
  };
}

function isResaleDisplayItem(
  item: CartDisplayItem,
): boolean {
  return (
    item.type === "resale"
  );
}

function normalizeQty(
  value: unknown,
): number {
  if (
    !isFiniteNumber(value) ||
    value <= 0
  ) {
    return 1;
  }

  return Math.floor(
    value,
  );
}

function normalizePrice(
  value: unknown,
): number {
  if (
    !isFiniteNumber(value) ||
    value < 0
  ) {
    return 0;
  }

  return Math.floor(
    value,
  );
}

function asNonEmptyString(
  value: unknown,
): string {
  if (
    typeof value !== "string"
  ) {
    return "";
  }

  return value.trim();
}