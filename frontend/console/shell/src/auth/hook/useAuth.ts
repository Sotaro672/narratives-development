// frontend/console/shell/src/auth/application/authRepository.ts
/// <reference types="vite/client" />

import { useEffect, useMemo, useState } from "react";
import { useAuthContext } from "../application/AuthContext";

// -------------------------------
// Backend base URL
// -------------------------------
const API_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/,
    "",
  ) ?? "";

// -------------------------------
// Types
// -------------------------------
export type CompanyDTO = {
  id?: string;
  name?: string;
};

// -------------------------------
// In-memory cache for company names
// -------------------------------
const nameCache = new Map<string, Promise<string | null>>();

// -------------------------------
// Plain fetchers
// -------------------------------
export async function getCompanyById(companyId: string): Promise<CompanyDTO | null> {
  const id = (companyId ?? "").trim();
  if (!id) return null;

  const res = await fetch(`${API_BASE}/companies/${encodeURIComponent(id)}`, { method: "GET" });
  if (!res.ok) return null;

  const data = (await res.json()) as CompanyDTO;
  return data ?? null;
}

export async function getCompanyNameById(companyId: string): Promise<string | null> {
  const data = await getCompanyById(companyId);
  const name = (data?.name ?? "").trim();
  return name || null;
}

// Cached version (recommended)
export function getCompanyNameByIdCached(companyId: string): Promise<string | null> {
  const id = (companyId ?? "").trim();
  if (!id) return Promise.resolve(null);

  const cached = nameCache.get(id);
  if (cached) return cached;

  const p = getCompanyNameById(id).catch(() => {
    // On failure, drop cache entry so callers can retry later
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

// -------------------------------
// Hook: useAuth (adds companyName)
// -------------------------------
export function useAuth() {
  const ctx = useAuthContext();

  // Normalize to empty string to keep deps stable & avoid null issues
  const companyId = ctx.user?.companyId?.trim() ?? "";

  const [companyName, setCompanyName] = useState<string | null>(null);
  const [loadingCompanyName, setLoadingCompanyName] = useState(false);
  const [companyError, setCompanyError] = useState<string | null>(null);

  useEffect(() => {
    let disposed = false;

    async function run() {
      if (!companyId) {
        setCompanyName(null);
        setCompanyError(null);
        setLoadingCompanyName(false);
        return;
      }

      setLoadingCompanyName(true);
      setCompanyError(null);

      try {
        const name = await getCompanyNameByIdCached(companyId);
        if (!disposed) setCompanyName(name);
      } catch (e: any) {
        if (!disposed) {
          setCompanyName(null);
          setCompanyError(e?.message ?? "failed to fetch company name");
        }
      } finally {
        if (!disposed) setLoadingCompanyName(false);
      }
    }

    run();
    return () => {
      disposed = true;
    };
  }, [companyId]);

  // Return original auth context values + extras
  return useMemo(
    () => ({
      ...ctx,
      companyName,
      loadingCompanyName,
      companyError,
    }),
    [ctx, companyName, loadingCompanyName, companyError],
  );
}
