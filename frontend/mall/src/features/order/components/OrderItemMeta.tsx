// frontend/mall/src/features/order/components/OrderItemMeta.tsx

import type { OrderDetailItem } from "../../shared/types/orderDetailTypes";

type OrderItemMetaProps = {
  item: OrderDetailItem;
};

type MetaItem = {
  label: string;
  value: string;
};

function getModelMetaItems(item: OrderDetailItem): MetaItem[] {
  const metaItems: MetaItem[] = [];

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

function getMeasurementLabel(key: string): string {
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

export default function OrderItemMeta({ item }: OrderItemMetaProps) {
  const metaItems = getModelMetaItems(item);
  const measurements =
    item.measurements && typeof item.measurements === "object"
      ? Object.entries(item.measurements).filter(([, value]) =>
          Number.isFinite(value),
        )
      : [];

  if (metaItems.length === 0 && measurements.length === 0) {
    return null;
  }

  return (
    <div className="order-detail-page__model-meta">
      <dl className="order-detail-page__item-meta">
        {metaItems.map((meta) => (
          <div
            key={meta.label}
            className="order-detail-page__item-meta-row"
          >
            <dt>{meta.label}</dt>
            <dd>{meta.value}</dd>
          </div>
        ))}

        {measurements.map(([key, value]) => (
          <div
            key={key}
            className="order-detail-page__item-meta-row"
          >
            <dt>{getMeasurementLabel(key)}</dt>
            <dd>{value} mm</dd>
          </div>
        ))}
      </dl>
    </div>
  );
}