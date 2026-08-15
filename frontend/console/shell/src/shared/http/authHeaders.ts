// frontend/console/shell/src/shared/http/authHeaders.ts

import { onAuthStateChanged, type User } from "firebase/auth";

import { auth } from "../../auth/infrastructure/config/firebaseClient";

let authReadyPromise: Promise<User | null> | null = null;

/**
 * Firebase Authenticationの初期状態が確定するまで待つ。
 * 複数リクエストが同時に待機する場合はPromiseを共有する。
 */
function waitForAuthReady(): Promise<User | null> {
  if (auth.currentUser) {
    return Promise.resolve(auth.currentUser);
  }

  if (authReadyPromise) {
    return authReadyPromise;
  }

  authReadyPromise = new Promise<User | null>((resolve, reject) => {
    let unsubscribe = () => {};

    unsubscribe = onAuthStateChanged(
      auth,
      (user) => {
        unsubscribe();
        authReadyPromise = null;
        resolve(user ?? null);
      },
      (error) => {
        unsubscribe();
        authReadyPromise = null;
        reject(error);
      },
    );
  });

  return authReadyPromise;
}

/**
 * Backendリクエスト用のAuthorizationヘッダーを返す。
 * 未認証またはtoken取得失敗時はErrorを送出する。
 */
export async function getAuthHeaders(
  forceRefresh = false,
): Promise<Record<string, string>> {
  let user: User | null;

  try {
    user = auth.currentUser ?? (await waitForAuthReady());
  } catch {
    throw new Error("Firebase Authenticationの状態を確認できませんでした。");
  }

  if (!user) {
    throw new Error("ログインユーザーが確認できません。");
  }

  let token: string;

  try {
    token = await user.getIdToken(forceRefresh);
  } catch {
    throw new Error("Firebase ID tokenの取得に失敗しました。");
  }

  if (!token) {
    throw new Error("Firebase ID tokenが空です。");
  }

  return {
    Authorization: `Bearer ${token}`,
  };
}