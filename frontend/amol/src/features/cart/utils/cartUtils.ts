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

function getStringValue(value: unknown): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }

  const normalized = value.trim();

  if (normalized === "") {
    return undefined;
  }

  return normalized;
}

function getNumberValue(value: unknown): number | undefined {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  if (typeof value === "string") {
    const parsed = Number(value);

    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }

  return undefined;
}

function normalizeCartItem(args: {
  cart: CartDTO;
  item: CartItemDTO;
  fallbackItemKey: string;
}): CartDisplayItem | null {
  const { cart, item, fallbackItemKey } = args;

  const listId = getStringValue(item.listId) ?? "";
  const modelId = getStringValue(item.modelId) ?? "";
  const inventoryId = getStringValue(item.inventoryId) ?? "";
  const itemKey = getStringValue(item.itemKey) ?? fallbackItemKey;

  if (listId === "" || modelId === "") {
    return null;
  }

  return {
    itemKey,
    avatarId: getStringValue(item.avatarId) ?? cart.avatarId,
    inventoryId,
    listId,
    modelId,
    qty: getCartItemQty(item),

    title: getStringValue(item.title),
    listImage: getStringValue(item.listImage),
    price: getNumberValue(item.price),
    productName: getStringValue(item.productName),

    modelKind:
      getStringValue(item.modelKind) ??
      getStringValue(item.kind) ??
      undefined,
    modelNumber: getStringValue(item.modelNumber),
    modelLabel: getStringValue(item.modelLabel),

    size: getStringValue(item.size),
    colorName: getStringValue(item.colorName),
    colorRGB: getNumberValue(item.colorRGB),

    volumeValue: getNumberValue(item.volumeValue),
    volumeUnit: getStringValue(item.volumeUnit),

    catalog: null,
  };
}

export function normalizeCartItems(cart: CartDTO): CartDisplayItem[] {
  if (Array.isArray(cart.items)) {
    return cart.items
      .map((item: CartItemDTO, index: number) => {
        const inventoryId = getStringValue(item.inventoryId) ?? "";
        const listId = getStringValue(item.listId) ?? "";
        const modelId = getStringValue(item.modelId) ?? "";
        const fallbackItemKey = `${inventoryId}__${listId}__${modelId}__${index}`;

        return normalizeCartItem({
          cart,
          item,
          fallbackItemKey,
        });
      })
      .filter((item): item is CartDisplayItem => item !== null);
  }

  return Object.entries(cart.items ?? {})
    .map(([key, item]: [string, CartItemDTO]) => {
      return normalizeCartItem({
        cart,
        item,
        fallbackItemKey: key,
      });
    })
    .filter((item): item is CartDisplayItem => item !== null);
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

export function getCartDisplayItemPrice(item: CartDisplayItem): number | null {
  const catalogPrice = getModelPrice(item.catalog, item.modelId);

  if (catalogPrice !== null) {
    return catalogPrice;
  }

  if (typeof item.price === "number" && Number.isFinite(item.price)) {
    return item.price;
  }

  return null;
}

export function calculateCartTotalAmount(items: CartDisplayItem[]): number {
  return items.reduce((sum, item) => {
    const price = getCartDisplayItemPrice(item);

    if (price === null) {
      return sum;
    }

    return sum + price * item.qty;
  }, 0);
}