// frontend\console\shell\src\features\company\application\locationCreateService.ts

import type { ShippingAddress } from "../../../shared/types/shippingAddress";
import { buildConsoleUrl } from "../../../shared/http/apiBase";
import { fetchJSON } from "../../../shared/http/fetchJSON";

export type CompanyShippingAddressCreateInput = {
  name: string;
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2: string;
  country: string;
};

/**
 * POST /companies/me/shipping-addresses
 *
 * UserIDとCompanyIDはBackend側で認証contextから設定する。
 * CreatedByとUpdatedByはBackend側で認証contextのmember document IDから設定する。
 * Frontendは在庫保管場所名と住所入力値だけを送信する。
 */
export async function createCompanyShippingAddress(
  input: CompanyShippingAddressCreateInput,
): Promise<ShippingAddress> {
  const url = buildConsoleUrl("/companies/me/shipping-addresses");

  return fetchJSON<ShippingAddress>(url, {
    method: "POST",
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
}