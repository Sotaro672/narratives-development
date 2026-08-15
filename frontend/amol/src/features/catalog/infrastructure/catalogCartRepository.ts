// frontend/amol/src/features/catalog/infrastructure/catalogCartRepository.ts

import { getFirebaseIdToken } from "../../../lib/authToken";
import type { CatalogModelVariation, CatalogResponse } from "../../shared/types/catalog";
import { readResponseErrorMessage } from "./httpErrorReader";

export async function addCatalogItemToCart(args: {
  apiBaseUrl: string;
  catalog: CatalogResponse;
  selectedModel: CatalogModelVariation;
}): Promise<void> {
  const { apiBaseUrl, catalog, selectedModel } = args;
  const idToken = await getFirebaseIdToken();

  const response = await fetch(`${apiBaseUrl}/mall/me/cart/items`, {
    method: "POST",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      Authorization: `Bearer ${idToken}`,
    },
    credentials: "include",
    body: JSON.stringify({
      inventoryId: catalog.inventory.id,
      listId: catalog.list.id,
      modelId: selectedModel.id,
      qty: 1,
    }),
  });

  if (!response.ok) {
    const message = await readResponseErrorMessage(response);
    throw new Error(message || "カートへの追加に失敗しました。");
  }
}