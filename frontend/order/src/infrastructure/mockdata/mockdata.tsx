// frontend/order/src/infrastructure/mockdata/mockdata.tsx

import type { Order } from "../../../../shell/src/shared/types/order";
import type { OrderItem } from "../../../../shell/src/shared/types/orderItem";
import type {
  Invoice,
  OrderItemInvoice,
} from "../../../../shell/src/shared/types/invoice";

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

/**
 * モック用 OrderItemInvoice データ
 * backend/internal/domain/invoice/entity.go の OrderItemInvoice に準拠。
 */
export const ORDER_ITEM_INVOICES: OrderItemInvoice[] = [
  {
    id: "oii_001",
    orderItemId: "item_001",
    unitPrice: 6000,
    totalPrice: 12000,
    createdAt: "2024-03-21T10:00:00Z",
    updatedAt: "2024-03-21T10:00:00Z",
  },
  {
    id: "oii_002",
    orderItemId: "item_002",
    unitPrice: 8000,
    totalPrice: 8000,
    createdAt: "2024-03-21T10:00:00Z",
    updatedAt: "2024-03-21T10:00:00Z",
  },
  {
    id: "oii_003",
    orderItemId: "item_003",
    unitPrice: 5000,
    totalPrice: 5000,
    createdAt: "2024-03-20T09:00:00Z",
    updatedAt: "2024-03-21T13:30:00Z",
  },
  {
    id: "oii_004",
    orderItemId: "item_004",
    unitPrice: 7000,
    totalPrice: 21000,
    createdAt: "2024-03-20T09:00:00Z",
    updatedAt: "2024-03-21T13:30:00Z",
  },
  {
    id: "oii_005",
    orderItemId: "item_005",
    unitPrice: 9000,
    totalPrice: 9000,
    createdAt: "2024-03-20T09:00:00Z",
    updatedAt: "2024-03-21T13:30:00Z",
  },
];

/**
 * モック用 Invoice データ
 * frontend/shell/src/shared/types/invoice.ts に準拠。
 */
export const INVOICES: Invoice[] = [
  {
    orderId: "order_0001",
    orderItemInvoices: [ORDER_ITEM_INVOICES[0], ORDER_ITEM_INVOICES[1]],
    subtotal: 20000,
    discountAmount: 1000,
    taxAmount: 1800,
    shippingCost: 800,
    totalAmount: 21600, // subtotal - discount + tax + shipping
    currency: "JPY",
    createdAt: "2024-03-21T10:00:00Z",
    updatedAt: "2024-03-21T10:00:00Z",
    billingAddressId: "bill_001",
  },
  {
    orderId: "order_0002",
    orderItemInvoices: [
      ORDER_ITEM_INVOICES[2],
      ORDER_ITEM_INVOICES[3],
      ORDER_ITEM_INVOICES[4],
    ],
    subtotal: 35000,
    discountAmount: 0,
    taxAmount: 3500,
    shippingCost: 1200,
    totalAmount: 39700,
    currency: "JPY",
    createdAt: "2024-03-20T09:00:00Z",
    updatedAt: "2024-03-21T13:30:00Z",
    billingAddressId: "bill_002",
  },
];
