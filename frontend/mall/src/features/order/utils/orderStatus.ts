// frontend/mall/src/features/order/utils/orderStatus.ts

import type {
  OrderDetail,
  OrderDetailItem,
} from "../../shared/types/orderDetailTypes";

export function getOrderStatusLabel(order: OrderDetail): string {
  if (order.items.length === 0) return "商品なし";

  const activeItems = order.items.filter((item) => !item.isCancelled);
  if (activeItems.length === 0) return "キャンセル済み";

  const hasCancelledItem = activeItems.length !== order.items.length;
  if (hasCancelledItem) return "一部キャンセル済み";

  const allReturnCompleted = activeItems.every((item) => item.isReturnCompleted);
  if (allReturnCompleted) return "返品済み";

  const partiallyReturnCompleted = activeItems.some(
    (item) => item.isReturnCompleted,
  );
  if (partiallyReturnCompleted) return "一部返品済み";

  const allReturnRequested = activeItems.every((item) => item.isReturnRequested);
  if (allReturnRequested) return "返品申請済み";

  const partiallyReturnRequested = activeItems.some(
    (item) => item.isReturnRequested,
  );
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

export function getRefundStatusLabel(order: OrderDetail): string {
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

export function getItemStatusLabel(item: OrderDetailItem): string {
  if (item.isCancelled) return "キャンセル済み";
  if (item.isReturnCompleted) return "返品済み";
  if (item.isReturnRequested) return "返品申請済み";
  if (item.transferred) return "受け取り済み";
  if (item.isDispatched) return "発送済み";
  return "発送前";
}