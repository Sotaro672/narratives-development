// frontend/console/mintRequest/src/infrastructure/http/firebaseAuth.ts

import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

/**
 * Firebase Auth から ID トークンを取得（未ログインなら throw）
 */
export async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) {
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }
  return await user.getIdToken();
}
