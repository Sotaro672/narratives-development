// frontend/order/src/infrastructure/mockdata/order_mockdata.ts
import type { Order } from "../../../../shell/src/shared/types/order";

/**
 * モック用 Order データ
 * frontend/shell/src/shared/types/order.ts に準拠。
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
    transferredDate: null,
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
    transferredDate: "2024-03-21T13:30:00Z",
    createdAt: "2024-03-20T09:00:00Z",
    updatedAt: "2024-03-21T13:30:00Z",
    updatedBy: "system",
    deletedAt: null,
    deletedBy: null,
  },
];
