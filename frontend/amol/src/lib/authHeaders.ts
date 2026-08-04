// frontend/amol/src/lib/authHeaders.ts

import {
  getFirebaseIdToken,
} from "./authToken";

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
  try {
    return await getAuthHeaders();
  } catch {
    return undefined;
  }
}