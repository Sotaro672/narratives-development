// frontend/console/shell/src/auth/infrastructure/repository/authRepositoryHTTP.ts

import {
  buildConsoleUrl,
} from "../../../shared/http/apiBase";

import {
  fetchJSON,
} from "../../../shared/http/fetchJSON";

/**
 * JSONオブジェクトの共通型。
 */
type JsonObject =
  Record<string, unknown>;

/**
 * 認証必須APIへリクエストし、
 * 成功時はJSONレスポンスを返す。
 *
 * ID token取得、Authorizationヘッダー設定、
 * 401時のtoken強制更新と1回限りの再送は、
 * shared/http/fetchJSON.tsへ委譲する。
 *
 * 既存のRepository契約を維持するため、
 * 認証失敗・通信失敗・非2xxレスポンス・
 * JSON解析失敗時はnullを返す。
 */
async function requestAuthenticatedJsonOrNull<
  T,
>(
  url: string,
  init: RequestInit = {},
): Promise<T | null> {
  try {
    return await fetchJSON<T>(
      url,
      {
        ...init,
        auth: "required",
      },
    );
  } catch {
    return null;
  }
}

/**
 * Authorization tokenから現在のmemberを取得し、
 * Backendから返された生のJSONを返す。
 *
 * Backend側の使い分け:
 *
 * GET /members/me
 * - ログイン中ユーザー自身のmember取得用
 * - Firebase Auth UIDはURLではなく
 *   Authorization tokenからBackendが取得する
 *
 * PATCH /members/{docId}
 * - Firestore membersのdocument IDを使用する
 */
export async function fetchCurrentMemberRaw():
  Promise<JsonObject | null> {
  return requestAuthenticatedJsonOrNull<
    JsonObject
  >(
    buildConsoleUrl(
      "/members/me",
    ),
    {
      method: "GET",
    },
  );
}

/**
 * members/{docId}へPATCHする。
 *
 * 注意:
 * - idはFirebase Auth UIDではない
 * - Firestore membersのdocument IDを渡す
 * - fetchCurrentMemberRaw()のresponse.idを使用する
 */
export async function updateCurrentMemberProfileRaw(
  id: string,
  payload: JsonObject,
): Promise<JsonObject | null> {
  const memberDocId =
    String(
      id ?? "",
    ).trim();

  if (!memberDocId) {
    return null;
  }

  return requestAuthenticatedJsonOrNull<
    JsonObject
  >(
    buildConsoleUrl(
      `/members/${encodeURIComponent(memberDocId)}`,
    ),
    {
      method: "PATCH",
      headers: {
        "Content-Type":
          "application/json",
      },
      body: JSON.stringify(
        payload,
      ),
    },
  );
}

/**
 * companies/{id}を取得し、
 * Backendから返された生のJSONを返す。
 */
export async function fetchCompanyByIdRaw(
  companyId: string,
): Promise<JsonObject | null> {
  const id =
    String(
      companyId ?? "",
    ).trim();

  if (!id) {
    return null;
  }

  return requestAuthenticatedJsonOrNull<
    JsonObject
  >(
    buildConsoleUrl(
      `/companies/${encodeURIComponent(id)}`,
    ),
    {
      method: "GET",
    },
  );
}