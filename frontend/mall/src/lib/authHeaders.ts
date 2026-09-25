// frontend/mall/src/lib/authHeaders.ts

import { waitForAuthReady } from "./authReady";
import { getFirebaseIdToken } from "./authToken";

export async function getAuthHeaders(): Promise<Record<string, string>> {
  await waitForAuthReady();

  const token = await getFirebaseIdToken();

  return {
    Authorization: `Bearer ${token}`,
  };
}

export async function getOptionalAuthHeaders(): Promise<
  Record<string, string> | undefined
> {
  const user = await waitForAuthReady();

  if (!user) {
    return undefined;
  }

  return getAuthHeaders();
}