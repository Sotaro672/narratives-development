// frontend/console/shell/src/features/order/application/orderDetailCalculations.ts

import type { OrderDetailItemDTO } from "./orderDetailBuilder";

export function formatJPY(
  value: number,
): string {
  const amount =
    Number.isFinite(value)
      ? value
      : 0;

  return `¥${amount.toLocaleString()}`;
}

export function calculateOrderQuantity(
  items: OrderDetailItemDTO[],
): number {
  return items.reduce(
    (total, item) =>
      total + item.qty,
    0,
  );
}

export function calculateOrderTotalPrice(
  items: OrderDetailItemDTO[],
): number {
  return items.reduce(
    (total, item) =>
      total +
      item.price * item.qty,
    0,
  );
}

export function hasTransferredItem(
  items: OrderDetailItemDTO[],
): boolean {
  return items.some(
    (item) => item.transferred,
  );
}

export function extractListIds(
  items: OrderDetailItemDTO[],
): string[] {
  const listIds =
    new Set<string>();

  for (const item of items) {
    const listId =
      item.listId?.trim();

    if (listId) {
      listIds.add(listId);
    }
  }

  return Array.from(listIds);
}