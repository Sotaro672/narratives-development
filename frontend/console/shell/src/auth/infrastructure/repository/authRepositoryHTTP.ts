/// <reference types="vite/client" />

// frontend/console/shell/src/auth/infrastructure/repository/authRepositoryHTTP.ts

import { auth } from "../config/firebaseClient";

// -------------------------------
// Backend base URL
// -------------------------------
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

// -------------------------------
// HTTP functions (Auth / Member / Company 用)
// -------------------------------

/**
 * members/{uid} を叩いて「生の JSON」を返すだけの関数
 * - トークン取得や fetch, Content-Type チェックなど HTTP 周りのみ担当
 * - DTO への変換は application 層 (memberService) に委譲
 */
export async function fetchCurrentMemberRaw(
  uid: string,
): Promise<any | null> {
  const token = await auth.currentUser?.getIdToken();
  if (!token) return null;

  const url = `${API_BASE}/members/${encodeURIComponent(uid)}`;
  console.log("[authRepositoryHTTP] fetchCurrentMemberRaw uid:", uid, "GET", url);

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
      "[authRepositoryHTTP] fetchCurrentMemberRaw failed:",
      res.status,
      res.statusText,
      text,
    );
    return null;
  }

  const ct = res.headers.get("Content-Type") ?? "";
  if (!ct.includes("application/json")) {
    throw new Error(
      `members API が JSON を返していません (content-type=${ct}). ` +
        `VITE_BACKEND_BASE_URL または API_BASE=${API_BASE} を確認してください。`,
    );
  }

  const raw = await res.json();
  return raw;
}

/**
 * members/{id} に PATCH する HTTP 関数
 * - payload の shape は caller (memberService) 側で決める
 * - 戻り値は生 JSON
 */
export async function updateCurrentMemberProfileRaw(
  id: string,
  payload: any,
): Promise<any | null> {
  const token = await auth.currentUser?.getIdToken();
  if (!token) return null;

  const url = `${API_BASE}/members/${encodeURIComponent(id)}`;
  console.log(
    "[authRepositoryHTTP] updateCurrentMemberProfileRaw PATCH",
    url,
    payload,
  );

  const res = await fetch(url, {
    method: "PATCH",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    console.warn(
      "[authRepositoryHTTP] updateCurrentMemberProfileRaw failed:",
      res.status,
      res.statusText,
      text,
    );
    return null;
  }

  const ct = res.headers.get("Content-Type") ?? "";
  if (!ct.includes("application/json")) {
    return null;
  }

  const raw = await res.json();
  return raw;
}

/**
 * companies/{id} を叩いて「生の JSON」を返す関数
 * - Firebase ログインユーザーから ID トークンを取得
 * - 404 などは null で返す（従来の companyService と同じ挙動）
 */
export async function fetchCompanyByIdRaw(
  companyId: string,
): Promise<any | null> {
  const id = (companyId ?? "").trim();
  if (!id) return null;

  const user = auth.currentUser;
  if (!user) {
    console.warn(
      "[authRepositoryHTTP] fetchCompanyByIdRaw called without logged-in user",
    );
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }

  const idToken = await user.getIdToken();

  const url = `${API_BASE}/companies/${encodeURIComponent(id)}`;
  console.log("[authRepositoryHTTP] fetchCompanyByIdRaw GET", url);

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
  });

  if (!res.ok) {
    console.error("[authRepositoryHTTP] GET /companies failed", {
      status: res.status,
      statusText: res.statusText,
    });
    return null;
  }

  const ct = res.headers.get("Content-Type") ?? "";
  if (!ct.includes("application/json")) {
    console.error(
      "[authRepositoryHTTP] companies API did not return JSON (content-type=%s)",
      ct,
    );
    return null;
  }

  const raw = await res.json();
  return raw;
}
