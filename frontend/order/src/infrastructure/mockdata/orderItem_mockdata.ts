// frontend/order/src/infrastructure/mockdata/orderItem_mockdata.ts
import type { OrderItem } from "../../../../shell/src/shared/types/orderItem";

/**
 * モック用 OrderItem データ
 * backend/internal/domain/orderItem/entity.go に準拠。
 */
export const ORDER_ITEMS: OrderItem[] = [
  { id: "item_001", modelId: "model_001", saleId: "sale_001", inventoryId: "inv_001", quantity: 2 },
  { id: "item_002", modelId: "model_002", saleId: "sale_001", inventoryId: "inv_002", quantity: 1 },
  { id: "item_003", modelId: "model_003", saleId: "sale_002", inventoryId: "inv_003", quantity: 1 },
  { id: "item_004", modelId: "model_004", saleId: "sale_002", inventoryId: "inv_004", quantity: 3 },
  { id: "item_005", modelId: "model_005", saleId: "sale_002", inventoryId: "inv_005", quantity: 1 },
];
