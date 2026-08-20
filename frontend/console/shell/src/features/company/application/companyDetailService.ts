// frontend/console/shell/src/features/company/application/companyDetailService.ts

import type { Company } from "../../../shared/types/company";
import { buildConsoleUrl } from "../../../shared/http/apiBase";
import { fetchJSON } from "../../../shared/http/fetchJSON";

export type UpdateCompanyDetailInput = {
  name: string;
  admin: string;
};

function requireCompanyID(companyId: string): string {
  if (!companyId) {
    throw new Error("companyId is required");
  }

  return companyId;
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