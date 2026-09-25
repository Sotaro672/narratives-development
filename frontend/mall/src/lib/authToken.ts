// frontend/amol/src/lib/authToken.ts

import { waitForAuthReady } from "./authReady";

export async function getFirebaseIdToken(
  forceRefresh = false,
): Promise<string> {
  const user = await waitForAuthReady();

  if (!user) {
    throw new Error(
      "ログイン情報が見つかりません。再ログインしてください。",
    );
  }

  const token = await user.getIdToken(forceRefresh);
  const normalizedToken = token.trim();

  if (!normalizedToken) {
    throw new Error(
      "認証トークンを取得できませんでした。再ログインしてください。",
    );
  }

  return normalizedToken;
}