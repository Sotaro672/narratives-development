// frontend/amol/src/lib/authHeaders.ts

import {
  getFirebaseIdToken,
} from "./authToken";

export async function getOptionalAuthHeaders(): Promise<
  Record<string, string> | undefined
> {
  try {
    const token = (
      await getFirebaseIdToken()
    ).trim();

    if (!token) {
      return undefined;
    }

    return {
      Authorization: `Bearer ${token}`,
    };
  } catch {
    return undefined;
  }
}