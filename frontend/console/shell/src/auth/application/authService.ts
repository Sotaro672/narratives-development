/// <reference types="vite/client" />

import { onAuthStateChanged, type User } from "firebase/auth";
import { auth } from "../infrastructure/config/firebaseClient";

/**
 * auth.currentUser が即時に得られないケースに備えて、
 * 一度だけ onAuthStateChanged を待つ Promise をメモ化。
 */
let authReadyPromise: Promise<User | null> | null = null;

function waitForAuthReady(): Promise<User | null> {
  if (auth.currentUser) return Promise.resolve(auth.currentUser);

  if (!authReadyPromise) {
    authReadyPromise = new Promise<User | null>((resolve) => {
      const unsub = onAuthStateChanged(auth, (u) => {
        unsub();
        authReadyPromise = null;
        resolve(u ?? null);
      });
    });
  }

  return authReadyPromise;
}

/** 現在の Firebase User を返す（未確定なら onAuthStateChanged を1回だけ待つ） */
export async function getCurrentUser(): Promise<User | null> {
  if (auth.currentUser) return auth.currentUser;
  return await waitForAuthReady();
}

/** 現在ユーザーの ID トークンを取得（無ければ null） */
export async function getIdToken(): Promise<string | null> {
  const u = await getCurrentUser();
  if (!u?.getIdToken) return null;

  try {
    return await u.getIdToken(false);
  } catch {
    return null;
  }
}

/** Authorization ヘッダを返す（取得できない場合は空オブジェクト） */
export async function getAuthHeaders(): Promise<Record<string, string>> {
  const token = await getIdToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}

// ─────────────────────────────────────────────
// Backend base URL
// ─────────────────────────────────────────────

const RAW_ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

function sanitizeBase(u: string): string {
  return (u || "").replace(/\/+$/g, "");
}

const API_BASE = sanitizeBase(RAW_ENV_BASE || FALLBACK_BASE);

if (!API_BASE) {
  throw new Error(
    "[authService] BACKEND BASE URL is empty. Set VITE_BACKEND_BASE_URL in .env.local",
  );
}

/**
 * Backend: /members/me から companyId を取得する。
 *
 * NOTE:
 * members の Firestore docId は Firebase Auth UID ではない。
 * そのため frontend から members/{uid} を直接読みに行かない。
 */
export async function getCompanyId(): Promise<string | null> {
  const headers = await getAuthHeaders();

  if (!headers.Authorization) {
    return null;
  }

  try {
    const res = await fetch(`${API_BASE}/members/me`, {
      method: "GET",
      mode: "cors",
      headers,
    });

    if (!res.ok) {
      console.error("[authService] getCompanyId /members/me failed:", res.status);
      return null;
    }

    const json = await res.json();
    const data = json?.data ?? json;

    const cid = String(data?.companyId ?? "").trim();
    return cid || null;
  } catch (error) {
    console.error("[authService] getCompanyId failed:", error);
    return null;
  }
}

/** 便利ユーティリティ：companyId を取得して返す */
export async function ensureCompanyId(): Promise<string | null> {
  return await getCompanyId();
}