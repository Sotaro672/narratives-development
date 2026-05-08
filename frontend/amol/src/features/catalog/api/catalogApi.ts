//frontend\amol\src\features\catalog\api\catalogApi.ts
import { getAuth } from "firebase/auth";

import {
  DEFAULT_REVIEW_PAGE,
  DEFAULT_REVIEW_PER_PAGE,
} from "../constants";
import type {
  CatalogProductBlueprintReviewPage,
  CatalogResponse,
  CatalogModelVariation,
  MeAvatarStateResponse,
} from "../types";

export function getApiBaseUrl(): string {
  const env = import.meta.env.VITE_API_BASE_URL;

  if (typeof env === "string" && env.trim() !== "") {
    return env.replace(/\/$/, "");
  }

  return "";
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

export async function fetchCatalogDetail(args: {
  apiBaseUrl: string;
  listId: string;
}): Promise<CatalogResponse> {
  const { apiBaseUrl, listId } = args;

  const response = await fetch(
    `${apiBaseUrl}/mall/catalog/${encodeURIComponent(listId)}`,
    {
      method: "GET",
      headers: {
        Accept: "application/json",
      },
      credentials: "include",
    },
  );

  const contentType = response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    throw new Error("カタログ詳細APIがJSON以外を返しました。");
  }

  const data = (await response.json()) as CatalogResponse;

  if (!response.ok) {
    throw new Error("カタログ詳細の取得に失敗しました。");
  }

  if (!data.list || typeof data.list.title !== "string") {
    throw new Error("カタログ詳細APIのlistが不正です。");
  }

  if (!data.productBlueprint || typeof data.productBlueprint.id !== "string") {
    throw new Error("カタログ詳細APIのproductBlueprintが不正です。");
  }

  return data;
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

export async function fetchCatalogReviews(
  apiBaseUrl: string,
  productBlueprintId: string,
): Promise<CatalogProductBlueprintReviewPage> {
  const searchParams = new URLSearchParams({
    page: String(DEFAULT_REVIEW_PAGE),
    perPage: String(DEFAULT_REVIEW_PER_PAGE),
  });

  const response = await fetch(
    `${apiBaseUrl}/mall/catalog/product-blueprints/${encodeURIComponent(
      productBlueprintId,
    )}/reviews?${searchParams.toString()}`,
    {
      method: "GET",
      headers: {
        Accept: "application/json",
      },
      credentials: "include",
    },
  );

  const contentType = response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    throw new Error("レビュー一覧APIがJSON以外を返しました。");
  }

  const data =
    (await response.json()) as Partial<CatalogProductBlueprintReviewPage>;

  if (!response.ok) {
    throw new Error("レビュー一覧の取得に失敗しました。");
  }

  return {
    items: Array.isArray(data.items) ? data.items : [],
    page:
      typeof data.page === "number" && data.page > 0
        ? data.page
        : DEFAULT_REVIEW_PAGE,
    perPage:
      typeof data.perPage === "number" && data.perPage > 0
        ? data.perPage
        : DEFAULT_REVIEW_PER_PAGE,
    total: typeof data.total === "number" && data.total > 0 ? data.total : 0,
    hasNext: Boolean(data.hasNext),
  };
}

export async function addCatalogItemToCart(args: {
  apiBaseUrl: string;
  avatarId: string;
  catalog: CatalogResponse;
  selectedModel: CatalogModelVariation;
}): Promise<void> {
  const { apiBaseUrl, avatarId, catalog, selectedModel } = args;

  const inventoryId = catalog.inventory.id || catalog.list.inventoryId;
  const idToken = await getFirebaseIdToken();

  const searchParams = new URLSearchParams({
    avatarId,
  });

  const response = await fetch(
    `${apiBaseUrl}/mall/me/cart/items?${searchParams.toString()}`,
    {
      method: "POST",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
        Authorization: `Bearer ${idToken}`,
      },
      credentials: "include",
      body: JSON.stringify({
        avatarId,
        inventoryId,
        listId: catalog.list.id,
        modelId: selectedModel.id,
        qty: 1,
      }),
    },
  );

  if (!response.ok) {
    const message = await readResponseErrorMessage(response);
    throw new Error(message || "カートへの追加に失敗しました。");
  }
}