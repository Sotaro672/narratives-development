// frontend/shell/src/shared/types/shippingAddress.ts
// Generated from frontend/order/src/domain/entity/shippingAddress.ts
// Mirrors backend/internal/domain/shippingAddress/entity.go

/**
 * ShippingAddress
 * 配送先住所エンティティ（ユーザー単位で複数登録可能）
 * backend/internal/domain/shippingAddress/entity.go に対応。
 */
export interface ShippingAddress {
  /** 一意なID（例: "ship_001"） */
  id: string;
  /** ユーザーID（例: "user_001"） */
  userId: string;
  /** 番地・丁目など詳細住所 */
  street: string;
  /** 市区町村 */
  city: string;
  /** 都道府県・州 */
  state: string;
  /** 郵便番号 */
  zipCode: string;
  /** 国名（例: "Japan"） */
  country: string;
  /** 作成日時（ISO8601文字列） */
  createdAt: string;
  /** 更新日時（ISO8601文字列） */
  updatedAt: string;
}

/**
 * 配送先住所のバリデーション関数
 * backend/internal/domain/shippingAddress/entity.go の validate() に準拠。
 */
export function isValidShippingAddress(addr: ShippingAddress): boolean {
  if (!addr) return false;
  if (!addr.id?.trim()) return false;
  if (!addr.userId?.trim()) return false;
  if (!addr.street?.trim()) return false;
  if (!addr.city?.trim()) return false;
  if (!addr.state?.trim()) return false;
  if (!addr.zipCode?.trim()) return false;
  if (!addr.country?.trim()) return false;

  const created = new Date(addr.createdAt);
  const updated = new Date(addr.updatedAt);
  if (isNaN(created.getTime()) || isNaN(updated.getTime())) return false;
  if (updated < created) return false;

  return true;
}

/**
 * 住所の更新関数
 * backend の UpdateLines() 相当（値のトリムと更新日時更新）。
 */
export function updateShippingAddress(
  addr: ShippingAddress,
  {
    street,
    city,
    state,
    zipCode,
    country,
  }: Partial<
    Pick<ShippingAddress, "street" | "city" | "state" | "zipCode" | "country">
  >,
  now: Date = new Date()
): ShippingAddress {
  const next: ShippingAddress = { ...addr };
  if (street) next.street = street.trim();
  if (city) next.city = city.trim();
  if (state) next.state = state.trim();
  if (zipCode) next.zipCode = zipCode.trim();
  if (country) next.country = country.trim();
  next.updatedAt = now.toISOString();
  return next;
}
