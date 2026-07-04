// frontend/amol/src/features/cart/api/cartApi.ts

import { getAuth } from "firebase/auth";

import type {
  CartDTO,
  CartDisplayItem,
  CatalogResponse,
} from "../types";

type CartItemType = "list" | "resale";

type RawCartItem = {
  type?: string;

  inventoryId?: string;
  listId?: string;
  modelId?: string;

  resaleId?: string;
  productId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
  brandId?: string;

  title?: string;
  productName?: string;
  listImage?: string;
  imageUrl?: string;

  price?: number;
  qty?: number;

  [key: string]: unknown;
};

type CartDisplayItemWithResale = CartDisplayItem & {
  type?: CartItemType;

  resaleId?: string;
  productId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
  brandId?: string;

  title?: string;
  productName?: string;
  listImage?: string;
  imageUrl?: string;

  price?: number;
};

type MeAvatarResponse = {
  avatarId?: string;
};

function unwrapData(value: unknown): unknown {
  if (!value || typeof value !== "object") {
    return value;
  }

  const record = value as Record<string, unknown>;

  return record.data ?? value;
}

function isMeAvatarResponse(value: unknown): value is MeAvatarResponse {
  if (!value || typeof value !== "object") {
    return false;
  }

  const record = value as Record<string, unknown>;

  return typeof record.avatarId === "string";
}

export async function readResponseErrorMessage(
  response: Response,
): Promise<string> {
  const contentType = response.headers.get("content-type") ?? "";

  if (contentType.includes("application/json")) {
    const data = (await response.json().catch(() => null)) as
      | { error?: unknown; message?: unknown }
      | null;

    if (typeof data?.error === "string" && data.error.trim() !== "") {
      return data.error;
    }

    if (typeof data?.message === "string" && data.message.trim() !== "") {
      return data.message;
    }
  }

  const text = await response.text().catch(() => "");

  if (text.trim() !== "") {
    return text;
  }

  return "リクエストに失敗しました。";
}

export async function getFirebaseIdToken(): Promise<string> {
  const auth = getAuth();
  const user = auth.currentUser;

  if (!user) {
    throw new Error("ログイン情報が見つかりません。再ログインしてください。");
  }

  return user.getIdToken();
}

export async function fetchCurrentAvatarId(apiBaseUrl: string): Promise<string> {
  const idToken = await getFirebaseIdToken();
  const base = apiBaseUrl.replace(/\/+$/, "");

  const response = await fetch(`${base}/mall/me/avatars`, {
    method: "GET",
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${idToken}`,
    },
    credentials: "include",
  });

  if (!response.ok) {
    const message = await readResponseErrorMessage(response);
    throw new Error(message || "現在のアバター情報の取得に失敗しました。");
  }

  const contentType = response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    throw new Error("現在のアバター情報APIがJSON以外を返しました。");
  }

  const responseBody: unknown = await response.json();
  const data = unwrapData(responseBody);

  if (!isMeAvatarResponse(data)) {
    throw new Error("現在のアバター情報APIのレスポンス形式が不正です。");
  }

  const avatarId = data.avatarId?.trim();

  if (!avatarId) {
    throw new Error("現在のavatarIdが見つかりません。");
  }

  return avatarId;
}

async function fetchCartFromPath(args: {
  apiBaseUrl: string;
  avatarId: string;
  idToken: string;
  path: string;
}): Promise<Response> {
  const { apiBaseUrl, avatarId, idToken, path } = args;
  const base = apiBaseUrl.replace(/\/+$/, "");

  const searchParams = new URLSearchParams({
    avatarId,
  });

  return fetch(`${base}${path}?${searchParams.toString()}`, {
    method: "GET",
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${idToken}`,
    },
    credentials: "include",
  });
}

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
    avatarId: normalizedAvatarId,
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
      typeof data.avatarId === "string" && data.avatarId.trim() !== ""
        ? data.avatarId
        : normalizedAvatarId,
    items:
      data.items && (Array.isArray(data.items) || typeof data.items === "object")
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

  const normalized = item as CartDisplayItemWithResale;
  const isResale =
    normalized.type === "resale" ||
    Boolean(normalized.resaleId || normalized.productId);

  const path = isResale ? "/mall/me/cart/resales" : "/mall/me/cart/items";
  const base = apiBaseUrl.replace(/\/+$/, "");

  const body = isResale
    ? {
        avatarId: normalized.avatarId,
        resaleId: normalized.resaleId,
        productId: normalized.productId,
      }
    : {
        avatarId: normalized.avatarId,
        inventoryId: normalized.inventoryId,
        listId: normalized.listId,
        modelId: normalized.modelId,
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
    throw new Error(message || "カート商品の削除に失敗しました。");
  }

  const contentType = response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    throw new Error("カート商品削除APIがJSON以外を返しました。");
  }

  const data = (await response.json()) as Partial<CartDTO>;

  return {
    avatarId:
      typeof data.avatarId === "string" && data.avatarId.trim() !== ""
        ? data.avatarId
        : normalized.avatarId,
    items:
      data.items && (Array.isArray(data.items) || typeof data.items === "object")
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
      const normalized = item as CartDisplayItemWithResale;

      if (isResaleDisplayItem(normalized)) {
        return {
          ...normalized,
          catalog: null,
        };
      }

      try {
        const catalog = normalized.listId
          ? await fetchCatalog(apiBaseUrl, normalized.listId)
          : null;

        return {
          ...normalized,
          catalog,
        };
      } catch {
        return {
          ...normalized,
          catalog: null,
        };
      }
    }),
  );
}

