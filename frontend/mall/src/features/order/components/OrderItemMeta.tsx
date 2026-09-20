// frontend/mall/src/features/order/components/OrderItemMeta.tsx

import InfoList, { InfoRow } from "../../../components/ui/InfoList";
import type { OrderDetailItem } from "../../shared/types/orderDetailTypes";
import {
  getMeasurementEntries,
  getMeasurementLabel,
  getModelMetaItems,
} from "../util/orderItemDisplay";

type OrderItemMetaProps = {
  item: OrderDetailItem;
};

export default function OrderItemMeta({ item }: OrderItemMetaProps) {
  const metaItems = getModelMetaItems(item);
  const measurements = getMeasurementEntries(item);

  if (metaItems.length === 0 && measurements.length === 0) {
    return null;
  }

  return (
    <InfoList>
      {metaItems.map((meta) => (
        <InfoRow key={meta.label} label={meta.label}>
          {meta.value}
        </InfoRow>
      ))}

      {measurements.map(([key, value]) => (
        <InfoRow key={key} label={getMeasurementLabel(key)}>
          {value} mm
        </InfoRow>
      ))}
    </InfoList>
  );
}