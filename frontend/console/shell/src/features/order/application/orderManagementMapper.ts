// frontend/console/order/src/application/orderManagementMapper.ts
import { OrderItemInventoryRowDTO } from "../infrastructure/repostiroty";

export type OrderManagementRow = {
  orderId: string;
  listId: string;

  productName: string;
  tokenName: string;

  inventoryId: string;

  avatarName: string;
  avatarId: string;

  createdAt: string;
  transferred: boolean;
};

export function mapOrderItemInventoryRowToOrderManagementRow(
  x: OrderItemInventoryRowDTO,
): OrderManagementRow {
  return {
    orderId: String((x as any).orderId ?? ""),

    // /orders/items response の正フィールド: listReadableId
    listId: String((x as any).listReadableId ?? ""),

    inventoryId: String((x as any).inventoryId ?? ""),

    productName: String((x as any).productName ?? ""),
    tokenName: String((x as any).tokenName ?? ""),

    // /orders/items response の正フィールド: avatarName
    avatarName: String((x as any).avatarName ?? ""),
    avatarId: String((x as any).avatarId ?? ""),

    createdAt: String((x as any).createdAt ?? ""),
    transferred: Boolean((x as any).transferred),
  };
}

export function mapOrderItemInventoryRowsToOrderManagementRows(
  rows: OrderItemInventoryRowDTO[] | null | undefined,
): OrderManagementRow[] {
  return (rows ?? []).map(mapOrderItemInventoryRowToOrderManagementRow);
}