// frontend/console/shell/src/auth/application/companyService.ts

import type {
  CompanyDTO,
} from "../../shared/types/company";

import {
  fetchCompanyByIdRaw,
} from "../infrastructure/repository/authRepositoryHTTP";

// -------------------------------
// 会社名キャッシュ
// -------------------------------

const companyNameCache = new Map<
  string,
  Promise<string | null>
>();

// -------------------------------
// Company取得
// -------------------------------

async function getCompanyById(
  companyId: string,
): Promise<CompanyDTO | null> {
  const normalizedCompanyId =
    companyId.trim();

  if (!normalizedCompanyId) {
    return null;
  }

  const raw =
    await fetchCompanyByIdRaw(
      normalizedCompanyId,
    );

  if (!raw) {
    return null;
  }

  return raw as CompanyDTO;
}

async function getCompanyNameById(
  companyId: string,
): Promise<string | null> {
  const company =
    await getCompanyById(
      companyId,
    );

  const companyName =
    company?.name?.trim() ?? "";

  return companyName || null;
}

/**
 * companyIdに対応する会社名を取得する。
 *
 * 同じcompanyIdへの重複リクエストを防ぐため、
 * 取得中および取得済みのPromiseをメモリ上に保持する。
 */
export function getCompanyNameByIdCached(
  companyId: string,
): Promise<string | null> {
  const normalizedCompanyId =
    companyId.trim();

  if (!normalizedCompanyId) {
    return Promise.resolve(
      null,
    );
  }

  const cachedRequest =
    companyNameCache.get(
      normalizedCompanyId,
    );

  if (cachedRequest) {
    return cachedRequest;
  }

  const request =
    getCompanyNameById(
      normalizedCompanyId,
    ).catch(
      (
        error: unknown,
      ) => {
        companyNameCache.delete(
          normalizedCompanyId,
        );

        console.error(
          "[companyService] failed to fetch company name:",
          error,
        );

        return null;
      },
    );

  companyNameCache.set(
    normalizedCompanyId,
    request,
  );

  return request;
}