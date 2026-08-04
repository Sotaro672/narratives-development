// frontend/amol/src/features/cart/presentation/utils/cartItemDisplay.ts

import {
  textOrEmpty,
} from "../../../../components/utils/textOrEmpty";

import type {
  CartDisplayItem,
  CartModelSnapshot,
} from "../../types/cart";

import {
  getPrimaryCatalogImage,
} from "../../utils/cartUtils";

export function formatAlcoholVolume(
  item: CartDisplayItem,
  model?: CartModelSnapshot | null,
): string {
  const volumeValue =
    model?.volumeValue ??
    item.volumeValue;

  const volumeUnit =
    textOrEmpty(
      model?.volumeUnit,
    ) ||
    textOrEmpty(
      item.volumeUnit,
    );

  if (
    typeof volumeValue === "number" &&
    Number.isFinite(volumeValue) &&
    volumeUnit
  ) {
    return `${volumeValue}${volumeUnit}`;
  }

  return (
    textOrEmpty(
      model?.modelLabel,
    ) ||
    textOrEmpty(
      item.modelLabel,
    ) ||
    "-"
  );
}

export function getCartItemBrandName(
  item: CartDisplayItem,
): string {
  return (
    textOrEmpty(
      item.catalog
        ?.productBlueprint
        .brandName,
    ) ||
    textOrEmpty(
      item.brandName,
    ) ||
    "ブランド未設定"
  );
}

export function getCartItemProductName(
  item: CartDisplayItem,
): string {
  return (
    textOrEmpty(
      item.catalog
        ?.productBlueprint
        .productName,
    ) ||
    textOrEmpty(
      item.productName,
    ) ||
    textOrEmpty(
      item.catalog?.list.title,
    ) ||
    textOrEmpty(
      item.title,
    ) ||
    "商品名未設定"
  );
}

export function getCartItemListTitle(
  item: CartDisplayItem,
): string {
  const catalogTitle =
    textOrEmpty(
      item.catalog?.list.title,
    );

  const itemTitle =
    textOrEmpty(
      item.title,
    );

  const productName =
    getCartItemProductName(item);

  if (
    catalogTitle &&
    catalogTitle !== productName
  ) {
    return catalogTitle;
  }

  if (
    itemTitle &&
    itemTitle !== productName
  ) {
    return itemTitle;
  }

  return "";
}

export function getCartItemImageUrl(
  item: CartDisplayItem,
): string {
  return (
    textOrEmpty(
      getPrimaryCatalogImage(
        item.catalog,
      ),
    ) ||
    textOrEmpty(
      item.imageUrl,
    ) ||
    textOrEmpty(
      item.listImage,
    )
  );
}

export function getCartItemNavigationPath(
  item: CartDisplayItem,
): string {
  const listId =
    textOrEmpty(
      item.listId,
    );

  if (listId) {
    return `/lists/${encodeURIComponent(
      listId,
    )}`;
  }

  const resaleId =
    textOrEmpty(
      item.resaleId,
    );

  if (resaleId) {
    return `/market/${encodeURIComponent(
      resaleId,
    )}`;
  }

  return "";
}