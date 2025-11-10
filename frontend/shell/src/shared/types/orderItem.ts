// frontend/shell/src/shared/types/orderItem.ts
// Generated from frontend/order/src/domain/entity/orderItem.ts
// backend/internal/domain/orderItem/entity.go に準拠した共通型定義

/**
 * OrderItem
 * - backend/internal/domain/orderItem/entity.go の構造を反映
 * - quantity: 注文数量（1以上）
 */
export interface OrderItem {
  id: string;
  modelId: string;
  saleId: string;
  inventoryId: string;
  quantity: number;
}

/**
 * 定数ポリシー
 * backend 側の MinQuantity, MaxQuantity と同期
 */
export const ORDER_ITEM_MIN_QUANTITY = 1;
export const ORDER_ITEM_MAX_QUANTITY = 0; // 0 = 上限なし

/**
 * バリデーション関数
 * - ID系がすべて非空文字列
 * - 数量が有効範囲内
 */
export function isValidOrderItem(item: OrderItem): boolean {
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
 * 数量を設定するユーティリティ
 * @returns 更新後の OrderItem または null（不正値）
 */
export function setOrderItemQuantity(
  item: OrderItem,
  quantity: number,
): OrderItem | null {
  if (quantity < ORDER_ITEM_MIN_QUANTITY) return null;
  if (
    ORDER_ITEM_MAX_QUANTITY > 0 &&
    quantity > ORDER_ITEM_MAX_QUANTITY
  ) {
    return null;
  }
  return { ...item, quantity };
}

/**
 * 数量を増減するユーティリティ
 * @returns 更新後の OrderItem または null（範囲外の場合）
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
  ) {
    return null;
  }
  return { ...item, quantity: newQuantity };
}

/**
 * Inventory ID を再割当て
 */
export function reassignOrderItemInventory(
  item: OrderItem,
  inventoryId: string,
): OrderItem | null {
  if (!inventoryId?.trim()) return null;
  return { ...item, inventoryId: inventoryId.trim() };
}

/**
 * Sale ID を再割当て
 */
export function reassignOrderItemSale(
  item: OrderItem,
  saleId: string,
): OrderItem | null {
  if (!saleId?.trim()) return null;
  return { ...item, saleId: saleId.trim() };
}

/**
 * Model ID を更新
 */
export function updateOrderItemModel(
  item: OrderItem,
  modelId: string,
): OrderItem | null {
  if (!modelId?.trim()) return null;
  return { ...item, modelId: modelId.trim() };
}
