// frontend/order/src/domain/entity/billingAddress.ts
// Mirrors backend/internal/domain/billingAddress/entity.go

export interface BillingAddress {
  /** 一意のID（UUID形式） */
  id: string;
  /** 紐づくユーザーID */
  userId: string;
  /** 口座名義（任意） */
  nameOnAccount?: string | null;

  /** 請求タイプ（例: "credit_card", "bank_transfer" など） */
  billingType: string;

  /** カード情報（任意） */
  cardBrand?: string | null;
  cardLast4?: string | null;
  cardExpMonth?: number | null;
  cardExpYear?: number | null;
  cardToken?: string | null;

  /** 住所情報（任意） */
  postalCode?: number | null;
  state?: string | null;
  city?: string | null;
  street?: string | null;
  country?: string | null;

  /** デフォルト住所フラグ */
  isDefault: boolean;

  /** 作成日時（ISO文字列） */
  createdAt: string;
  /** 更新日時（ISO文字列） */
  updatedAt: string;
}

/**
 * バリデーション関数
 * backend/internal/domain/billingAddress/entity.go の validate() に対応。
 */
export function isValidBillingAddress(b: BillingAddress): boolean {
  const uuidRe =
    /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$/;
  const last4Re = /^\d{4}$/;

  if (!uuidRe.test(b.id)) return false;
  if (!b.userId?.trim()) return false;
  if (!b.billingType?.trim()) return false;

  if (b.cardLast4 && !last4Re.test(b.cardLast4)) return false;
  if (b.cardExpMonth && (b.cardExpMonth < 1 || b.cardExpMonth > 12)) return false;
  if (b.cardExpYear && (b.cardExpYear < 2000 || b.cardExpYear > 2100)) return false;
  if (b.postalCode && b.postalCode < 0) return false;

  const created = new Date(b.createdAt);
  const updated = new Date(b.updatedAt);
  if (isNaN(created.getTime()) || isNaN(updated.getTime())) return false;
  if (updated < created) return false;

  return true;
}

/**
 * 部分更新用のパッチ型
 * backend/internal/domain/billingAddress/entity.go の BillingAddressPatch に準拠。
 */
export interface BillingAddressPatch {
  fullName?: string | null;
  company?: string | null;
  country?: string | null;
  postalCode?: string | null;
  state?: string | null;
  city?: string | null;
  address1?: string | null;
  address2?: string | null;
  phone?: string | null;

  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
}
