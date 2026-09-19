// frontend/console/shell/src/features/order/presentation/formatter/formatOrderVolume.ts

import type { OrderDetailItemDTO } from "../hooks/useOrderDetail";

/**
 * 注文アイテムの容量を表示用文字列へ整形する。
 *
 * - volumeValue が未定義の場合は "-"
 * - volumeUnit が存在する場合は値の末尾へ単位を付与
 * - volumeUnit が空の場合は値のみ表示
 */
export function formatOrderVolume(item: OrderDetailItemDTO): string {
  if (item.volumeValue === undefined) {
    return "-";
  }

  const unit = item.volumeUnit?.trim();

  return unit
    ? `${item.volumeValue}${unit}`
    : String(item.volumeValue);
}