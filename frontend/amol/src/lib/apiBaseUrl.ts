// frontend/amol/src/lib/apiBaseUrl.ts

export function getApiBaseUrl(): string {
  const value =
    import.meta.env.VITE_API_BASE_URL;

  if (typeof value !== "string") {
    return "";
  }

  return value
    .trim()
    .replace(/\/+$/, "");
}

export function getRequiredApiBaseUrl(): string {
  const baseUrl =
    getApiBaseUrl();

  if (!baseUrl) {
    throw new Error(
      "VITE_API_BASE_URLが設定されていません。",
    );
  }

  return baseUrl;
}

function joinPaths(
  basePath: string,
  path: string,
): string {
  if (
    !basePath ||
    basePath === "/"
  ) {
    return path.startsWith("/")
      ? path
      : `/${path}`;
  }

  if (
    !path ||
    path === "/"
  ) {
    return basePath;
  }

  if (
    basePath.endsWith("/") &&
    path.startsWith("/")
  ) {
    return (
      basePath +
      path.slice(1)
    );
  }

  if (
    !basePath.endsWith("/") &&
    !path.startsWith("/")
  ) {
    return `${basePath}/${path}`;
  }

  return basePath + path;
}

export function buildApiUrl(
  baseUrl: string,
  path: string,
): string {
  const base =
    baseUrl
      .trim()
      .replace(/\/+$/, "");

  if (!base) {
    throw new Error(
      "API baseが未設定です。",
    );
  }

  const url =
    new URL(base);

  url.pathname =
    joinPaths(
      url.pathname,
      path,
    );

  url.search = "";
  url.hash = "";

  return url.toString();
}

export function buildBackendUrl(
  path: string,
): string {
  return buildApiUrl(
    getRequiredApiBaseUrl(),
    path,
  );
}