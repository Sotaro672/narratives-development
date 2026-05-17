import type { CartDisplayItem } from "../../cart/types";
import { getModelPrice, getModelVariation } from "../../cart/utils/cartUtils";
import type { OrderConfirmedItemViewModel } from "../types";

type CartItemWithDirectFields = CartDisplayItem & {
  price?: number;
  productName?: string;
  title?: string;
  modelKind?: string;
  kind?: string;
  modelNumber?: string;
  modelLabel?: string;
  volumeValue?: number;
  volumeUnit?: string;
  colorName?: string;
  size?: string;
};

type ModelWithExtraFields = {
  colorName?: string;
  size?: string;
  modelKind?: string;
  kind?: string;
  modelNumber?: string;
  modelLabel?: string;
  volumeValue?: number;
  volumeUnit?: string;
  price?: number;
};

function getItemTitle(item: CartDisplayItem): string {
  const typedItem = item as CartItemWithDirectFields;
  const catalog = item.catalog;

  return (
    typedItem.productName ||
    typedItem.title ||
    catalog?.productBlueprint.productName ||
    catalog?.list.title ||
    "商品名未設定"
  );
}

function getItemPrice(item: CartDisplayItem): number | null {
  const typedItem = item as CartItemWithDirectFields;

  if (typeof typedItem.price === "number") {
    return typedItem.price;
  }

  return getModelPrice(item.catalog, item.modelId);
}

function getAlcoholModelLabel(item: CartDisplayItem): string {
  const typedItem = item as CartItemWithDirectFields;
  const model = getModelVariation(item.catalog, item.modelId) as
    | ModelWithExtraFields
    | null;

  if (typedItem.modelLabel) {
    return typedItem.modelLabel;
  }

  if (model?.modelLabel) {
    return model.modelLabel;
  }

  const modelNumber = typedItem.modelNumber ?? model?.modelNumber ?? "";
  const volumeValue = typedItem.volumeValue ?? model?.volumeValue;
  const volumeUnit = typedItem.volumeUnit ?? model?.volumeUnit ?? "";

  const volumeLabel =
    typeof volumeValue === "number" && volumeUnit
      ? `${volumeValue}${volumeUnit}`
      : "";

  return [modelNumber, volumeLabel].filter(Boolean).join(" / ");
}

function getApparelModelLabel(item: CartDisplayItem): string {
  const typedItem = item as CartItemWithDirectFields;
  const model = getModelVariation(item.catalog, item.modelId) as
    | ModelWithExtraFields
    | null;

  const colorName = typedItem.colorName ?? model?.colorName ?? "";
  const size = typedItem.size ?? model?.size ?? "";

  return [
    colorName ? `カラー: ${colorName}` : "",
    size ? `サイズ: ${size}` : "",
  ]
    .filter(Boolean)
    .join(" / ");
}

function getItemModelKind(item: CartDisplayItem): string {
  const typedItem = item as CartItemWithDirectFields;
  const model = getModelVariation(item.catalog, item.modelId) as
    | ModelWithExtraFields
    | null;

  return (
    typedItem.modelKind ??
    typedItem.kind ??
    model?.modelKind ??
    model?.kind ??
    ""
  );
}

function getItemModelLabel(item: CartDisplayItem): string {
  const modelKind = getItemModelKind(item);

  if (modelKind === "alcohol") {
    return getAlcoholModelLabel(item);
  }

  return getApparelModelLabel(item);
}

export function toOrderConfirmedItemViewModel(
  item: CartDisplayItem,
): OrderConfirmedItemViewModel {
  const price = getItemPrice(item);
  const lineAmount = price === null ? null : price * item.qty;

  return {
    itemKey: item.itemKey,
    title: getItemTitle(item),
    modelLabel: getItemModelLabel(item),
    qty: item.qty,
    lineAmount,
  };
}

export function toOrderConfirmedItemViewModels(
  items: CartDisplayItem[],
): OrderConfirmedItemViewModel[] {
  return items.map(toOrderConfirmedItemViewModel);
}