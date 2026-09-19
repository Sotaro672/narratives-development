// frontend/console/shell/src/features/order/presentation/formatter/formatOrderItemValue.ts

function hasDisplayValue(value: unknown): boolean {
  if (value === null || value === undefined) {
    return false;
  }

  if (typeof value === "string") {
    return value.trim() !== "";
  }

  return true;
}

/**
 * 注文アイテムの表示用値を文字列へ整形する。
 *
 * - null / undefined / 空文字は "-"
 * - 配列は空要素を除外して ", " 区切り
 * - boolean は "あり" / "なし"
 * - unit 指定時は値の末尾へ単位を付与
 */
export function formatOrderItemValue(
  value: unknown,
  unit?: string,
): string {
  if (!hasDisplayValue(value)) {
    return "-";
  }

  if (Array.isArray(value)) {
    const joined = value
      .map((item) => String(item ?? "").trim())
      .filter(Boolean)
      .join(", ");

    return joined || "-";
  }

  if (typeof value === "boolean") {
    return value ? "あり" : "なし";
  }

  const text = String(value);

  return unit ? `${text}${unit}` : text;
}