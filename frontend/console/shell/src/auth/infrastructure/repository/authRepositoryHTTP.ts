/// <reference types="vite/client" />

// frontend/console/shell/src/auth/infrastructure/repository/authRepositoryHTTP.ts

import { auth } from "../config/firebaseClient";
import { buildConsoleUrl } from "../../../shared/http/apiBase";

// -------------------------------
// HTTP functions (Auth / Member / Company 用)
// -------------------------------

/**
 * Firebase Auth UID から現在 member を取得して「生の JSON」を返す関数。
 *
 * Backend 側の使い分け:
 * - GET /members/{uid}
 *   - Firebase Auth UID 用
 *
 * - PATCH /members/{docId}
 *   - Firestore members の docId 用
 */
export async function fetchCurrentMemberRaw(
  uid: string,
): Promise<any | null> {
  const token = await auth.currentUser?.getIdToken();
  if (!token) return null;

  const firebaseUid = (uid ?? "").trim();
  if (!firebaseUid) return null;

  const url = buildConsoleUrl(`/members/${encodeURIComponent(firebaseUid)}`);

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
  });

  if (!res.ok) {
    await res.text().catch(() => "");
    return null;
  }

  const ct = res.headers.get("Content-Type") ?? "";
  if (!ct.includes("application/json")) {
    throw new Error(
      `members API が JSON を返していません (content-type=${ct}). ` +
        `VITE_BACKEND_BASE_URL または buildConsoleUrl の設定を確認してください。`,
    );
  }

  const raw = await res.json();
  return raw;
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
  const token = await auth.currentUser?.getIdToken();
  if (!token) return null;

  const memberDocId = (id ?? "").trim();
  if (!memberDocId) return null;

  const url = buildConsoleUrl(`/members/${encodeURIComponent(memberDocId)}`);

  const res = await fetch(url, {
    method: "PATCH",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  if (!res.ok) {
    await res.text().catch(() => "");
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
 */
export async function fetchCompanyByIdRaw(
  companyId: string,
): Promise<any | null> {
  const id = (companyId ?? "").trim();
  if (!id) return null;

  const user = auth.currentUser;
  if (!user) {
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }

  const idToken = await user.getIdToken();

  const url = buildConsoleUrl(`/companies/${encodeURIComponent(id)}`);

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
  });

  if (!res.ok) {
    await res.text().catch(() => "");
    return null;
  }

  const ct = res.headers.get("Content-Type") ?? "";
  if (!ct.includes("application/json")) {
    return null;
  }

  const raw = await res.json();
  return raw;
}