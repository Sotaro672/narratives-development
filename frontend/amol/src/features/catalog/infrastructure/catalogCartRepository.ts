// frontend/amol/src/features/catalog/infrastructure/catalogCartRepository.ts

import { getFirebaseIdToken } from "../../../lib/authToken";
import type {
  CatalogModelVariation,
  CatalogResponse,
} from "../types";
import { readResponseErrorMessage } from "./httpErrorReader";

export async function addCatalogItemToCart(args: {
  apiBaseUrl: string;
  avatarId: string;
  catalog: CatalogResponse;
  selectedModel: CatalogModelVariation;
}): Promise<void> {
  const { apiBaseUrl, avatarId, catalog, selectedModel } = args;

  const inventoryId = catalog.inventory.id || catalog.list.inventoryId;
  const idToken = await getFirebaseIdToken();
  const base = apiBaseUrl.replace(/\/+$/, "");

  const searchParams = new URLSearchParams({
    avatarId,
  });

  const response = await fetch(
    `${base}/mall/me/cart/items?${searchParams.toString()}`,
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