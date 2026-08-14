// frontend/console/order/src/application/orderManagementFilter.ts

import type { OrderItemInventoryRowDTO } from "../infrastructure/repository";

export type TokenFilterValue = "移譲済" | "未移譲";

export function tokenLabelFromTransferred(transferred: boolean): TokenFilterValue {
  return transferred ? "移譲済" : "未移譲";
}

export function filterOrderRowsByToken(
  rows: OrderItemInventoryRowDTO[],
  tokenFilter: TokenFilterValue[],
): OrderItemInventoryRowDTO[] {
  if (tokenFilter.length === 0) {
    return rows;
  }

  return rows.filter((row) =>
    tokenFilter.includes(tokenLabelFromTransferred(row.transferred)),
  );
}