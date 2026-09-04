// frontend/amol/src/lib/authHeaders.ts

import {
  getFirebaseIdToken,
} from "./authToken";

import {
  auth,
} from "./firebase";

export async function getAuthHeaders(): Promise<
  Record<string, string>
> {
  const token =
    await getFirebaseIdToken();

  return {
    Authorization: `Bearer ${token}`,
  };
}

export async function getOptionalAuthHeaders(): Promise<
  Record<string, string> | undefined
> {
  if (!auth.currentUser) {
    return undefined;
  }

  return getAuthHeaders();
}