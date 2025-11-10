// frontend/order/src/infrastructure/mockdata/mockdata.tsx

import type { Order } from "../../../../shell/src/shared/types/order";
import type { OrderItem } from "../../../../shell/src/shared/types/orderItem";

/**
 * モック用 OrderItem データ
 * backend/internal/domain/orderItem/entity.go に準拠。
 *
 * - quantity は 1 以上
 * - 各 ID は非空文字列
 */
export const ORDER_ITEMS: OrderItem[] = [
  {
    id: "item_001",
    modelId: "model_001",
    saleId: "sale_001",
    inventoryId: "inv_001",
    quantity: 2,
  },
  {
    id: "item_002",
    modelId: "model_002",
    saleId: "sale_001",
    inventoryId: "inv_002",
    quantity: 1,
  },
  {
    id: "item_003",
    modelId: "model_003",
    saleId: "sale_002",
    inventoryId: "inv_003",
    quantity: 1,
  },
  {
    id: "item_004",
    modelId: "model_004",
    saleId: "sale_002",
    inventoryId: "inv_004",
    quantity: 3,
  },
  {
    id: "item_005",
    modelId: "model_005",
    saleId: "sale_002",
    inventoryId: "inv_005",
    quantity: 1,
  },
];

/**
 * モック用 Order データ
 * frontend/shell/src/shared/types/order.ts に準拠。
 *
 * - items は OrderItem の「ID文字列配列」
 */
export const ORDERS: Order[] = [
  {
    id: "order_0001",
    orderNumber: "ORD-2024-0001",
    status: "paid",
    userId: "user_001",
    shippingAddressId: "ship_001",
    billingAddressId: "bill_001",
    listId: "list_001",
    items: ["item_001", "item_002"],
    invoiceId: "inv_001",
    paymentId: "pay_001",
    fulfillmentId: "ful_001",
    trackingId: "track_001",
    transfferedDate: null,
    createdAt: "2024-03-21T10:00:00Z",
    updatedAt: "2024-03-21T10:00:00Z",
    updatedBy: "system",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "order_0002",
    orderNumber: "ORD-2024-0002",
    status: "transferred",
    userId: "user_002",
    shippingAddressId: "ship_002",
    billingAddressId: "bill_002",
    listId: "list_002",
    items: ["item_003", "item_004", "item_005"],
    invoiceId: "inv_002",
    paymentId: "pay_002",
    fulfillmentId: "ful_002",
    trackingId: "track_002",
    transfferedDate: "2024-03-21T13:30:00Z",
    createdAt: "2024-03-20T09:00:00Z",
    updatedAt: "2024-03-21T13:30:00Z",
    updatedBy: "system",
    deletedAt: null,
    deletedBy: null,
  },
];
