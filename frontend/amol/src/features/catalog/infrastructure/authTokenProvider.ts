// frontend/amol/src/features/catalog/infrastructure/authTokenProvider.ts

import { getAuth } from "firebase/auth";

export async function getFirebaseIdToken(): Promise<string> {
  const auth = getAuth();
  const user = auth.currentUser;

  if (!user) {
    throw new Error("ログイン情報が見つかりません。再ログインしてください。");
  }

  return user.getIdToken();
}