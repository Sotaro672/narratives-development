// frontend/amol/src/features/cart/api/cartApi.ts

import { getAuth } from "firebase/auth";

import type {
  CartDTO,
  CartDisplayItem,
  CatalogResponse,
  MeAvatarStateResponse,
} from "../types";
import { normalizeCartItems } from "../utils/cartUtils";

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

export async function fetchCurrentAvatarId(
  apiBaseUrl: string,
): Promise<string> {
  const idToken = await getFirebaseIdToken();

  const response = await fetch(`${apiBaseUrl}/mall/me/avatars/state`, {
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

  const data = (await response.json()) as Partial<MeAvatarStateResponse>;
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

  const searchParams = new URLSearchParams({
    avatarId,
  });

  return fetch(`${apiBaseUrl}${path}?${searchParams.toString()}`, {
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
  const idToken = await getFirebaseIdToken();

  let response = await fetchCartFromPath({
    apiBaseUrl,
    avatarId,
    idToken,
    path: "/mall/me/cart/query",
  });

  if (response.status === 404) {
    response = await fetchCartFromPath({
      apiBaseUrl,
      avatarId,
      idToken,
      path: "/mall/me/cart",
    });
  }

  const contentType = response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    throw new Error("カート取得APIがJSON以外を返しました。");
  }

  if (!response.ok) {
    const message = await readResponseErrorMessage(response);
    throw new Error(message || "カートの取得に失敗しました。");
  }

  const data = (await response.json()) as Partial<CartDTO>;

  return {
    avatarId:
      typeof data.avatarId === "string" && data.avatarId.trim() !== ""
        ? data.avatarId
        : avatarId,
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

  const response = await fetch(`${apiBaseUrl}/mall/me/cart/items`, {
    method: "DELETE",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      Authorization: `Bearer ${idToken}`,
    },
    credentials: "include",
    body: JSON.stringify({
      avatarId: item.avatarId,
      inventoryId: item.inventoryId,
      listId: item.listId,
      modelId: item.modelId,
    }),
  });

  const contentType = response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    throw new Error("カート商品削除APIがJSON以外を返しました。");
  }

  if (!response.ok) {
    const message = await readResponseErrorMessage(response);
    throw new Error(message || "カート商品の削除に失敗しました。");
  }

  const data = (await response.json()) as Partial<CartDTO>;

  return {
    avatarId:
      typeof data.avatarId === "string" && data.avatarId.trim() !== ""
        ? data.avatarId
        : item.avatarId,
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

  const response = await fetch(
    `${apiBaseUrl}/mall/catalog/${encodeURIComponent(listId)}`,
    {
      method: "GET",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${idToken}`,
      },
      credentials: "include",
    },
  );

  const contentType = response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    return null;
  }

  if (!response.ok) {
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
  const baseItems = normalizeCartItems(cart);

  return Promise.all(
    baseItems.map(async (item) => {
      const catalog = await fetchCatalog(apiBaseUrl, item.listId);

      return {
        ...item,
        catalog,
      };
    }),
  );
}