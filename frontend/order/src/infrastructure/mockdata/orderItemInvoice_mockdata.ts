// frontend\order\src\infrastructure\mockdata\orderItemInvoice_mockdata.ts
import type { OrderItemInvoice } from "../../../../shell/src/shared/types/invoice";

/**
 * モック用 OrderItemInvoice データ
 * backend/internal/domain/invoice/entity.go の OrderItemInvoice に準拠。
 */
export const ORDER_ITEM_INVOICES: OrderItemInvoice[] = [
  { id: "oii_001", orderItemId: "item_001", unitPrice: 6000, totalPrice: 12000, createdAt: "2024-03-21T10:00:00Z", updatedAt: "2024-03-21T10:00:00Z" },
  { id: "oii_002", orderItemId: "item_002", unitPrice: 8000, totalPrice: 8000, createdAt: "2024-03-21T10:00:00Z", updatedAt: "2024-03-21T10:00:00Z" },
  { id: "oii_003", orderItemId: "item_003", unitPrice: 5000, totalPrice: 5000, createdAt: "2024-03-20T09:00:00Z", updatedAt: "2024-03-21T13:30:00Z" },
  { id: "oii_004", orderItemId: "item_004", unitPrice: 7000, totalPrice: 21000, createdAt: "2024-03-20T09:00:00Z", updatedAt: "2024-03-21T13:30:00Z" },
  { id: "oii_005", orderItemId: "item_005", unitPrice: 9000, totalPrice: 9000, createdAt: "2024-03-20T09:00:00Z", updatedAt: "2024-03-21T13:30:00Z" },
];
