// frontend/amol/src/features/resale/constants/resaleConditions.ts

import {
  RESALE_CONDITIONS,
  type ResaleCondition,
} from "../../shared/types/resale";

export const DEFAULT_RESALE_CONDITION:
  ResaleCondition =
    "未使用に近い";

export const RESALE_CONDITION_OPTIONS =
  RESALE_CONDITIONS.map(
    (condition) => ({
      value: condition,
      label: condition,
    }),
  );

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

/**
 * 値を再販商品の状態へ正規化する。
 *
 * 未定義または不正な値の場合は、
 * デフォルトの商品状態を返す。
 */
export function normalizeResaleCondition(
  value: unknown,
): ResaleCondition {
  return isResaleCondition(value)
    ? value
    : DEFAULT_RESALE_CONDITION;
}