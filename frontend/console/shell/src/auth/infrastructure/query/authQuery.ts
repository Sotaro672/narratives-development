// authQuery.ts（インフラ・クエリ）

import { onAuthStateChanged, type User } from "firebase/auth";
import { auth } from "../config/firebaseClient";

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

export async function getCurrentUser(): Promise<User | null> {
  if (auth.currentUser) return auth.currentUser;
  return await waitForAuthReady();
}

export async function getIdToken(): Promise<string | null> {
  const u = await getCurrentUser();
  if (!u?.getIdToken) return null;
  try {
    return await u.getIdToken(false);
  } catch {
    return null;
  }
}

export async function getAuthHeaders(): Promise<Record<string, string>> {
  const token = await getIdToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}
