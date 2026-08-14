// frontend/console/order/src/application/orderManagementSort.ts

import type { OrderItemInventoryRowDTO } from "../infrastructure/repository";

export type OrderManagementRow = OrderItemInventoryRowDTO;
export type SortKey = "createdAt" | null;
export type SortDir = "asc" | "desc" | null;

export function sortOrderRows(
  rows: OrderManagementRow[],
  activeKey: SortKey,
  direction: SortDir,
): OrderManagementRow[] {
  if (!activeKey || !direction) {
    return rows;
  }

  if (activeKey !== "createdAt") {
    return rows;
  }

  return [...rows].sort((a, b) => {
    const aTs = a.createdAt ? Date.parse(a.createdAt) : NaN;
    const bTs = b.createdAt ? Date.parse(b.createdAt) : NaN;

    if (Number.isNaN(aTs) && Number.isNaN(bTs)) return 0;
    if (Number.isNaN(aTs)) return direction === "asc" ? 1 : -1;
    if (Number.isNaN(bTs)) return direction === "asc" ? -1 : 1;

    return direction === "asc" ? aTs - bTs : bTs - aTs;
  });
}