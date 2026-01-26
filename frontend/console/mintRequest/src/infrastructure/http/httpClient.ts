// frontend/console/mintRequest/src/infrastructure/http/httpClient.ts

/**
 * 共通: Authorization / Content-Type 付与
 */
export function buildHeaders(idToken: string): HeadersInit {
  return {
    Authorization: `Bearer ${idToken}`,
    "Content-Type": "application/json",
  };
}
