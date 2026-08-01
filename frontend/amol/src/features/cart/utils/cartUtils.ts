// frontend/amol/src/features/cart/utils/cartUtils.ts

import type {
  CartCatalogSnapshot,
  CartDisplayItem,
  CartModelSnapshot,
} from "../types";

function getModelVariations(
  catalog: CartCatalogSnapshot | null | undefined,
): CartModelSnapshot[] {
  return catalog?.modelVariations ?? [];
}

export function getModelVariation(
  catalog: CartCatalogSnapshot | null | undefined,
  modelId: string,
): CartModelSnapshot | null {
  if (!modelId) {
    return null;
  }

  const models = getModelVariations(catalog);

  return (
    models.find((model) => {
      return model.id === modelId;
    }) ?? null
  );
}

export function getModelPrice(
  catalog: CartCatalogSnapshot | null | undefined,
  modelId: string,
): number | null {
  if (!modelId) {
    return null;
  }

  const model = getModelVariation(
    catalog,
    modelId,
  );

  if (
    model &&
    typeof model.price === "number" &&
    Number.isFinite(model.price)
  ) {
    return model.price;
  }

  const price = catalog?.list.prices.find(
    (item) => item.modelId === modelId,
  );

  if (
    typeof price?.price === "number" &&
    Number.isFinite(price.price)
  ) {
    return price.price;
  }

  return null;
}

export function getCartItemPrice(
  item: CartDisplayItem,
): number | null {
  if (
    typeof item.price === "number" &&
    Number.isFinite(item.price)
  ) {
    return item.price;
  }

  return getModelPrice(
    item.catalog,
    item.modelId ?? "",
  );
}

export function calculateCartTotalAmount(
  items: CartDisplayItem[],
): number {
  return items.reduce(
    (total, item) => {
      const price =
        getCartItemPrice(item);

      if (price === null) {
        return total;
      }

      return total + price * item.qty;
    },
    0,
  );
}

export function getPrimaryCatalogImage(
  catalog: CartCatalogSnapshot | null | undefined,
): string {
  const primaryImage =
    catalog?.listImages?.[0];

  if (
    typeof primaryImage?.url === "string" &&
    primaryImage.url !== ""
  ) {
    return primaryImage.url;
  }

  if (
    typeof catalog?.list.image === "string" &&
    catalog.list.image !== ""
  ) {
    return catalog.list.image;
  }

  return "";
}

export function formatYen(
  amount: number,
): string {
  return new Intl.NumberFormat(
    "ja-JP",
    {
      style: "currency",
      currency: "JPY",
      maximumFractionDigits: 0,
    },
  ).format(amount);
}

export function formatPrice(
  amount: number,
): string {
  return formatYen(amount);
}