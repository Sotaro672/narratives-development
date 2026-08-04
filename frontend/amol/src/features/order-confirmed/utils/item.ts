// frontend/amol/src/features/order-confirmed/utils/item.ts

import {
  getCartItemPrice,
  getModelVariation,
} from "../../cart/utils/cartUtils";
import type {
  CartDisplayItem,
} from "../../shared/types/cart";
import type {
  OrderConfirmedItemViewModel,
} from "../../shared/types/orderConfirmed";

function getItemTitle(
  item: CartDisplayItem,
): string {
  const catalog = item.catalog;

  return (
    item.productName ||
    item.title ||
    catalog?.productBlueprint.productName ||
    catalog?.list.title ||
    "商品名未設定"
  );
}

function getAlcoholModelLabel(
  item: CartDisplayItem,
): string {
  const model = getModelVariation(
    item.catalog,
    item.modelId ?? "",
  );

  if (item.modelLabel) {
    return item.modelLabel;
  }

  if (model?.modelLabel) {
    return model.modelLabel;
  }

  const modelNumber =
    item.modelNumber ??
    model?.modelNumber ??
    "";

  const volumeValue =
    item.volumeValue ??
    model?.volumeValue;

  const volumeUnit =
    item.volumeUnit ??
    model?.volumeUnit ??
    "";

  const volumeLabel =
    typeof volumeValue === "number" &&
    volumeUnit
      ? `${volumeValue}${volumeUnit}`
      : "";

  return [
    modelNumber,
    volumeLabel,
  ]
    .filter(Boolean)
    .join(" / ");
}

function getApparelModelLabel(
  item: CartDisplayItem,
): string {
  const model = getModelVariation(
    item.catalog,
    item.modelId ?? "",
  );

  const colorName =
    item.colorName ??
    model?.colorName ??
    "";

  const size =
    item.size ??
    model?.size ??
    "";

  return [
    colorName
      ? `カラー: ${colorName}`
      : "",
    size
      ? `サイズ: ${size}`
      : "",
  ]
    .filter(Boolean)
    .join(" / ");
}

function getItemModelKind(
  item: CartDisplayItem,
): string {
  const model = getModelVariation(
    item.catalog,
    item.modelId ?? "",
  );

  return (
    item.modelKind ??
    model?.kind ??
    ""
  );
}

function getItemModelLabel(
  item: CartDisplayItem,
): string {
  const modelKind =
    getItemModelKind(item);

  if (modelKind === "alcohol") {
    return getAlcoholModelLabel(item);
  }

  return getApparelModelLabel(item);
}

export function toOrderConfirmedItemViewModel(
  item: CartDisplayItem,
): OrderConfirmedItemViewModel {
  const price =
    getCartItemPrice(item);

  const lineAmount =
    price === null
      ? null
      : price * item.qty;

  return {
    itemKey: item.itemKey,
    title: getItemTitle(item),
    modelLabel:
      getItemModelLabel(item),
    qty: item.qty,
    lineAmount,
  };
}

export function toOrderConfirmedItemViewModels(
  items: CartDisplayItem[],
): OrderConfirmedItemViewModel[] {
  return items.map(
    toOrderConfirmedItemViewModel,
  );
}