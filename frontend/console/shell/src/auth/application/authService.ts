/// <reference types="vite/client" />

import { onAuthStateChanged, type User } from "firebase/auth";
import { auth, db } from "../config/firebaseClient";
import { doc, getDoc } from "firebase/firestore";

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
    return await u.getIdToken(/* forceRefresh? */ false);
  } catch {
    return null;
  }
}

/** Authorization ヘッダを返す（取得できない場合は空オブジェクト） */
export async function getAuthHeaders(): Promise<Record<string, string>> {
  const token = await getIdToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}

/**
 * Firestore: members/{uid} から companyId を取得（存在しなければ null）
 * サーバ側で context.companyId を強制する方針でも、クライアント側の表示/補助用途に便利。
 */
export async function getCompanyId(): Promise<string | null> {
  const u = await getCurrentUser();
  const uid = u?.uid?.trim();
  if (!uid) return null;

  try {
    const snap = await getDoc(doc(db, "members", uid));
    if (!snap.exists()) return null;
    const data = snap.data() || {};
    const cid = (data.companyId ?? "").toString().trim();
    return cid || null;
  } catch {
    return null;
  }
}

/** 便利ユーティリティ：companyId が取得できるまで待って返す（タイムアウト無し版） */
export async function ensureCompanyId(): Promise<string | null> {
  // 必要に応じてリトライ/タイムアウトを足してください
  return await getCompanyId();
}
