//frontend\amol\src\lib\apiBaseUrl.ts
export function getApiBaseUrl(): string {
  const env = import.meta.env.VITE_API_BASE_URL;

  if (typeof env === "string" && env.trim() !== "") {
    return env.replace(/\/+$/, "");
  }

  return "";
}