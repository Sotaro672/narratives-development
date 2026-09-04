// frontend/admin/shell/src/auth/application/adminAuth.ts

import {
  onAuthStateChanged,
  sendEmailVerification,
  signInWithEmailAndPassword,
  signOut,
  type User,
} from "firebase/auth";

import { auth } from "../infrastructure/firebaseClient";

const BACKEND_BASE_URL = import.meta.env.VITE_BACKEND_BASE_URL
  ?.trim()
  .replace(/\/+$/, "");

function requireBackendBaseUrl(): string {
  if (!BACKEND_BASE_URL) {
    throw new Error("VITE_BACKEND_BASE_URL is not configured.");
  }

  return BACKEND_BASE_URL;
}

async function authorizeAdmin(user: User): Promise<void> {
  const idToken = await user.getIdToken();
  const backendBaseUrl = requireBackendBaseUrl();

  const response = await fetch(`${backendBaseUrl}/admin/me`, {
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

async function requireVerifiedEmail(user: User): Promise<void> {
  if (user.emailVerified) {
    return;
  }

  await sendEmailVerification(user);
  await signOut(auth);

  throw new Error(
    "メールアドレスが未認証です。認証メールを送信しました。メール内のリンクから認証してください。",
  );
}

export async function signInAdmin(
  email: string,
  password: string,
): Promise<User> {
  const credential = await signInWithEmailAndPassword(
    auth,
    email.trim(),
    password,
  );

  const user = credential.user;

  try {
    await requireVerifiedEmail(user);
    await authorizeAdmin(user);

    return user;
  } catch (error) {
    if (auth.currentUser) {
      await signOut(auth);
    }

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

    if (!user.emailVerified) {
      void signOut(auth)
        .catch((error) => {
          console.error("[admin-auth] sign out failed", error);
        })
        .finally(() => {
          callback(null);
        });

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