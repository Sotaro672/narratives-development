// frontend/amol/src/features/cart/api/cartApi.ts

import { getFirebaseIdToken } from "../../../lib/authToken";
import { fetchCurrentAvatarId } from "../../catalog/infrastructure/avatarStateRepository";
import { readResponseErrorMessage } from "../../catalog/infrastructure/httpErrorReader";

import type {
  CartDTO,
  CartDisplayItem,
  CartItemDTO,
  CartItemType,
  CatalogResponse,
} from "../types";

async function fetchCartFromPath(args: {
  apiBaseUrl: string;
  idToken: string;
  path: string;
}): Promise<Response> {
  const { apiBaseUrl, idToken, path } = args;
  const base = apiBaseUrl.replace(/\/+$/, "");

  return fetch(`${base}${path}`, {
    method: "GET",
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${idToken}`,
    },
    credentials: "include",
  });
}

export { fetchCurrentAvatarId };

export async function fetchCart(
  apiBaseUrl: string,
  avatarId: string,
): Promise<CartDTO> {
  const normalizedAvatarId = avatarId.trim();

  if (!normalizedAvatarId) {
    throw new Error("現在のavatarIdが見つかりません。");
  }

  const idToken = await getFirebaseIdToken();

  const response = await fetchCartFromPath({
    apiBaseUrl,
    idToken,
    path: "/mall/me/cart",
  });

  if (!response.ok) {
    const message = await readResponseErrorMessage(response);
    throw new Error(message || "カートの取得に失敗しました。");
  }

  const contentType = response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    throw new Error("カート取得APIがJSON以外を返しました。");
  }

  const data = (await response.json()) as Partial<CartDTO>;

  return {
    avatarId:
      typeof data.avatarId === "string" &&
      data.avatarId.trim() !== ""
        ? data.avatarId
        : normalizedAvatarId,
    items:
      data.items &&
      !Array.isArray(data.items) &&
      typeof data.items === "object"
        ? data.items
        : {},
    createdAt: data.createdAt ?? null,
    updatedAt: data.updatedAt ?? null,
    expiresAt: data.expiresAt ?? null,
  };
}

export async function removeCartItem(args: {
  apiBaseUrl: string;
  item: CartDisplayItem;
}): Promise<CartDTO> {
  const { apiBaseUrl, item } = args;
  const idToken = await getFirebaseIdToken();

  const isResale = item.type === "resale";
  const path = isResale
    ? "/mall/me/cart/resales"
    : "/mall/me/cart/items";

  const base = apiBaseUrl.replace(/\/+$/, "");

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

  const response = await fetch(`${base}${path}`, {
    method: "DELETE",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      Authorization: `Bearer ${idToken}`,
    },
    credentials: "include",
    body: JSON.stringify(body),
  });

  if (!response.ok) {
    const message = await readResponseErrorMessage(response);
    throw new Error(
      message || "カート商品の削除に失敗しました。",
    );
  }

  const contentType = response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    throw new Error(
      "カート商品削除APIがJSON以外を返しました。",
    );
  }

  const data = (await response.json()) as Partial<CartDTO>;

  return {
    avatarId:
      typeof data.avatarId === "string" &&
      data.avatarId.trim() !== ""
        ? data.avatarId
        : item.avatarId,
    items:
      data.items &&
      !Array.isArray(data.items) &&
      typeof data.items === "object"
        ? data.items
        : {},
    createdAt: data.createdAt ?? null,
    updatedAt: data.updatedAt ?? null,
    expiresAt: data.expiresAt ?? null,
  };
}

export async function fetchCatalog(
  apiBaseUrl: string,
  listId: string,
): Promise<CatalogResponse | null> {
  const idToken = await getFirebaseIdToken();
  const base = apiBaseUrl.replace(/\/+$/, "");

  const response = await fetch(
    `${base}/mall/catalog/${encodeURIComponent(listId)}`,
    {
      method: "GET",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${idToken}`,
      },
      credentials: "include",
    },
  );

  if (!response.ok) {
    return null;
  }

  const contentType = response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    return null;
  }

  return (await response.json()) as CatalogResponse;
}

