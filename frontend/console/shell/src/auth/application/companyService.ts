/// <reference types="vite/client" />

// frontend/console/shell/src/auth/application/companyService.ts

import type { CompanyDTO } from "../domain/entity/company";
import { fetchCompanyByIdRaw } from "../infrastructure/repository/authRepositoryHTTP";

// -------------------------------
// In-memory cache for company names
// -------------------------------
const nameCache = new Map<string, Promise<string | null>>();

// -------------------------------
// Company: fetchers
// -------------------------------
export async function getCompanyById(
  companyId: string,
): Promise<CompanyDTO | null> {
  const id = (companyId ?? "").trim();
  if (!id) return null;

  const raw = await fetchCompanyByIdRaw(id);
  if (!raw) return null;

  // backend が CompanyDTO 互換 JSON を返している前提
  return raw as CompanyDTO;
}

export async function getCompanyNameById(
  companyId: string,
): Promise<string | null> {
  const data = await getCompanyById(companyId);
  const name = (data?.name ?? "").trim();
  return name || null;
}

// Cached version
export function getCompanyNameByIdCached(
  companyId: string,
): Promise<string | null> {
  const id = (companyId ?? "").trim();
  if (!id) return Promise.resolve(null);

  const cached = nameCache.get(id);
  if (cached) return cached;

  const p = getCompanyNameById(id).catch((err) => {
    console.error("[companyService] getCompanyNameByIdCached error", err);
    nameCache.delete(id);
    return null;
  });
  nameCache.set(id, p);
  return p;
}

export function clearCompanyNameCache(companyId?: string) {
  if (!companyId) {
    nameCache.clear();
  } else {
    nameCache.delete((companyId ?? "").trim());
  }
}
