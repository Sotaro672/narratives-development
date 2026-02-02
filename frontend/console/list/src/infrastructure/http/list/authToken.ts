//frontend\console\list\src\infrastructure\http\list\authToken.ts
import { auth } from "../../../../../shell/src/auth/infrastructure/config/firebaseClient";
import { s } from "./string";

export async function getIdToken(): Promise<string> {
  const u = auth.currentUser;
  if (!u) throw new Error("not_authenticated");
  return await u.getIdToken();
}

/**
 * payload の createdBy / updatedBy に使う（必要な箇所のみ）
 */
export function getCurrentUserUid(): string {
  const u = auth.currentUser;
  return s(u?.uid);
}
