// frontend/list/src/domain/entity/discount.ts
// Mirror of backend/internal/domain/discount/entity.go
// and shared/types/discount.ts (to be generated/used as source of truth).

/**
 * 個別モデルの割引アイテム
 * - modelNumber: モデル番号
 * - discount: 割引率 (%)
 */
export interface DiscountItem {
  modelNumber: string;
  discount: number; // percent (0..100 by default policy)
}

/**
 * 割引設定エンティティ
 *
 * backend/internal/domain/discount/entity.go に準拠。
 */
export interface Discount {
  /** 主キー（例: discount_xxx） */
  id: string;
  /** 出品ID */
  listId: string;
  /** モデル番号ごとの割引率配列（ユニーク & 正当値） */
  discounts: DiscountItem[];
  /** 割引の説明（任意） */
  description?: string;
  /** 割引設定者のメンバーID */
  discountedBy: string;
  /** 割引設定日時（ISO8601文字列） */
  discountedAt: string;
  /** 最終更新日時（ISO8601文字列） */
  updatedAt: string;
  /** 最終更新者のメンバーID */
  updatedBy: string;
}

/**
 * Policy (backend と同期)
 */
export const DISCOUNT_ID_PREFIX = "discount_";
export const DISCOUNT_ENFORCE_ID_PREFIX = false;

export const DISCOUNT_MIN_PERCENT = 0;
export const DISCOUNT_MAX_PERCENT = 100;

export const DISCOUNT_MODEL_NUMBER_REGEX = /^[A-Za-z0-9._-]{1,64}$/;

/** description の最大長 (0 の場合は無制限) */
export const DISCOUNT_MAX_DESCRIPTION_LENGTH = 1000;

/** discounts の最小件数 (0 の場合はチェック無効) */
export const DISCOUNT_MIN_ITEMS_REQUIRED = 1;

/**
 * 割引率がポリシー上妥当か判定
 */
export function isValidDiscountPercent(v: number): boolean {
  if (!Number.isInteger(v)) return false;
  if (v < DISCOUNT_MIN_PERCENT) return false;
  if (DISCOUNT_MAX_PERCENT > 0 && v > DISCOUNT_MAX_PERCENT) return false;
  return true;
}

/**
 * 単一 DiscountItem の検証
 */
export function validateDiscountItem(item: DiscountItem): boolean {
  const modelNumber = item.modelNumber?.trim();
  if (!modelNumber) return false;
  if (
    DISCOUNT_MODEL_NUMBER_REGEX &&
    !DISCOUNT_MODEL_NUMBER_REGEX.test(modelNumber)
  ) {
    return false;
  }
  if (!isValidDiscountPercent(item.discount)) return false;
  return true;
}

/**
 * Discount 全体の検証
 * - ID / listId / discountedBy / updatedBy が非空
 * - discounts がユニークかつ有効
 * - description が長さ制約内
 * - 日付が ISO として parse 可能（厳密 UTC はここでは強制しない）
 * - discountedAt <= updatedAt を推奨チェック
 */
export function validateDiscount(discount: Discount): boolean {
  // id
  const id = discount.id?.trim();
  if (!id) return false;
  if (
    DISCOUNT_ENFORCE_ID_PREFIX &&
    DISCOUNT_ID_PREFIX &&
    !id.startsWith(DISCOUNT_ID_PREFIX)
  ) {
    return false;
  }

  // listId
  if (!discount.listId?.trim()) return false;

  // discountedBy / updatedBy
  if (!discount.discountedBy?.trim()) return false;
  if (!discount.updatedBy?.trim()) return false;

  // discounts
  if (
    DISCOUNT_MIN_ITEMS_REQUIRED > 0 &&
    (!discount.discounts ||
      discount.discounts.length < DISCOUNT_MIN_ITEMS_REQUIRED)
  ) {
    return false;
  }

  const seen = new Set<string>();
  for (const item of discount.discounts) {
    if (!validateDiscountItem(item)) return false;
    const key = item.modelNumber.trim();
    if (seen.has(key)) return false;
    seen.add(key);
  }

  // description length
  if (discount.description != null) {
    const desc = discount.description;
    if (
      DISCOUNT_MAX_DESCRIPTION_LENGTH > 0 &&
      [...desc].length > DISCOUNT_MAX_DESCRIPTION_LENGTH
    ) {
      return false;
    }
  }

  // datetime checks (best-effort)
  const discountedAt = new Date(discount.discountedAt);
  if (Number.isNaN(discountedAt.getTime())) return false;

  const updatedAt = new Date(discount.updatedAt);
  if (Number.isNaN(updatedAt.getTime())) return false;

  if (updatedAt.getTime() < discountedAt.getTime()) return false;

  return true;
}

/**
 * DiscountItem 配列を modelNumber 単位で正規化（最後の定義が優先）
 * - 不正な item はスキップ
 */
export function normalizeDiscountItems(items: DiscountItem[]): DiscountItem[] {
  const map = new Map<string, number>();
  const order: string[] = [];

  for (const item of items) {
    if (!validateDiscountItem(item)) continue;
    const key = item.modelNumber.trim();
    if (!map.has(key)) {
      order.push(key);
    }
    map.set(key, item.discount);
  }

  return order.map((modelNumber) => ({
    modelNumber,
    discount: map.get(modelNumber)!,
  }));
}