export async function fetchCartItemsWithCatalog(args: {
  apiBaseUrl: string;
  avatarId: string;
}): Promise<CartDisplayItem[]> {
  const { apiBaseUrl, avatarId } = args;

  const cart = await fetchCart(apiBaseUrl, avatarId);
  const baseItems = cartDTOToDisplayItems(cart);

  return Promise.all(
    baseItems.map(async (item) => {
      if (isResaleDisplayItem(item)) {
        return {
          ...item,
          catalog: null,
        };
      }

      try {
        const catalog = item.listId
          ? await fetchCatalog(apiBaseUrl, item.listId)
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
    }),
  );
}

function cartDTOToDisplayItems(
  cart: CartDTO,
): CartDisplayItem[] {
  const avatarId = cart.avatarId;
  const rawItems = cart.items;

  if (
    !rawItems ||
    Array.isArray(rawItems) ||
    typeof rawItems !== "object"
  ) {
    return [];
  }

  return Object.entries(rawItems)
    .map(([itemKey, item]) =>
      cartItemToDisplayItem({
        avatarId,
        itemKey,
        item,
      }),
    )
    .filter(
      (item): item is CartDisplayItem =>
        item !== null,
    );
}

function cartItemToDisplayItem(args: {
  avatarId: string;
  itemKey: string;
  item: CartItemDTO;
}): CartDisplayItem | null {
  const { avatarId, itemKey, item } = args;

  if (item.type === "resale") {
    return resaleCartItemToDisplayItem({
      avatarId,
      itemKey,
      item,
    });
  }

  if (item.type === "list") {
    return listCartItemToDisplayItem({
      avatarId,
      itemKey,
      item,
    });
  }

  return null;
}

function listCartItemToDisplayItem(args: {
  avatarId: string;
  itemKey: string;
  item: CartItemDTO;
}): CartDisplayItem | null {
  const { avatarId, itemKey, item } = args;

  const inventoryId = asNonEmptyString(
    item.inventoryId,
  );
  const listId = asNonEmptyString(item.listId);
  const modelId = asNonEmptyString(item.modelId);
  const qty = normalizeQty(item.qty);

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
    type: "list",
    inventoryId,
    listId,
    modelId,
    qty,
    catalog: null,
  };
}

function resaleCartItemToDisplayItem(args: {
  avatarId: string;
  itemKey: string;
  item: CartItemDTO;
}): CartDisplayItem | null {
  const { avatarId, itemKey, item } = args;

  const resaleId = asNonEmptyString(item.resaleId);
  const productId = asNonEmptyString(item.productId);

  if (!resaleId || !productId) {
    return null;
  }

  return {
    ...item,
    avatarId,
    itemKey,
    type: "resale",
    resaleId,
    productId,
    productBlueprintId: asNonEmptyString(
      item.productBlueprintId,
    ),
    tokenBlueprintId: asNonEmptyString(
      item.tokenBlueprintId,
    ),
    brandId: asNonEmptyString(item.brandId),
    title: asNonEmptyString(item.title),
    productName: asNonEmptyString(item.productName),
    listImage: asNonEmptyString(item.listImage),
    imageUrl: asNonEmptyString(item.imageUrl),
    price: normalizePrice(item.price),
    qty: 1,
    catalog: null,
  };
}

function isResaleDisplayItem(
  item: CartDisplayItem,
): boolean {
  return item.type === "resale";
}

function normalizeQty(value: unknown): number {
  if (
    typeof value !== "number" ||
    !Number.isFinite(value) ||
    value <= 0
  ) {
    return 1;
  }

  return Math.floor(value);
}

function normalizePrice(value: unknown): number {
  if (
    typeof value !== "number" ||
    !Number.isFinite(value) ||
    value < 0
  ) {
    return 0;
  }

  return Math.floor(value);
}

function asNonEmptyString(
  value: unknown,
): string {
  if (typeof value !== "string") {
    return "";
  }

  return value.trim();
}