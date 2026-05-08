// frontend/amol/src/features/cart/utils/cartUtils.ts

import type {
  CartDTO,
  CartDisplayItem,
  CartItemDTO,
  CatalogModelVariation,
  CatalogResponse,
} from "../types";

export function formatPrice(price: number): string {
  if (!Number.isFinite(price)) {
    return "価格未設定";
  }

  return `${price.toLocaleString("ja-JP")}円`;
}

export function formatYen(amount: number): string {
  return new Intl.NumberFormat("ja-JP", {
    style: "currency",
    currency: "JPY",
  }).format(amount);
}

export function getCartItemQty(item: CartItemDTO): number {
  const rawQty = item.qty ?? item.quantity;

  if (typeof rawQty === "number" && Number.isFinite(rawQty)) {
    return rawQty;
  }

  if (typeof rawQty === "string") {
    const parsed = Number(rawQty);

    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }

  return 1;
}

export function normalizeCartItems(cart: CartDTO): CartDisplayItem[] {
  if (Array.isArray(cart.items)) {
    return cart.items
      .map((item: CartItemDTO, index: number) => {
        const listId =
          typeof item.listId === "string" ? item.listId.trim() : "";
        const modelId =
          typeof item.modelId === "string" ? item.modelId.trim() : "";
        const inventoryId =
          typeof item.inventoryId === "string" ? item.inventoryId.trim() : "";
        const itemKey =
          typeof item.itemKey === "string" && item.itemKey.trim() !== ""
            ? item.itemKey.trim()
            : `${inventoryId}__${listId}__${modelId}__${index}`;

        return {
          itemKey,
          avatarId: cart.avatarId,
          inventoryId,
          listId,
          modelId,
          qty: getCartItemQty(item),
          catalog: null,
        };
      })
      .filter((item) => item.listId !== "" && item.modelId !== "");
  }

  return Object.entries(cart.items ?? {})
    .map(([key, item]: [string, CartItemDTO]) => {
      const listId = typeof item.listId === "string" ? item.listId.trim() : "";
      const modelId =
        typeof item.modelId === "string" ? item.modelId.trim() : "";
      const inventoryId =
        typeof item.inventoryId === "string" ? item.inventoryId.trim() : "";
      const itemKey =
        typeof item.itemKey === "string" && item.itemKey.trim() !== ""
          ? item.itemKey.trim()
          : key;

      return {
        itemKey,
        avatarId: cart.avatarId,
        inventoryId,
        listId,
        modelId,
        qty: getCartItemQty(item),
        catalog: null,
      };
    })
    .filter((item) => item.listId !== "" && item.modelId !== "");
}

export function getPrimaryCatalogImage(catalog: CatalogResponse | null): string {
  const images = [...(catalog?.listImages ?? [])]
    .filter((image) => image.url)
    .sort((a, b) => {
      if (a.displayOrder !== b.displayOrder) {
        return a.displayOrder - b.displayOrder;
      }

      return a.id.localeCompare(b.id);
    });

  return images[0]?.url ?? "";
}

export function getModelVariation(
  catalog: CatalogResponse | null,
  modelId: string,
): CatalogModelVariation | null {
  return catalog?.modelVariations.find((model) => model.id === modelId) ?? null;
}

export function getModelPrice(
  catalog: CatalogResponse | null,
  modelId: string,
): number | null {
  const price = catalog?.list.prices.find((row) => row.modelId === modelId);

  if (!price || !Number.isFinite(price.price)) {
    return null;
  }

  return price.price;
}

export function calculateCartTotalAmount(items: CartDisplayItem[]): number {
  return items.reduce((sum, item) => {
    const price = getModelPrice(item.catalog, item.modelId);

    if (price === null) {
      return sum;
    }

    return sum + price * item.qty;
  }, 0);
}