// frontend/console/order/src/application/orderManagementSort.ts
import { OrderManagementRow } from "./orderManagementMapper";

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
    const aTime = a.createdAt;
    const bTime = b.createdAt;

    const aTs =
      aTime && !Number.isNaN(Date.parse(aTime)) ? Date.parse(aTime) : null;
    const bTs =
      bTime && !Number.isNaN(Date.parse(bTime)) ? Date.parse(bTime) : null;

    if (aTs === null && bTs === null) return 0;
    if (aTs === null) return direction === "asc" ? 1 : -1;
    if (bTs === null) return direction === "asc" ? -1 : 1;

    return direction === "asc" ? aTs - bTs : bTs - aTs;
  });
}