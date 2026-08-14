// frontend/console/shell/src/features/order/application/orderDetailCalculations.ts

import type { OrderDetailItemDTO } from "../infrastructure/repository";

export function formatJPY(value: number): string {
  return `¥${value.toLocaleString()}`;
}

export function calculateOrderQuantity(items: OrderDetailItemDTO[]): number {
  return items.reduce((total, item) => total + item.qty, 0);
}

export function calculateOrderTotalPrice(items: OrderDetailItemDTO[]): number {
  return items.reduce((total, item) => total + item.price * item.qty, 0);
}

export function hasTransferredItem(items: OrderDetailItemDTO[]): boolean {
  return items.some((item) => item.transferred);
}

export function extractListIds(items: OrderDetailItemDTO[]): string[] {
  const listReadableIds = new Set<string>();

  for (const item of items) {
    if (item.listReadableId) {
      listReadableIds.add(item.listReadableId);
    }
  }

  return Array.from(listReadableIds);
}