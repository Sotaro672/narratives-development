// frontend/console/shell/src/features/company/application/locationManagementService.ts

import type {
  CompanyShippingAddressReadModel,
  ShippingAddress,
} from "../../../shared/types/shippingAddress";
import { buildConsoleUrl } from "../../../shared/http/apiBase";
import { fetchJSON } from "../../../shared/http/fetchJSON";

export type LocationUpdateInput = {
  name: string;
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2: string;
  country: string;
};

function requireShippingAddressID(shippingAddressId: string): string {
  if (!shippingAddressId) {
    throw new Error("shippingAddressId is required");
  }

  return shippingAddressId;
}

/**
 * GET /companies/me/shipping-addresses
 *
 * 認証中memberのcompanyIdはBackend middlewareで解決する。
 * FrontendからcompanyIdはquery/bodyへ送信しない。
 * createdBy / updatedByはmember document ID、
 * createdByName / updatedByNameはBackend Queryで解決した表示名を受け取る。
 */
export async function listCompanyShippingAddresses(): Promise<
  CompanyShippingAddressReadModel[]
> {
  const url = buildConsoleUrl("/companies/me/shipping-addresses");

  const response = await fetchJSON<CompanyShippingAddressReadModel[]>(url, {
    method: "GET",
    auth: "required",
  });

  return Array.isArray(response) ? response : [];
}

/**
 * GET /companies/me/shipping-addresses/{shippingAddressId}
 *
 * 認証中memberのcompanyIdはBackend middlewareで解決する。
 * Backend側で現在のcompanyに属する在庫保管場所か確認して取得する。
 * createdBy / updatedByはmember document ID、
 * createdByName / updatedByNameはBackend Queryで解決した表示名を受け取る。
 */
export async function fetchCompanyShippingAddress(
  shippingAddressId: string,
): Promise<CompanyShippingAddressReadModel> {
  const id = requireShippingAddressID(shippingAddressId);
  const url = buildConsoleUrl(
    `/companies/me/shipping-addresses/${encodeURIComponent(id)}`,
  );

  return fetchJSON<CompanyShippingAddressReadModel>(url, {
    method: "GET",
    auth: "required",
  });
}

/**
 * PATCH /companies/me/shipping-addresses/{shippingAddressId}
 *
 * BackendではGetByCompanyによって現在のcompanyに属する在庫保管場所か確認してから更新する。
 * UpdatedByはBackend側で認証contextのmember document IDから設定する。
 * FrontendからcreatedBy / updatedByは送信しない。
 *
 * PATCH responseはShippingAddress entityのため、
 * 更新後にGETを実行してcreatedByName / updatedByNameを含むReadModelを取得する。
 */
export async function updateCompanyShippingAddress(
  shippingAddressId: string,
  input: LocationUpdateInput,
): Promise<CompanyShippingAddressReadModel> {
  const id = requireShippingAddressID(shippingAddressId);
  const url = buildConsoleUrl(
    `/companies/me/shipping-addresses/${encodeURIComponent(id)}`,
  );

  await fetchJSON<ShippingAddress>(url, {
    method: "PATCH",
    auth: "required",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      name: input.name,
      zipCode: input.zipCode,
      state: input.state,
      city: input.city,
      street: input.street,
      street2: input.street2,
      country: input.country,
    }),
  });

  return fetchCompanyShippingAddress(id);
}

/**
 * DELETE /companies/me/shipping-addresses/{shippingAddressId}
 *
 * BackendではDeleteByCompanyによってcompany所有権を確認して削除する。
 * 正常時は204 No Contentを想定する。
 */
export async function deleteCompanyShippingAddress(
  shippingAddressId: string,
): Promise<void> {
  const id = requireShippingAddressID(shippingAddressId);
  const url = buildConsoleUrl(
    `/companies/me/shipping-addresses/${encodeURIComponent(id)}`,
  );

  await fetchJSON<void>(url, {
    method: "DELETE",
    auth: "required",
  });
}