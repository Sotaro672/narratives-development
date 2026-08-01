// frontend/console/shell/src/auth/application/authService.ts

import {
  onAuthStateChanged,
  type User,
} from "firebase/auth";

import {
  auth,
} from "../infrastructure/config/firebaseClient";

/**
 * auth.currentUserを即時に取得できない場合に備えて、
 * onAuthStateChangedを一度だけ待つPromiseを共有する。
 */
let authReadyPromise: Promise<User | null> | null =
  null;

function waitForAuthReady(): Promise<User | null> {
  if (auth.currentUser) {
    return Promise.resolve(auth.currentUser);
  }

  if (!authReadyPromise) {
    authReadyPromise =
      new Promise<User | null>((resolve) => {
        const unsubscribe =
          onAuthStateChanged(
            auth,
            (user) => {
              unsubscribe();
              authReadyPromise = null;
              resolve(user ?? null);
            },
          );
      });
  }

  return authReadyPromise;
}

/**
 * 現在のFirebase Userを返す。
 *
 * auth.currentUserが未確定の場合は、
 * onAuthStateChangedを一度だけ待つ。
 */
export async function getCurrentUser(): Promise<User | null> {
  if (auth.currentUser) {
    return auth.currentUser;
  }

  return waitForAuthReady();
}

/**
 * 現在のFirebase UserからIDトークンを取得する。
 *
 * ユーザーが存在しない場合、またはトークン取得に失敗した場合は
 * nullを返す。
 */
export async function getIdToken(
  forceRefresh = false,
): Promise<string | null> {
  const user = await getCurrentUser();

  if (!user) {
    return null;
  }

  try {
    return await user.getIdToken(
      forceRefresh,
    );
  } catch {
    return null;
  }
}

/**
 * Backendリクエスト用のAuthorizationヘッダーを返す。
 *
 * IDトークンを取得できない場合は空オブジェクトを返す。
 */
export async function getAuthHeaders(
  forceRefresh = false,
): Promise<Record<string, string>> {
  const token = await getIdToken(
    forceRefresh,
  );

  if (!token) {
    return {};
  }

  return {
    Authorization: `Bearer ${token}`,
  };
}