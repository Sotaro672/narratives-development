// frontend/order/src/infrastructure/mockdata/invoice_mockdata.ts
import type { Invoice } from "../../../../shell/src/shared/types/invoice";
import { ORDER_ITEM_INVOICES } from "./orderItemInvoice_mockdata";

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
    totalAmount: 21600,
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
