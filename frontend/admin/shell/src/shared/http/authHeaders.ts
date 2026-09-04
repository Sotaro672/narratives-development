//frontend\admin\shell\src\shared\http\authHeaders.ts
import {
  onAuthStateChanged,
  type User,
} from "firebase/auth";

import {
  auth,
} from "../../auth/infrastructure/firebaseClient";

function waitForAuthReady(): Promise<User | null> {
  if (auth.currentUser) {
    return Promise.resolve(
      auth.currentUser,
    );
  }

  return new Promise(
    (resolve, reject) => {
      let unsubscribe =
        () => {};

      unsubscribe =
        onAuthStateChanged(
          auth,
          (user) => {
            unsubscribe();
            resolve(
              user ?? null,
            );
          },
          (error) => {
            unsubscribe();
            reject(error);
          },
        );
    },
  );
}

export async function getAuthHeaders(): Promise<
  Record<string, string>
> {
  const user =
    auth.currentUser ??
    await waitForAuthReady();

  if (!user) {
    throw new Error(
      "ログインが必要です。",
    );
  }

  const token =
    await user.getIdToken();

  return {
    Authorization:
      `Bearer ${token}`,
  };
}