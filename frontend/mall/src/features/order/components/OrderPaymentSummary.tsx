// frontend/mall/src/features/order/components/OrderPaymentSummary.tsx

import SectionHeader from "../../../components/ui/SectionHeader";
import { formatDateTime } from "../../../components/utils/date";
import type { OrderDetail } from "../../shared/types/orderDetailTypes";
import { formatAmount } from "../../wallet/utils/format";

type OrderPaymentSummaryProps = {
  order: OrderDetail;
};

function getRefundStatusLabel(order: OrderDetail): string {
  const activeItems = order.items.filter((item) => !item.isCancelled);
  const allReturnCompleted =
    activeItems.length > 0 &&
    activeItems.every((item) => item.isReturnCompleted);

  if (allReturnCompleted) return "返金済み";

  const partiallyReturnCompleted = activeItems.some(
    (item) => item.isReturnCompleted,
  );

  if (partiallyReturnCompleted) return "一部返金済み";

  switch (order.refundStatus) {
    case "pending":
      return "返金処理中";
    case "requires_action":
      return "返金対応待ち";
    case "succeeded":
      return "返金済み";
    case "failed":
      return "返金失敗";
    case "canceled":
      return "返金キャンセル";
    case "none":
    default:
      return "未返金";
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
    <div className="page-card">
      <SectionHeader title="お支払い" titleAs="h2" />

      <dl className="order-detail-page__detail-list">
        <div className="order-detail-page__detail-row">
          <dt>商品小計</dt>
          <dd>{formatAmount(order.subtotalAmount)}</dd>
        </div>

        <div className="order-detail-page__detail-row">
          <dt>配送料</dt>
          <dd>{formatAmount(order.shippingAmount)}</dd>
        </div>

        <div className="order-detail-page__detail-row">
          <dt>消費税</dt>
          <dd>{formatAmount(order.consumptionTax)}</dd>
        </div>

        <div className="order-detail-page__detail-row order-detail-page__detail-row--total">
          <dt>合計</dt>
          <dd>{formatAmount(order.totalAmount)}</dd>
        </div>

        <div className="order-detail-page__detail-row">
          <dt>決済状況</dt>
          <dd>{order.paid ? "決済済み" : "未決済"}</dd>
        </div>

        {hasReturnInProgress ? (
          <div className="order-detail-page__detail-row">
            <dt>返金状況</dt>
            <dd>{getRefundStatusLabel(order)}</dd>
          </div>
        ) : null}

        {order.refundedAmount > 0 ? (
          <div className="order-detail-page__detail-row">
            <dt>返金額</dt>
            <dd>{formatAmount(order.refundedAmount)}</dd>
          </div>
        ) : null}

        {order.refundedAt ? (
          <div className="order-detail-page__detail-row">
            <dt>返金日時</dt>
            <dd>{formatDateTime(order.refundedAt)}</dd>
          </div>
        ) : null}
      </dl>
    </div>
  );
}