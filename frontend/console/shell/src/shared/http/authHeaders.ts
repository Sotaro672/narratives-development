// frontend/console/shell/src/shared/http/authHeaders.ts

import {
  onAuthStateChanged,
  type User,
} from "firebase/auth";

import {
  auth,
} from "../../auth/infrastructure/config/firebaseClient";

export type AuthTokenErrorCode =
  | "AUTH_NOT_AUTHENTICATED"
  | "AUTH_TOKEN_FETCH_FAILED"
  | "AUTH_TOKEN_EMPTY";

/**
 * 認証ヘッダー生成時の共通エラー。
 */
export class AuthTokenError extends Error {
  readonly code: AuthTokenErrorCode;

  readonly cause?: unknown;

  constructor(
    code: AuthTokenErrorCode,
    message: string,
    cause?: unknown,
  ) {
    super(message);

    this.name = "AuthTokenError";
    this.code = code;
    this.cause = cause;
  }
}

/**
 * auth.currentUserがまだ確定していない場合に、
 * onAuthStateChangedの初回通知を待つPromise。
 *
 * 複数のリクエストが同時に認証状態を待つ場合でも、
 * listenerを重複登録しないようPromiseを共有する。
 */
let authReadyPromise:
  | Promise<User | null>
  | null = null;

/**
 * Firebase Authenticationの初期状態が確定するまで待つ。
 */
function waitForAuthReady(): Promise<User | null> {
  if (auth.currentUser) {
    return Promise.resolve(
      auth.currentUser,
    );
  }

  if (authReadyPromise) {
    return authReadyPromise;
  }

  authReadyPromise =
    new Promise<User | null>(
      (
        resolve,
        reject,
      ) => {
        let unsubscribe =
          () => {};

        unsubscribe =
          onAuthStateChanged(
            auth,
            (user) => {
              unsubscribe();
              authReadyPromise = null;

              resolve(
                user ?? null,
              );
            },
            (error) => {
              unsubscribe();
              authReadyPromise = null;

              reject(error);
            },
          );
      },
    );

  return authReadyPromise;
}

/**
 * 現在のFirebase Userを取得する。
 *
 * auth.currentUserが未確定の場合は、
 * onAuthStateChangedの初回通知を待つ。
 */
export async function getCurrentUser(): Promise<User | null> {
  if (auth.currentUser) {
    return auth.currentUser;
  }

  return waitForAuthReady();
}

/**
 * 現在のFirebase UserからID tokenを取得する厳格版。
 *
 * @param forceRefresh
 * trueの場合、FirebaseへID tokenの強制更新を要求する。
 *
 * @throws AuthTokenError
 * - AUTH_NOT_AUTHENTICATED:
 *   ログインユーザーが存在しない
 * - AUTH_TOKEN_FETCH_FAILED:
 *   Firebaseからのtoken取得に失敗
 * - AUTH_TOKEN_EMPTY:
 *   tokenが空
 */
export async function getIdTokenOrThrow(
  forceRefresh = false,
): Promise<string> {
  let user: User | null;

  try {
    user =
      await getCurrentUser();
  } catch (error: unknown) {
    throw new AuthTokenError(
      "AUTH_TOKEN_FETCH_FAILED",
      "Firebase Authenticationの状態を確認できませんでした。",
      error,
    );
  }

  if (!user) {
    throw new AuthTokenError(
      "AUTH_NOT_AUTHENTICATED",
      "ログインユーザーが確認できません。",
    );
  }

  let token: string;

  try {
    token =
      await user.getIdToken(
        forceRefresh,
      );
  } catch (error: unknown) {
    throw new AuthTokenError(
      "AUTH_TOKEN_FETCH_FAILED",
      "Firebase ID tokenの取得に失敗しました。",
      error,
    );
  }

  if (!token) {
    throw new AuthTokenError(
      "AUTH_TOKEN_EMPTY",
      "Firebase ID tokenが空です。",
    );
  }

  return token;
}

/**
 * 現在のFirebase UserからID tokenを取得する任意認証版。
 *
 * 未認証またはtoken取得失敗時はnullを返す。
 * 公開APIなど、認証が必須ではない処理で使用する。
 */
export async function getIdToken(
  forceRefresh = false,
): Promise<string | null> {
  try {
    return await getIdTokenOrThrow(
      forceRefresh,
    );
  } catch {
    return null;
  }
}

/**
 * Backendリクエスト用のAuthorizationヘッダーを返す任意認証版。
 *
 * 未認証またはtoken取得失敗時は空オブジェクトを返す。
 */
export async function getAuthHeaders(
  forceRefresh = false,
): Promise<Record<string, string>> {
  const token =
    await getIdToken(
      forceRefresh,
    );

  if (!token) {
    return {};
  }

  return {
    Authorization:
      `Bearer ${token}`,
  };
}

/**
 * Backendリクエスト用のAuthorizationヘッダーを返す厳格版。
 *
 * 未認証またはtoken取得失敗時はAuthTokenErrorを送出する。
 */
export async function getAuthHeadersOrThrow(
  forceRefresh = false,
): Promise<Record<string, string>> {
  const token =
    await getIdTokenOrThrow(
      forceRefresh,
    );

  return {
    Authorization:
      `Bearer ${token}`,
  };
}

/**
 * AuthorizationとJSON Content-Typeを返す任意認証版。
 */
export async function getAuthJsonHeaders(
  forceRefresh = false,
): Promise<Record<string, string>> {
  const authHeaders =
    await getAuthHeaders(
      forceRefresh,
    );

  return {
    ...authHeaders,
    "Content-Type":
      "application/json",
  };
}

/**
 * AuthorizationとJSON Content-Typeを返す厳格版。
 */
export async function getAuthJsonHeadersOrThrow(
  forceRefresh = false,
): Promise<Record<string, string>> {
  const authHeaders =
    await getAuthHeadersOrThrow(
      forceRefresh,
    );

  return {
    ...authHeaders,
    "Content-Type":
      "application/json",
  };
}

/**
 * Authorizationヘッダーと追加ヘッダーを統合する任意認証版。
 *
 * extraにAuthorizationが指定されている場合は、
 * 呼出元の値を優先する。
 */
export async function withAuthHeaders(
  extra?: Record<string, string>,
  forceRefresh = false,
): Promise<Record<string, string>> {
  const authHeaders =
    await getAuthHeaders(
      forceRefresh,
    );

  return {
    ...authHeaders,
    ...(extra ?? {}),
  };
}

/**
 * Authorizationヘッダーと追加ヘッダーを統合する厳格版。
 *
 * extraにAuthorizationが指定されている場合は、
 * 呼出元の値を優先する。
 */
export async function withAuthHeadersOrThrow(
  extra?: Record<string, string>,
  forceRefresh = false,
): Promise<Record<string, string>> {
  const authHeaders =
    await getAuthHeadersOrThrow(
      forceRefresh,
    );

  return {
    ...authHeaders,
    ...(extra ?? {}),
  };
}