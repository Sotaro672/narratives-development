/// <reference types="vite/client" />

// frontend\console\shell\src\auth\infrastructure\repository\authRepositoryHTTP.ts

import { auth } from "../config/firebaseClient";
import { buildConsoleUrl } from "../../../shared/http/apiBase";

// -------------------------------
// HTTP functions (Auth / Member / Company 用)
// -------------------------------

async function getIdToken(forceRefresh = false): Promise<string | null> {
  const token = await auth.currentUser?.getIdToken(forceRefresh);
  return token ?? null;
}

function buildAuthHeaders(token: string, initHeaders?: HeadersInit): Headers {
  const headers = new Headers(initHeaders);

  headers.set("Authorization", `Bearer ${token}`);

  if (!headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  return headers;
}

async function fetchJsonWithAuth(
  url: string,
  init: RequestInit = {},
): Promise<any | null> {
  const token = await getIdToken(false);
  if (!token) return null;

  let res = await fetch(url, {
    ...init,
    headers: buildAuthHeaders(token, init.headers),
  });

  if (res.status === 401) {
    const refreshedToken = await getIdToken(true);
    if (!refreshedToken) return null;

    res = await fetch(url, {
      ...init,
      headers: buildAuthHeaders(refreshedToken, init.headers),
    });
  }

  if (!res.ok) {
    await res.text().catch(() => "");
    return null;
  }

  const ct = res.headers.get("Content-Type") ?? "";
  if (!ct.includes("application/json")) {
    return null;
  }

  return await res.json();
}

/**
 * Authorization token から現在 member を取得して「生の JSON」を返す関数。
 *
 * Backend 側の使い分け:
 * - GET /members/me
 *   - ログイン中ユーザー自身の member 取得用
 *   - Firebase Auth UID は URL ではなく Authorization token から backend が取得する
 *
 * - PATCH /members/{docId}
 *   - Firestore members の docId 用
 */
export async function fetchCurrentMemberRaw(): Promise<any | null> {
  return await fetchJsonWithAuth(buildConsoleUrl("/members/me"), {
    method: "GET",
  });
}

/**
 * members/{docId} に PATCH する HTTP 関数。
 *
 * 注意:
 * - ここで渡す id は Firebase Auth UID ではなく、Firestore members の docId。
 * - fetchCurrentMemberRaw の response.id を使う。
 */
export async function updateCurrentMemberProfileRaw(
  id: string,
  payload: any,
): Promise<any | null> {
  const memberDocId = (id ?? "").trim();
  if (!memberDocId) return null;

  return await fetchJsonWithAuth(
    buildConsoleUrl(`/members/${encodeURIComponent(memberDocId)}`),
    {
      method: "PATCH",
      body: JSON.stringify(payload),
    },
  );
}

/**
 * companies/{id} を叩いて「生の JSON」を返す関数。
 */
export async function fetchCompanyByIdRaw(
  companyId: string,
): Promise<any | null> {
  const id = (companyId ?? "").trim();
  if (!id) return null;

  return await fetchJsonWithAuth(
    buildConsoleUrl(`/companies/${encodeURIComponent(id)}`),
    {
      method: "GET",
    },
  );
}