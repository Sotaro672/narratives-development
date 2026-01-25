//frontend\console\production\src\infrastructure\auth\firebaseAuth.ts
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

/**
 * Firebase currentUser を返す（未ログインなら null）
 */
export function getCurrentUser() {
  return auth.currentUser;
}

/**
 * ID Token を取得（未ログインなら例外）
 */
export async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) throw new Error("未ログインです");
  return user.getIdToken();
}
