// frontend/order/src/domain/entity/orderItem.ts
// backend/internal/domain/orderItem/entity.go に対応するフロントエンド側エンティティ定義

/**
 * OrderItem
 * backend/internal/domain/orderItem/entity.go の構造に準拠。
 * - 各 ID は非空文字列
 * - quantity は 1 以上
 */
export interface OrderItem {
  id: string;
  modelId: string;
  saleId: string;
  inventoryId: string;
  quantity: number;
}

/**
 * Policy（バックエンドと同期）
 */
export const ORDER_ITEM_MIN_QUANTITY = 1;
export const ORDER_ITEM_MAX_QUANTITY = 0; // 0 の場合は上限なし

/**
 * OrderItem のバリデーション
 * - id, modelId, saleId, inventoryId は非空
 * - quantity は MinQuantity 以上、MaxQuantity 以下（MaxQuantity > 0 の場合）
 */
export function validateOrderItem(item: OrderItem): boolean {
  if (!item.id?.trim()) return false;
  if (!item.modelId?.trim()) return false;
  if (!item.saleId?.trim()) return false;
  if (!item.inventoryId?.trim()) return false;
  if (item.quantity < ORDER_ITEM_MIN_QUANTITY) return false;
  if (
    ORDER_ITEM_MAX_QUANTITY > 0 &&
    item.quantity > ORDER_ITEM_MAX_QUANTITY
  ) {
    return false;
  }
  return true;
}

/**
 * Mutator: 数量を変更
 */
export function setOrderItemQuantity(
  item: OrderItem,
  q: number,
): OrderItem | null {
  if (q < ORDER_ITEM_MIN_QUANTITY) return null;
  if (ORDER_ITEM_MAX_QUANTITY > 0 && q > ORDER_ITEM_MAX_QUANTITY) return null;
  return { ...item, quantity: q };
}

/**
 * Mutator: 数量を増減
 */
export function incrementOrderItemQuantity(
  item: OrderItem,
  delta: number,
): OrderItem | null {
  const newQuantity = item.quantity + delta;
  if (newQuantity < ORDER_ITEM_MIN_QUANTITY) return null;
  if (
    ORDER_ITEM_MAX_QUANTITY > 0 &&
    newQuantity > ORDER_ITEM_MAX_QUANTITY
  )
    return null;
  return { ...item, quantity: newQuantity };
}

/**
 * Mutator: Inventory の再割当て
 */
export function reassignOrderItemInventory(
  item: OrderItem,
  inventoryId: string,
): OrderItem | null {
  if (!inventoryId?.trim()) return null;
  return { ...item, inventoryId: inventoryId.trim() };
}

/**
 * Mutator: Sale の再割当て
 */
export function reassignOrderItemSale(
  item: OrderItem,
  saleId: string,
): OrderItem | null {
  if (!saleId?.trim()) return null;
  return { ...item, saleId: saleId.trim() };
}

/**
 * Mutator: Model の更新
 */
export function updateOrderItemModel(
  item: OrderItem,
  modelId: string,
): OrderItem | null {
  if (!modelId?.trim()) return null;
  return { ...item, modelId: modelId.trim() };
}
