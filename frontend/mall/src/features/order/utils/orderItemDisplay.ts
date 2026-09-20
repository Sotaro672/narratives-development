// frontend/mall/src/features/order/utils/orderItemDisplay.ts

import type { OrderDetailItem } from "../../shared/types/orderDetailTypes";

export type OrderItemMeta = {
  label: string;
  value: string;
};

export function getProductTitle(item: OrderDetailItem): string {
  return item.productName || item.tokenName || "商品";
}

export function getFallbackInitial(value?: string): string {
  const trimmed = value?.trim() || "";

  if (!trimmed) {
    return "?";
  }

  return trimmed.slice(0, 1).toUpperCase();
}

export function getModelMetaItems(item: OrderDetailItem): OrderItemMeta[] {
  const metaItems: OrderItemMeta[] = [];

  if (item.modelNumber) {
    metaItems.push({
      label: "モデル番号",
      value: item.modelNumber,
    });
  }

  if (item.size) {
    metaItems.push({
      label: "サイズ",
      value: item.size,
    });
  }

  if (item.color?.name) {
    metaItems.push({
      label: "カラー",
      value: item.color.name,
    });
  }

  if (item.volumeValue !== undefined && item.volumeValue !== null) {
    metaItems.push({
      label: "容量",
      value: `${item.volumeValue}${item.volumeUnit || ""}`,
    });
  }

  return metaItems;
}

export function getMeasurementLabel(key: string): string {
  switch (key) {
    case "length":
      return "着丈";
    case "shoulder":
      return "肩幅";
    case "chest":
      return "身幅";
    case "sleeve":
      return "袖丈";
    case "waist":
      return "ウエスト";
    case "rise":
      return "股上";
    case "inseam":
      return "股下";
    case "hem":
      return "裾幅";
    default:
      return key;
  }
}

export function getMeasurementEntries(
  item: OrderDetailItem,
): Array<[string, number]> {
  if (!item.measurements || typeof item.measurements !== "object") {
    return [];
  }

  return Object.entries(item.measurements).filter(
    (entry): entry is [string, number] => Number.isFinite(entry[1]),
  );
}