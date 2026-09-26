// frontend/mall/src/features/order/components/OrderDetailSummary.tsx

import Alert from "../../../components/ui/Alert";
import Badge from "../../../components/ui/Badge";
import SectionHeader from "../../../components/ui/SectionHeader";
import { formatDateTime } from "../../../components/utils/date";
import type { OrderDetail } from "../../shared/types/orderDetailTypes";
import { getOrderStatusLabel } from "../util/orderStatus";

type OrderDetailSummaryProps = {
  order: OrderDetail;
  error?: string | null;
  tradeNavigationError?: string | null;
};

export default function OrderDetailSummary({
  order,
  error = null,
  tradeNavigationError = null,
}: OrderDetailSummaryProps) {
  return (
    <section className="order-detail-page__section order-detail-page__summary">
      <SectionHeader
        className="ui-section-header--title-sm"
        title={`注文ID: ${order.id}`}
        titleAs="h1"
        eyebrow={`注文日時: ${order.createdAt ? formatDateTime(order.createdAt) : "-"}`}
        right={
          <Badge variant="neutral" size="md">
            {getOrderStatusLabel(order)}
          </Badge>
        }
      />

      {error ? <Alert variant="error">{error}</Alert> : null}

      {tradeNavigationError ? (
        <Alert variant="error">{tradeNavigationError}</Alert>
      ) : null}
    </section>
  );
}