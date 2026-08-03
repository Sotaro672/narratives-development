// frontend/amol/src/features/scan-result/infrastructure/scanResultHttp.ts

export function jsonHeaders(): HeadersInit {
  return {
    Accept: "application/json",
  };
}

export function jsonPostHeaders(): HeadersInit {
  return {
    Accept: "application/json",
    "Content-Type": "application/json",
  };
}

export function mergeHeaders(
  base: HeadersInit,
  extra?: HeadersInit,
): Headers {
  const headers = new Headers(base);

  if (extra) {
    new Headers(extra).forEach(
      (value, key) => {
        headers.set(key, value);
      },
    );
  }

  return headers;
}

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