// frontend/mall/src/features/order/components/OrderPaymentSummary.tsx

import Badge from "../../../components/ui/Badge";
import Card from "../../../components/ui/Card";
import InfoList, { InfoRow } from "../../../components/ui/InfoList";
import SectionHeader from "../../../components/ui/SectionHeader";
import { formatDateTime } from "../../../components/utils/date";
import type { OrderDetail } from "../../shared/types/orderDetailTypes";
import { formatAmount } from "../../wallet/utils/format";
import { getRefundStatusLabel } from "../utils/orderStatus";

type OrderPaymentSummaryProps = {
  order: OrderDetail;
};

type BadgeVariant = "neutral" | "info" | "success" | "warning" | "danger";

function getRefundStatusVariant(order: OrderDetail): BadgeVariant {
  const activeItems = order.items.filter((item) => !item.isCancelled);

  if (
    activeItems.length > 0 &&
    activeItems.every((item) => item.isReturnCompleted)
  ) {
    return "success";
  }

  if (activeItems.some((item) => item.isReturnCompleted)) {
    return "warning";
  }

  switch (order.refundStatus) {
    case "succeeded":
      return "success";
    case "pending":
    case "requires_action":
      return "warning";
    case "failed":
    case "canceled":
      return "danger";
    case "none":
    default:
      return "neutral";
  }
}

export default function OrderPaymentSummary({
  order,
}: OrderPaymentSummaryProps) {
  const hasReturnInProgress = order.items.some(
    (item) =>
      !item.isCancelled &&
      item.isReturnRequested &&
      !item.isReturnCompleted,
  );

  return (
    <Card as="section">
      <SectionHeader title="お支払い" titleAs="h2" />

      <InfoList>
        <InfoRow label="商品小計">
          {formatAmount(order.subtotalAmount)}
        </InfoRow>

        <InfoRow label="配送料">
          {formatAmount(order.shippingAmount)}
        </InfoRow>

        <InfoRow label="消費税">
          {formatAmount(order.consumptionTax)}
        </InfoRow>

        <InfoRow label="合計" className="order-detail-page__payment-total">
          {formatAmount(order.totalAmount)}
        </InfoRow>

        <InfoRow label="決済状況">
          <Badge variant={order.paid ? "success" : "warning"} size="sm">
            {order.paid ? "決済済み" : "未決済"}
          </Badge>
        </InfoRow>

        {hasReturnInProgress ? (
          <InfoRow label="返金状況">
            <Badge variant={getRefundStatusVariant(order)} size="sm">
              {getRefundStatusLabel(order)}
            </Badge>
          </InfoRow>
        ) : null}

        {order.refundedAmount > 0 ? (
          <InfoRow label="返金額">
            {formatAmount(order.refundedAmount)}
          </InfoRow>
        ) : null}

        {order.refundedAt ? (
          <InfoRow label="返金日時">
            {formatDateTime(order.refundedAt)}
          </InfoRow>
        ) : null}
      </InfoList>
    </Card>
  );
}