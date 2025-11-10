// frontend/shell/src/shared/types/billingAddress.ts
// Source of truth for BillingAddress type.
// Mirrors frontend/order/src/domain/entity/billingAddress.ts
// and backend/internal/domain/billingAddress/entity.go

/**
 * BillingAddress
 *
 * - id は UUID 形式
 * - nameOnAccount / 各種カード情報 / 住所は任意（null または省略可）
 * - createdAt / updatedAt は ISO8601 文字列
 */
export interface BillingAddress {
  /** 一意のID（UUID形式） */
  id: string;
  /** 紐づくユーザーID */
  userId: string;
  /** 口座名義（任意） */
  nameOnAccount?: string | null;

  /** 請求タイプ（例: "credit_card", "bank_transfer" など）*/
  billingType: string;

  /** カード情報（任意） */
  cardBrand?: string | null;
  cardLast4?: string | null;
  cardExpMonth?: number | null;
  cardExpYear?: number | null;
  cardToken?: string | null;

  /** 請求先住所情報（任意） */
  postalCode?: number | null;
  state?: string | null;
  city?: string | null;
  street?: string | null;
  country?: string | null;

  /** デフォルト請求先フラグ */
  isDefault: boolean;

  /** 作成日時（ISO8601） */
  createdAt: string;
  /** 更新日時（ISO8601） */
  updatedAt: string;
}

/**
 * 部分更新用パッチ型
 * backend/internal/domain/billingAddress/entity.go の BillingAddressPatch に対応。
 * - undefined: 変更なし
 * - null: クリア意図（対応実装側のポリシーに従う）
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
