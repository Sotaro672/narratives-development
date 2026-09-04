// frontend/amol/src/features/shared/utils/typeGuards.ts

/**
 * unknown値がnullではないobjectかを判定します。
 *
 * 既存の各ドメインにあるisRecordと挙動を合わせるため、
 * 配列もobjectとして許可します。
 */
export function isRecord(
  value: unknown,
): value is Record<string, unknown> {
  return (
    typeof value === "object" &&
    value !== null
  );
}

/**
 * unknown値が有限なnumberかを判定します。
 *
 * NaN、Infinity、-Infinityはfalseになります。
 */
export function isFiniteNumber(
  value: unknown,
): value is number {
  return (
    typeof value === "number" &&
    Number.isFinite(value)
  );
}