function cartDTOToDisplayItems(cart: CartDTO): CartDisplayItem[] {
  const avatarId = cart.avatarId;
  const rawItems = cart.items;

  if (!rawItems) {
    return [];
  }

  if (Array.isArray(rawItems)) {
    return rawItems
      .map((item, index) =>
        rawCartItemToDisplayItem({
          avatarId,
          itemKey: String(index),
          item: item as RawCartItem,
        }),
      )
      .filter((item): item is CartDisplayItem => item !== null);
  }

  if (typeof rawItems !== "object") {
    return [];
  }

  return Object.entries(rawItems)
    .map(([itemKey, item]) =>
      rawCartItemToDisplayItem({
        avatarId,
        itemKey,
        item: item as RawCartItem,
      }),
    )
    .filter((item): item is CartDisplayItem => item !== null);
}

function rawCartItemToDisplayItem(args: {
  avatarId: string;
  itemKey: string;
  item: RawCartItem;
}): CartDisplayItem | null {
  const { avatarId, itemKey, item } = args;

  if (!item || typeof item !== "object") {
    return null;
  }

  if (inferRawCartItemType(item) === "resale") {
    return rawResaleCartItemToDisplayItem({
      avatarId,
      itemKey,
      item,
    });
  }

  return rawListCartItemToDisplayItem({
    avatarId,
    itemKey,
    item,
  });
}

function rawListCartItemToDisplayItem(args: {
  avatarId: string;
  itemKey: string;
  item: RawCartItem;
}): CartDisplayItem | null {
  const { avatarId, itemKey, item } = args;

  const inventoryId = asNonEmptyString(item.inventoryId);
  const listId = asNonEmptyString(item.listId);
  const modelId = asNonEmptyString(item.modelId);
  const qty = normalizeQty(item.qty);

  if (!inventoryId || !listId || !modelId || qty <= 0) {
    return null;
  }

  return {
    ...(item as Record<string, unknown>),
    avatarId,
    itemKey,
    type: "list",
    inventoryId,
    listId,
    modelId,
    qty,
  } as CartDisplayItem;
}

function rawResaleCartItemToDisplayItem(args: {
  avatarId: string;
  itemKey: string;
  item: RawCartItem;
}): CartDisplayItem | null {
  const { avatarId, itemKey, item } = args;

  const resaleId = asNonEmptyString(item.resaleId);
  const productId = asNonEmptyString(item.productId);

  if (!resaleId || !productId) {
    return null;
  }

  return {
    ...(item as Record<string, unknown>),
    avatarId,
    itemKey,
    type: "resale",
    resaleId,
    productId,
    productBlueprintId: asNonEmptyString(item.productBlueprintId),
    tokenBlueprintId: asNonEmptyString(item.tokenBlueprintId),
    brandId: asNonEmptyString(item.brandId),
    title: asNonEmptyString(item.title),
    productName: asNonEmptyString(item.productName),
    listImage: asNonEmptyString(item.listImage),
    imageUrl: asNonEmptyString(item.imageUrl),
    price: normalizePrice(item.price),
    qty: 1,
  } as CartDisplayItem;
}

function inferRawCartItemType(item: RawCartItem): CartItemType {
  if (item.type === "resale") {
    return "resale";
  }

  if (item.type === "list") {
    return "list";
  }

  if (item.resaleId || item.productId) {
    return "resale";
  }

  return "list";
}

function isResaleDisplayItem(item: CartDisplayItemWithResale): boolean {
  return item.type === "resale" || Boolean(item.resaleId || item.productId);
}

function normalizeQty(value: unknown): number {
  if (typeof value !== "number" || !Number.isFinite(value) || value <= 0) {
    return 1;
  }

  return Math.floor(value);
}

function normalizePrice(value: unknown): number {
  if (typeof value !== "number" || !Number.isFinite(value) || value < 0) {
    return 0;
  }

  return Math.floor(value);
}

function asNonEmptyString(value: unknown): string {
  if (typeof value !== "string") {
    return "";
  }

  return value.trim();
}