// frontend/mall/src/features/order/components/OrderDetailSummary.tsx

import { formatDateTime } from "../../../components/utils/date";
import type { OrderDetail } from "../../shared/types/orderDetailTypes";

type OrderDetailSummaryProps = {
  order: OrderDetail;
  error?: string | null;
  tradeNavigationError?: string | null;
};

function getOrderStatusLabel(order: OrderDetail): string {
  if (order.items.length === 0) return "商品なし";

  const activeItems = order.items.filter((item) => !item.isCancelled);
  if (activeItems.length === 0) return "キャンセル済み";

  const hasCancelledItem = activeItems.length !== order.items.length;
  if (hasCancelledItem) return "一部キャンセル済み";

  const allReturnCompleted = activeItems.every((item) => item.isReturnCompleted);
  if (allReturnCompleted) return "返品済み";

  const partiallyReturnCompleted = activeItems.some((item) => item.isReturnCompleted);
  if (partiallyReturnCompleted) return "一部返品済み";

  const allReturnRequested = activeItems.every((item) => item.isReturnRequested);
  if (allReturnRequested) return "返品申請済み";

  const partiallyReturnRequested = activeItems.some((item) => item.isReturnRequested);
  if (partiallyReturnRequested) return "一部返品申請済み";

  const allTransferred = activeItems.every((item) => item.transferred);
  if (allTransferred) return "受け取り済み";

  const partiallyTransferred = activeItems.some((item) => item.transferred);
  if (partiallyTransferred) return "一部受け取り済み";

  const allDispatched = activeItems.every((item) => item.isDispatched);
  if (allDispatched) return "発送済み";

  const partiallyDispatched = activeItems.some((item) => item.isDispatched);
  if (partiallyDispatched) return "一部発送済み";

  return "発送前";
}

export default function OrderDetailSummary({
  order,
  error = null,
  tradeNavigationError = null,
}: OrderDetailSummaryProps) {
  return (
    <div className="page-card order-detail-page__summary-card">
      <div className="order-detail-page__summary-header">
        <div>
          <p className="order-detail-page__date">
            注文日時: {order.createdAt ? formatDateTime(order.createdAt) : "-"}
          </p>

          <h1 className="order-detail-page__order-id">
            注文ID: {order.id}
          </h1>
        </div>

        <span className="order-detail-page__status">
          {getOrderStatusLabel(order)}
        </span>
      </div>

      {error ? (
        <p className="page-card__text" role="alert">
          {error}
        </p>
      ) : null}

      {tradeNavigationError ? (
        <p className="page-card__text" role="alert">
          {tradeNavigationError}
        </p>
      ) : null}
    </div>
  );
}