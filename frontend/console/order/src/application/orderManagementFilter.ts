// frontend/console/order/src/application/orderManagementFilter.ts
import { OrderManagementRow } from "./orderManagementMapper";

export type TokenFilterValue = "移譲済" | "未移譲";

export function tokenLabelFromTransferred(
  transferred: boolean,
): TokenFilterValue {
  return transferred ? "移譲済" : "未移譲";
}

export function filterOrderRowsByToken(
  rows: OrderManagementRow[],
  tokenFilter: TokenFilterValue[],
): OrderManagementRow[] {
  if (tokenFilter.length === 0) {
    return rows;
  }

  return rows.filter((row) => {
    const label = tokenLabelFromTransferred(row.transferred);
    return tokenFilter.includes(label);
  });
}