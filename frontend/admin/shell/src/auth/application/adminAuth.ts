// frontend/admin/shell/src/auth/application/adminAuth.ts
import {
  onAuthStateChanged,
  signInWithEmailAndPassword,
  signOut,
  type User,
} from "firebase/auth";

import { auth } from "../infrastructure/firebaseClient";

const FALLBACK_BACKEND_BASE_URL =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

const BACKEND_BASE_URL = (
  import.meta.env.VITE_BACKEND_BASE_URL ||
  FALLBACK_BACKEND_BASE_URL
)
  .trim()
  .replace(/\/+$/, "");

async function authorizeAdmin(user: User): Promise<void> {
  const idToken = await user.getIdToken();

  const response = await fetch(`${BACKEND_BASE_URL}/admin/me`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      Accept: "application/json",
    },
  });

  if (!response.ok) {
    throw new Error(`Admin authorization failed. status=${response.status}`);
  }
}

export async function signInAdmin(email: string, password: string): Promise<User> {
  const credential = await signInWithEmailAndPassword(
    auth,
    email.trim(),
    password,
  );

  const user = credential.user;

  try {
    await authorizeAdmin(user);
    return user;
  } catch (error) {
    await signOut(auth);
    throw error;
  }
}

export async function signOutAdmin(): Promise<void> {
  await signOut(auth);
}

export function observeAdminAuth(
  callback: (user: User | null) => void,
): () => void {
  return onAuthStateChanged(auth, (user) => {
    if (!user) {
      callback(null);
      return;
    }

    void authorizeAdmin(user)
      .then(() => {
        callback(user);
      })
      .catch(async (error) => {
        console.error("[admin-auth] authorization failed", error);

        try {
          await signOut(auth);
        } finally {
          callback(null);
        }
      });
  });
}