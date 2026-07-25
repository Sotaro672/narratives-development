// frontend/console/order/src/application/orderDetailCalculations.ts
import { OrderDetailItemDTO } from "./orderDetailBuilder";

export function formatJPY(n: number | null | undefined): string {
  const v = typeof n === "number" && Number.isFinite(n) ? n : 0;
  return `¥${v.toLocaleString()}`;
}

export function calculateOrderQuantity(items: OrderDetailItemDTO[]): number {
  return items.reduce((sum, it) => sum + (Number(it?.qty ?? 0) || 0), 0);
}

export function calculateOrderTotalPrice(items: OrderDetailItemDTO[]): number {
  return items.reduce(
    (sum, it) =>
      sum + (Number(it?.price ?? 0) || 0) * (Number(it?.qty ?? 0) || 0),
    0,
  );
}

export function hasTransferredItem(items: OrderDetailItemDTO[]): boolean {
  return items.some((it) => Boolean(it?.transferred));
}

export function extractListIds(items: OrderDetailItemDTO[]): string[] {
  const set = new Set<string>();

  for (const it of items) {
    const v = String(it?.listId ?? "").trim();
    if (v) {
      set.add(v);
    }
  }

  return Array.from(set);
}