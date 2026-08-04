// frontend/amol/src/features/scan-result/infrastructure/scanResultHttp.ts

export function getAuthorizationHeader(
  headers?: HeadersInit,
): string {
  const normalizedHeaders =
    new Headers(headers);

  return (
    normalizedHeaders.get(
      "Authorization",
    ) ||
    normalizedHeaders.get(
      "authorization",
    ) ||
    ""
  ).trim();
}