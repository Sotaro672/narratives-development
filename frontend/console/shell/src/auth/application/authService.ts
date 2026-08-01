// frontend/console/shell/src/auth/application/authService.ts

import {
  onAuthStateChanged,
  type User,
} from "firebase/auth";

import {
  auth,
} from "../infrastructure/config/firebaseClient";

import {
  fetchCurrentMemberRaw,
} from "../infrastructure/repository/authRepositoryHTTP";

/**
 * auth.currentUser が即時に得られないケースに備えて、
 * 一度だけ onAuthStateChanged を待つ Promise をメモ化。
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
 * 未確定の場合はonAuthStateChangedを一度だけ待つ。
 */
export async function getCurrentUser(): Promise<User | null> {
  if (auth.currentUser) {
    return auth.currentUser;
  }

  return await waitForAuthReady();
}

/**
 * 現在ユーザーのIDトークンを取得する。
 * 取得できない場合はnullを返す。
 */
export async function getIdToken(
  forceRefresh = false,
): Promise<string | null> {
  const user = await getCurrentUser();

  if (!user?.getIdToken) {
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
 * Authorizationヘッダーを返す。
 * トークンを取得できない場合は空オブジェクトを返す。
 */
export async function getAuthHeaders(
  forceRefresh = false,
): Promise<Record<string, string>> {
  const token = await getIdToken(
    forceRefresh,
  );

  return token
    ? {
        Authorization: `Bearer ${token}`,
      }
    : {};
}

function sleep(
  milliseconds: number,
): Promise<void> {
  return new Promise((resolve) => {
    setTimeout(resolve, milliseconds);
  });
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

  const raw =
    await fetchCurrentMemberRaw();

  return raw?.data ?? raw ?? null;
}

/**
 * BackendのGET /members/meから現在のメンバーを取得する。
 *
 * 新規登録直後はFirebase Authenticationのユーザーが存在していても、
 * Backend側のMemberが作成直後または未反映の場合があるため、
 * 短時間だけ再試行する。
 */
export async function getCurrentMember(
  options?: {
    retries?: number;
    retryDelayMs?: number;
  },
): Promise<CurrentMemberResponse | null> {
  const retries =
    options?.retries ?? 5;

  const retryDelayMs =
    options?.retryDelayMs ?? 300;

  for (
    let attempt = 0;
    attempt <= retries;
    attempt += 1
  ) {
    try {
      const member =
        await fetchCurrentMemberOnce();

      if (
        member?.id &&
        member?.companyId
      ) {
        return member;
      }

      if (attempt < retries) {
        await sleep(
          retryDelayMs *
            (attempt + 1),
        );

        continue;
      }

      return member;
    } catch (error) {
      console.error(
        "[authService] getCurrentMember failed:",
        error,
      );

      if (attempt < retries) {
        await sleep(
          retryDelayMs *
            (attempt + 1),
        );

        continue;
      }

      return null;
    }
  }

  return null;
}