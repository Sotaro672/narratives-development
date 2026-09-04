//frontend\admin\shell\src\auth\application\adminAuth.ts
import {
  onAuthStateChanged,
  signInWithEmailAndPassword,
  signOut,
  type User,
} from "firebase/auth";

import {
  auth,
} from "../infrastructure/firebaseClient";

const ADMIN_EMAIL =
  "caotailangaogang@gmail.com";

export async function signInAdmin(
  email: string,
  password: string,
): Promise<User> {
  const credential =
    await signInWithEmailAndPassword(
      auth,
      email.trim(),
      password,
    );

  const user =
    credential.user;

  const authenticatedEmail =
    user.email?.trim().toLowerCase() ?? "";

  if (
    authenticatedEmail !==
    ADMIN_EMAIL
  ) {
    await signOut(auth);

    throw new Error(
      "Admin権限がありません。",
    );
  }

  return user;
}

export async function signOutAdmin(): Promise<void> {
  await signOut(auth);
}

export function observeAdminAuth(
  callback: (
    user: User | null,
  ) => void,
): () => void {
  return onAuthStateChanged(
    auth,
    callback,
  );
}