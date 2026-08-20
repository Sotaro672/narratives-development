// frontend/console/shell/src/shared/types/shippingAddress.ts

/**
 * 配送先住所・在庫保管場所エンティティ。
 *
 * backend/internal/domain/shippingAddress/entity.go に対応する。
 * name は配送先住所・在庫保管場所を識別する必須名称。
 * createdBy / updatedBy は Console から登録・更新された場合のみ members document ID を保持する。
 * User 側で登録された配送先住所では未設定を許可する。
 */
export interface ShippingAddress {
  id: string;
  userId: string;
  companyId: string;
  name: string;
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2: string;
  country: string;
  createdAt: string;
  createdBy?: string;
  updatedAt: string;
  updatedBy?: string;
}

/**
 * Console の在庫保管場所一覧・詳細 GET API 用 ReadModel。
 *
 * createdBy / updatedBy は members document ID。
 * createdByName / updatedByName は Backend Query で名前解決された画面表示用の値。
 *
 * GET /companies/me/shipping-addresses
 * GET /companies/me/shipping-addresses/{shippingAddressId}
 * に対応する。
 */
export interface CompanyShippingAddressReadModel extends ShippingAddress {
  createdByName: string;
  updatedByName: string;
}

/**
 * 配送先住所・在庫保管場所のフォーム入力値。
 *
 * id、userId、companyId、createdBy、updatedBy は Backend 側または既存 entity から解決するため、
 * Frontend のフォームからは送信しない。
 */
export type ShippingAddressFormValue = {
  name: string;
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2: string;
  country: "JP";
};

/**
 * 配送先住所が必要なフィールドを保持しているか検証する。
 *
 * street2 は任意項目のため空文字を許可する。
 * createdBy / updatedBy も User 側では未設定を許可する。
 */
export function isValidShippingAddress(address: ShippingAddress): boolean {
  if (!address) return false;
  if (!address.id.trim()) return false;
  if (!address.userId.trim()) return false;
  if (!address.companyId.trim()) return false;
  if (!address.name.trim()) return false;
  if (!address.zipCode.trim()) return false;
  if (!address.state.trim()) return false;
  if (!address.city.trim()) return false;
  if (!address.street.trim()) return false;
  if (!address.country.trim()) return false;

  const createdAt = new Date(address.createdAt);
  const updatedAt = new Date(address.updatedAt);

  if (Number.isNaN(createdAt.getTime()) || Number.isNaN(updatedAt.getTime())) return false;
  if (updatedAt < createdAt) return false;

  return true;
}

export type ShippingAddressPatch = Partial<
  Pick<
    ShippingAddress,
    "name" | "zipCode" | "state" | "city" | "street" | "street2" | "country"
  >
>;

/**
 * 配送先住所の入力項目を更新する。
 *
 * id、userId、companyId、createdAt、createdBy、updatedBy は変更しない。
 * updatedBy は Console Backend が認証 context の member document ID から更新する。
 */
export function updateShippingAddress(
  address: ShippingAddress,
  patch: ShippingAddressPatch,
  now: Date = new Date(),
): ShippingAddress {
  const next: ShippingAddress = { ...address };

  if (patch.name !== undefined) next.name = patch.name.trim();
  if (patch.zipCode !== undefined) next.zipCode = patch.zipCode.trim();
  if (patch.state !== undefined) next.state = patch.state.trim();
  if (patch.city !== undefined) next.city = patch.city.trim();
  if (patch.street !== undefined) next.street = patch.street.trim();
  if (patch.street2 !== undefined) next.street2 = patch.street2.trim();
  if (patch.country !== undefined) next.country = patch.country.trim();

  next.updatedAt = now.toISOString();
  return next;
}