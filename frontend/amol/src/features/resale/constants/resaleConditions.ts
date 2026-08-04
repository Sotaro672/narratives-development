// frontend/amol/src/features/resale/constants/resaleConditions.ts

export const RESALE_CONDITIONS = [
  "新品・未使用",
  "未使用に近い",
  "目立った傷や汚れなし",
  "やや傷や汚れあり",
  "傷や汚れあり",
] as const;

export type ResaleCondition =
  (typeof RESALE_CONDITIONS)[number];

export const DEFAULT_RESALE_CONDITION:
  ResaleCondition =
    "未使用に近い";

/**
 * 値が再販商品の状態として有効か判定する。
 */
export function isResaleCondition(
  value: unknown,
): value is ResaleCondition {
  return (
    typeof value === "string" &&
    (
      RESALE_CONDITIONS as
        readonly string[]
    ).includes(value)
  );
}