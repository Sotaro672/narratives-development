// frontend\console\shell\src\auth\hook\useAuth.ts
/// <reference types="vite/client" />

import { useEffect, useMemo, useState } from "react";
import { useAuthContext } from "../application/AuthContext";
import { auth } from "../config/firebaseClient";

// -------------------------------
// Backend base URL（useMemberDetail と同じ構成）
// -------------------------------
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

// 最終的に使うベース URL
const API_BASE = ENV_BASE || FALLBACK_BASE;

// -------------------------------
// Types
// -------------------------------
export type CompanyDTO = {
  id?: string;
  name?: string;
};

export type MemberDTO = {
  id: string;
  firstName?: string | null;
  lastName?: string | null;
  email?: string | null;
  companyId: string;
  /** 姓名を「姓 名」の形で結合したもの（空なら null） */
  fullName?: string | null;
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

  const res = await fetch(`${API_BASE}/companies/${encodeURIComponent(id)}`, {
    method: "GET",
  });
  if (!res.ok) return null;

  const data = (await res.json()) as CompanyDTO;
  return data ?? null;
}

export async function getCompanyNameById(companyId: string): Promise<string | null> {
  const data = await getCompanyById(companyId);
  const name = (data?.name ?? "").trim();
  return name || null;
}

// Cached version
export function getCompanyNameByIdCached(companyId: string): Promise<string | null> {
  const id = (companyId ?? "").trim();
  if (!id) return Promise.resolve(null);

  const cached = nameCache.get(id);
  if (cached) return cached;

  const p = getCompanyNameById(id).catch(() => {
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
// Fetch currentMember（useMemberDetail と同じ API_BASE & 防御）
// -------------------------------
async function fetchCurrentMember(uid: string): Promise<MemberDTO | null> {
  const token = await auth.currentUser?.getIdToken();
  if (!token) return null;

  const url = `${API_BASE}/members/${encodeURIComponent(uid)}`;
  console.log("[useAuth] fetchCurrentMember uid:", uid, "GET", url);

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    console.warn(
      "[useAuth] fetchCurrentMember failed:",
      res.status,
      res.statusText,
      text,
    );
    return null;
  }

  // HTML が返ってきていないかチェック（env ミス検出用）
  const ct = res.headers.get("Content-Type") ?? "";
  if (!ct.includes("application/json")) {
    throw new Error(
      `currentMember API が JSON を返していません (content-type=${ct}). ` +
        `VITE_BACKEND_BASE_URL または API_BASE=${API_BASE} を確認してください。`,
    );
  }

  const raw = (await res.json()) as any;
  if (!raw) return null;

  const noFirst =
    raw.firstName === null ||
    raw.firstName === undefined ||
    raw.firstName === "";
  const noLast =
    raw.lastName === null ||
    raw.lastName === undefined ||
    raw.lastName === "";

  const firstName = noFirst ? null : (raw.firstName as string);
  const lastName = noLast ? null : (raw.lastName as string);

  const full =
    `${lastName ?? ""} ${firstName ?? ""}`.trim() || null;

  return {
    id: raw.id ?? uid,
    firstName,
    lastName,
    email: raw.email ?? null,
    companyId: raw.companyId ?? "",
    fullName: full,
  };
}

// -------------------------------
// Hook: useAuth (adds companyName + currentMember)
// -------------------------------
export function useAuth() {
  const ctx = useAuthContext();

  const uid = ctx.user?.uid ?? "";
  const companyId = ctx.user?.companyId?.trim() ?? "";

  const [companyName, setCompanyName] = useState<string | null>(null);
  const [loadingCompanyName, setLoadingCompanyName] = useState(false);
  const [companyError, setCompanyError] = useState<string | null>(null);

  const [currentMember, setCurrentMember] = useState<MemberDTO | null>(null);
  const [loadingMember, setLoadingMember] = useState(false);
  const [memberError, setMemberError] = useState<string | null>(null);

  // -------------------------------
  // Fetch companyName
  // -------------------------------
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

  // -------------------------------
  // Fetch currentMember (via backend)
// -------------------------------
  useEffect(() => {
    let disposed = false;

    async function loadMember() {
      if (!uid) {
        setCurrentMember(null);
        setMemberError(null);
        return;
      }

      setLoadingMember(true);
      setMemberError(null);

      try {
        const m = await fetchCurrentMember(uid);
        if (!disposed) setCurrentMember(m);
      } catch (e: any) {
        if (!disposed) {
          setCurrentMember(null);
          setMemberError(e?.message ?? "failed to fetch member");
        }
      } finally {
        if (!disposed) setLoadingMember(false);
      }
    }

    loadMember();
    return () => {
      disposed = true;
    };
  }, [uid]);

  // -------------------------------
  // Return
  // -------------------------------
  return useMemo(
    () => ({
      ...ctx,
      companyName,
      loadingCompanyName,
      companyError,
      currentMember,
      loadingMember,
      memberError,
    }),
    [
      ctx,
      companyName,
      loadingCompanyName,
      companyError,
      currentMember,
      loadingMember,
      memberError,
    ],
  );
}
