//frontend\amol\src\features\cart\utils\cartUtils.ts
import type {
  CartDTO,
  CartDisplayItem,
  CartItemDTO,
  CatalogModelVariation,
  CatalogResponse,
} from "../types";

function normalizeQty(item: CartItemDTO): number {
  if (typeof item.qty === "number" && !Number.isNaN(item.qty)) {
    return item.qty;
  }

  if (typeof item.quantity === "number" && !Number.isNaN(item.quantity)) {
    return item.quantity;
  }

  return 0;
}

function normalizePrice(value: unknown): number | undefined {
  if (typeof value !== "number" || Number.isNaN(value)) {
    return undefined;
  }

  return value;
}

function normalizeVolumeValue(value: unknown): number | undefined {
  if (typeof value !== "number" || Number.isNaN(value)) {
    return undefined;
  }

  return value;
}

function normalizeColorRGB(value: unknown): number | undefined {
  if (typeof value !== "number" || Number.isNaN(value)) {
    return undefined;
  }

  return value;
}

function normalizeString(value: unknown): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }

  return value;
}

function normalizeCartItem(args: {
  avatarId: string;
  itemKey: string;
  item: CartItemDTO;
}): CartDisplayItem {
  const { avatarId, itemKey, item } = args;

  const inventoryId = normalizeString(item.inventoryId) ?? "";
  const listId = normalizeString(item.listId) ?? "";
  const modelId = normalizeString(item.modelId) ?? "";

  return {
    itemKey: normalizeString(item.itemKey) ?? itemKey,
    avatarId: normalizeString(item.avatarId) ?? avatarId,
    inventoryId,
    listId,
    modelId,
    qty: normalizeQty(item),

    title: normalizeString(item.title),
    listImage: normalizeString(item.listImage),
    price: normalizePrice(item.price),
    productName: normalizeString(item.productName),

    modelKind: item.modelKind ?? item.kind ?? "unknown",
    modelNumber: normalizeString(item.modelNumber),
    modelLabel: normalizeString(item.modelLabel),

    size: normalizeString(item.size),
    colorName: normalizeString(item.colorName),
    colorRGB: normalizeColorRGB(item.colorRGB),

    volumeValue: normalizeVolumeValue(item.volumeValue),
    volumeUnit: normalizeString(item.volumeUnit),

    catalog: null,
  };
}

export function normalizeCartItems(cart: CartDTO): CartDisplayItem[] {
  const avatarId = cart.avatarId;

  if (Array.isArray(cart.items)) {
    return cart.items.map((item, index) => {
      const itemKey = normalizeString(item.itemKey) ?? `${avatarId}__${index}`;

      return normalizeCartItem({
        avatarId,
        itemKey,
        item,
      });
    });
  }

  return Object.entries(cart.items).map(([itemKey, item]) => {
    return normalizeCartItem({
      avatarId,
      itemKey,
      item,
    });
  });
}

export function getModelVariations(
  catalog: CatalogResponse | null | undefined,
): CatalogModelVariation[] {
  return catalog?.modelVariations ?? [];
}

export function getModelVariation(
  catalog: CatalogResponse | null | undefined,
  modelId: string,
): CatalogModelVariation | null {
  const models = getModelVariations(catalog);

  return (
    models.find((model) => {
      return model.id === modelId;
    }) ?? null
  );
}

export function getModelPrice(
  catalog: CatalogResponse | null | undefined,
  modelId: string,
): number | null {
  const model = getModelVariation(catalog, modelId);

  if (model && "price" in model && typeof model.price === "number") {
    return model.price;
  }

  const price = catalog?.list.prices.find((item) => item.modelId === modelId);

  if (typeof price?.price === "number") {
    return price.price;
  }

  return null;
}

export function getCartItemPrice(item: CartDisplayItem): number | null {
  if (typeof item.price === "number") {
    return item.price;
  }

  return getModelPrice(item.catalog, item.modelId);
}

export function calculateCartTotalAmount(items: CartDisplayItem[]): number {
  return items.reduce((total, item) => {
    const price = getCartItemPrice(item);

    if (price === null) {
      return total;
    }

    return total + price * item.qty;
  }, 0);
}

export function getPrimaryCatalogImage(
  catalog: CatalogResponse | null | undefined,
): string {
  const primaryImage = catalog?.listImages?.[0];

  if (typeof primaryImage?.url === "string" && primaryImage.url !== "") {
    return primaryImage.url;
  }

  if (typeof catalog?.list.image === "string" && catalog.list.image !== "") {
    return catalog.list.image;
  }

  return "";
}

export function formatYen(amount: number): string {
  return new Intl.NumberFormat("ja-JP", {
    style: "currency",
    currency: "JPY",
    maximumFractionDigits: 0,
  }).format(amount);
}

export function formatPrice(amount: number): string {
  return formatYen(amount);
}