// frontend/console/shell/src/features/order/application/orderManagementFilter.ts

import { getOrderStatusLabel } from "../../../shared/types/order";
import type { OrderItemInventoryRowDTO } from "../infrastructure/repository";

export type OrderManagementFilters = {
  listIds: string[];
  productNames: string[];
  tokenNames: string[];
  statuses: string[];
};

export function getOrderManagementStatus(
  order: OrderItemInventoryRowDTO,
): string {
  if (order.isCancelled) return "キャンセル";
  if (order.isReturnCompleted) return "返品済";
  if (order.isReturnRequested) return "返品対応中";
  if (order.transferred) return "移譲済";

  return getOrderStatusLabel(order.paid);
}

function matchesSelected(
  value: string | undefined,
  selected: string[],
): boolean {
  return selected.length === 0 || selected.includes(value ?? "");
}

export function filterOrderRows(
  rows: OrderItemInventoryRowDTO[],
  filters: OrderManagementFilters,
): OrderItemInventoryRowDTO[] {
  const {
    listIds,
    productNames,
    tokenNames,
    statuses,
  } = filters;

  if (
    listIds.length === 0 &&
    productNames.length === 0 &&
    tokenNames.length === 0 &&
    statuses.length === 0
  ) {
    return rows;
  }

  return rows.filter((row) =>
    matchesSelected(row.listReadableId, listIds) &&
    matchesSelected(row.productName, productNames) &&
    matchesSelected(row.tokenName, tokenNames) &&
    matchesSelected(getOrderManagementStatus(row), statuses)
  );
}