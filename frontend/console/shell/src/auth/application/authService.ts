/// <reference types="vite/client" />

import { onAuthStateChanged, type User } from "firebase/auth";
import { auth } from "../infrastructure/config/firebaseClient";
import { fetchCurrentMemberRaw } from "../infrastructure/repository/authRepositoryHTTP";

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
export async function getIdToken(forceRefresh = false): Promise<string | null> {
  const u = await getCurrentUser();
  if (!u?.getIdToken) return null;

  try {
    return await u.getIdToken(forceRefresh);
  } catch {
    return null;
  }
}

/** Authorization ヘッダを返す（取得できない場合は空オブジェクト） */
export async function getAuthHeaders(
  forceRefresh = false,
): Promise<Record<string, string>> {
  const token = await getIdToken(forceRefresh);
  return token ? { Authorization: `Bearer ${token}` } : {};
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export type CurrentMemberResponse = {
  id?: string;
  uid?: string;
  firstName?: string | null;
  lastName?: string | null;
  firstNameKana?: string | null;
  lastNameKana?: string | null;
  email?: string | null;
  permissions?: string[];
  companyId?: string;
  status?: string;
  createdAt?: string;
  updatedAt?: string;
  displayName?: string | null;
};

async function fetchCurrentMemberOnce(): Promise<CurrentMemberResponse | null> {
  const user = await getCurrentUser();

  if (!user) {
    return null;
  }

  const raw = await fetchCurrentMemberRaw();
  const member = raw?.data ?? raw ?? null;

  return member;
}

/**
 * Backend: /members/me から current member を取得する。
 *
 * 新規登録直後は Firebase Auth user は存在していても、
 * backend 側 member document がまだ作成直後/未反映のことがあるため、
 * 404 は短時間だけ retry する。
 */
export async function getCurrentMember(options?: {
  retries?: number;
  retryDelayMs?: number;
}): Promise<CurrentMemberResponse | null> {
  const retries = options?.retries ?? 5;
  const retryDelayMs = options?.retryDelayMs ?? 300;

  for (let i = 0; i <= retries; i += 1) {
    try {
      const member = await fetchCurrentMemberOnce();

      if (member?.id && member?.companyId) {
        return member;
      }

      if (i < retries) {
        await sleep(retryDelayMs * (i + 1));
        continue;
      }

      return member;
    } catch (error) {
      console.error("[authService] getCurrentMember failed:", error);

      if (i < retries) {
        await sleep(retryDelayMs * (i + 1));
        continue;
      }

      return null;
    }
  }

  return null;
}

/**
 * Backend: /members/me から companyId を取得する。
 *
 * NOTE:
 * members の Firestore docId は Firebase Auth UID ではない。
 * そのため frontend から members/{uid} を直接読みに行かない。
 */
export async function getCompanyId(): Promise<string | null> {
  const member = await getCurrentMember({
    retries: 5,
    retryDelayMs: 300,
  });

  const cid = String(member?.companyId ?? "").trim();
  return cid || null;
}

/** 便利ユーティリティ：companyId を取得して返す */
export async function ensureCompanyId(): Promise<string | null> {
  return await getCompanyId();
}