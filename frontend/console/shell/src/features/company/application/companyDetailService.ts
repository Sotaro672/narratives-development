// frontend/console/shell/src/features/company/application/companyDetailService.ts 
 
import type { Company } from "../../../shared/types/company"; 
import type { ShippingAddress } from "../../../shared/types/shippingAddress"; 
import { buildConsoleUrl } from "../../../shared/http/apiBase"; 
import { fetchJSON } from "../../../shared/http/fetchJSON"; 
 
export type UpdateCompanyDetailInput = { 
  name: string; 
  admin: string; 
}; 
 
export type CompanyShippingAddressInput = { 
  name: string; 
  zipCode: string; 
  state: string; 
  city: string; 
  street: string; 
  street2: string; 
  country: string; 
}; 
 
function requireCompanyID(companyId: string): string { 
  if (!companyId) { 
    throw new Error("companyId is required"); 
  } 
  return companyId; 
} 
 
function requireShippingAddressID(shippingAddressId: string): string { 
  if (!shippingAddressId) { 
    throw new Error("shippingAddressId is required"); 
  } 
  return shippingAddressId; 
} 
 
/** 
 * GET /companies/{companyId} 
 * 
 * 現在ログイン中memberのcompanyIdを呼び出し側から受け取り、 
 * Company詳細を取得する。 
 */ 
export async function fetchCompanyDetail(companyId: string): Promise<Company> { 
  const id = requireCompanyID(companyId); 
  const url = buildConsoleUrl(`/companies/${encodeURIComponent(id)}`); 
 
  return fetchJSON<Company>(url, { 
    method: "GET", 
    auth: "required", 
  }); 
} 
 
/** 
 * PATCH /companies/{companyId} 
 * 
 * CompanyDetail画面から会社名とadmin memberを更新する。 
 * adminはFirebase Auth UIDではなくmembers document IDを送信する。 
 */ 
export async function updateCompanyDetail( 
  companyId: string, 
  input: UpdateCompanyDetailInput, 
): Promise<Company> { 
  const id = requireCompanyID(companyId); 
  const url = buildConsoleUrl(`/companies/${encodeURIComponent(id)}`); 
 
  return fetchJSON<Company>(url, { 
    method: "PATCH", 
    auth: "required", 
    headers: { 
      "Content-Type": "application/json", 
    }, 
    body: JSON.stringify({ 
      name: input.name, 
      admin: input.admin, 
    }), 
  }); 
} 
 
/** 
 * GET /companies/me/shipping-addresses 
 * 
 * 認証中memberのcompanyIdはBackend middlewareで解決する。 
 * FrontendからcompanyIdはquery/bodyへ送信しない。 
 */ 
export async function listCompanyShippingAddresses(): Promise<ShippingAddress[]> { 
  const url = buildConsoleUrl("/companies/me/shipping-addresses"); 
 
  const response = await fetchJSON<ShippingAddress[]>(url, { 
    method: "GET", 
    auth: "required", 
  }); 
 
  return Array.isArray(response) ? response : []; 
} 
 
/** 
 * GET /companies/me/shipping-addresses/{shippingAddressId} 
 * 
 * 認証中memberのcompanyIdはBackend middlewareで解決する。 
 * Backend側で現在のcompanyに属する住所か確認して取得する。 
 */ 
export async function fetchCompanyShippingAddress( 
  shippingAddressId: string, 
): Promise<ShippingAddress> { 
  const id = requireShippingAddressID(shippingAddressId); 
  const url = buildConsoleUrl( 
    `/companies/me/shipping-addresses/${encodeURIComponent(id)}`, 
  ); 
 
  return fetchJSON<ShippingAddress>(url, { 
    method: "GET", 
    auth: "required", 
  }); 
} 
 
/** 
 * POST /companies/me/shipping-addresses 
 * 
 * UserIDとCompanyIDはBackend側で認証contextから設定する。 
 * Frontendは在庫保管場所名と住所入力値だけを送信する。 
 */ 
export async function createCompanyShippingAddress( 
  input: CompanyShippingAddressInput, 
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
 
/** 
 * PATCH /companies/me/shipping-addresses/{shippingAddressId} 
 * 
 * BackendではGetByCompanyによって現在のcompanyに属する住所か確認してから更新する。 
 */ 
export async function updateCompanyShippingAddress( 
  shippingAddressId: string, 
  input: CompanyShippingAddressInput, 
): Promise<ShippingAddress> { 
  const id = requireShippingAddressID(shippingAddressId); 
  const url = buildConsoleUrl( 
    `/companies/me/shipping-addresses/${encodeURIComponent(id)}`, 
  ); 
 
  return fetchJSON<ShippingAddress>(url, { 
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