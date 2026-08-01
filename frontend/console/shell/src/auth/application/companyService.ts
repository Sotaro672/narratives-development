// frontend/console/shell/src/auth/application/companyService.ts

import type {
  CompanyDTO,
} from "../../shared/types/company";

import {
  fetchCompanyByIdRaw,
} from "../infrastructure/repository/authRepositoryHTTP";

/**
 * companyIdに対応する会社情報を取得する。
 */
async function getCompanyById(
  companyId: string,
): Promise<CompanyDTO | null> {
  if (!companyId) {
    return null;
  }

  const raw =
    await fetchCompanyByIdRaw(
      companyId,
    );

  if (!raw) {
    return null;
  }

  return raw as CompanyDTO;
}

/**
 * companyIdに対応する会社名を取得する。
 *
 * キャッシュは保持せず、呼び出しのたびに
 * Backendから最新の会社情報を取得する。
 */
export async function getCompanyNameById(
  companyId: string,
): Promise<string | null> {
  if (!companyId) {
    return null;
  }

  const company =
    await getCompanyById(
      companyId,
    );

  const companyName =
    company?.name?.trim() ?? "";

  return companyName || null;
}