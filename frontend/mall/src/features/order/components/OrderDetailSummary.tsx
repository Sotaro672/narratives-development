// frontend/mall/src/features/order/components/OrderDetailSummary.tsx

import Alert from "../../../components/ui/Alert";
import Badge from "../../../components/ui/Badge";
import Copy from "../../../components/ui/Copy";
import SectionHeader from "../../../components/ui/SectionHeader";
import { formatDateTime } from "../../../components/utils/date";
import type { OrderDetail } from "../../shared/types/orderDetailTypes";
import { getOrderStatusLabel } from "../util/orderStatus";

type OrderDetailSummaryProps = {
  order: OrderDetail;
  error?: string | null;
  tradeNavigationError?: string | null;
  showStatus?: boolean;
};

export default function OrderDetailSummary({
  order,
  error = null,
  tradeNavigationError = null,
  showStatus = true,
}: OrderDetailSummaryProps) {
  const handleCopyOrderId = async (): Promise<void> => {
    await navigator.clipboard.writeText(order.id);
  };

  return (
    <section className="order-detail-page__section order-detail-page__summary">
      <SectionHeader
        title={
          <span className="order-detail-page__order-id">
            <span className="order-detail-page__order-id-text">
              注文ID: {order.id}
            </span>

            <Copy
              onClick={handleCopyOrderId}
              ariaLabel="注文IDをコピー"
              title="注文IDをコピー"
              copiedLabel="コピーしました"
            />
          </span>
        }
        titleAs="h1"
        titleSize="sm"
        eyebrow={`注文日時: ${order.createdAt ? formatDateTime(order.createdAt) : "-"}`}
        right={
          showStatus ? (
            <Badge variant="neutral" size="md">
              {getOrderStatusLabel(order)}
            </Badge>
          ) : undefined
        }
      />

      {error ? <Alert variant="error">{error}</Alert> : null}

      {tradeNavigationError ? (
        <Alert variant="error">{tradeNavigationError}</Alert>
      ) : null}
    </section>
  );
}