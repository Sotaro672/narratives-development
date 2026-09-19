// frontend/console/shell/src/shared/util/currency.ts

const jpyFormatter = new Intl.NumberFormat("ja-JP", {
  style: "currency",
  currency: "JPY",
  maximumFractionDigits: 0,
});

/**
 * 金額を日本円表記へ整形する。
 *
 * @example
 * formatJPY(1000); // "￥1,000"
 */
export function formatJPY(value: number): string {
  if (!Number.isFinite(value)) {
    return "-";
  }

  return jpyFormatter.format(value);
}