// frontend/order/src/domain/entity/shippingAddress.ts
// Mirror of backend/internal/domain/shippingAddress/entity.go
// and shared ShippingAddress schema.

/**
 * ShippingAddress
 * - ユーザーの配送先住所
 * - 日付は ISO8601 文字列 or Date オブジェクト
 */
export interface ShippingAddress {
  id: string;
  userId: string;
  street: string;
  city: string;
  state: string;
  zipCode: string;
  country: string;
  createdAt: string | Date;
  updatedAt: string | Date;
}

/**
 * ドメインエラーメッセージ（開発・デバッグ用）
 * backend/internal/domain/shippingAddress/entity.go と整合
 */
export const ShippingAddressError = {
  InvalidID: "shippingAddress: invalid id",
  InvalidUserID: "shippingAddress: invalid userId",
  InvalidStreet: "shippingAddress: invalid street",
  InvalidCity: "shippingAddress: invalid city",
  InvalidState: "shippingAddress: invalid state",
  InvalidZipCode: "shippingAddress: invalid zipCode",
  InvalidCountry: "shippingAddress: invalid country",
  InvalidCreatedAt: "shippingAddress: invalid createdAt",
  InvalidUpdatedAt: "shippingAddress: invalid updatedAt",
} as const;

/**
 * 簡易バリデーション
 * - 空文字チェック
 * - createdAt / updatedAt の存在と順序チェック
 */
export function validateShippingAddress(
  a: ShippingAddress
): { valid: true } | { valid: false; error: string } {
  if (!a.id?.trim()) {
    return { valid: false, error: ShippingAddressError.InvalidID };
  }
  if (!a.userId?.trim()) {
    return { valid: false, error: ShippingAddressError.InvalidUserID };
  }
  if (!a.street?.trim()) {
    return { valid: false, error: ShippingAddressError.InvalidStreet };
  }
  if (!a.city?.trim()) {
    return { valid: false, error: ShippingAddressError.InvalidCity };
  }
  if (!a.state?.trim()) {
    return { valid: false, error: ShippingAddressError.InvalidState };
  }
  if (!a.zipCode?.trim()) {
    return { valid: false, error: ShippingAddressError.InvalidZipCode };
  }
  if (!a.country?.trim()) {
    return { valid: false, error: ShippingAddressError.InvalidCountry };
  }

  const created = toDate(a.createdAt);
  const updated = toDate(a.updatedAt);
  if (!created) {
    return { valid: false, error: ShippingAddressError.InvalidCreatedAt };
  }
  if (!updated || updated.getTime() < created.getTime()) {
    return { valid: false, error: ShippingAddressError.InvalidUpdatedAt };
  }

  return { valid: true };
}

/**
 * string | Date → Date 変換ユーティリティ（無効値は null）
 */
function toDate(v: string | Date | undefined | null): Date | null {
  if (!v) return null;
  if (v instanceof Date) return isNaN(v.getTime()) ? null : v;
  const d = new Date(v);
  return isNaN(d.getTime()) ? null : d;
}
