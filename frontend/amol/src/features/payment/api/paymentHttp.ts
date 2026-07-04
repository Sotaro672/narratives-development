// frontend/amol/src/features/payment/api/paymentHttp.ts
import { getFirebaseIdToken } from "../../../lib/authToken";

export const API_BASE_URL = getApiBaseUrl();

export function getApiBaseUrl(): string {
  const env = import.meta.env.VITE_API_BASE_URL;

  if (typeof env === "string" && env.trim() !== "") {
    return env.replace(/\/+$/, "");
  }

  return "";
}

export function getResponseErrorMessage(
  body: unknown,
  fallback: string,
): string {
  if (!body || typeof body !== "object") {
    return fallback;
  }

  const errorBody = body as {
    error?: string;
    detail?: string;
    message?: string;
    errorMessage?: string;
  };

  return (
    errorBody.errorMessage ??
    errorBody.detail ??
    errorBody.message ??
    errorBody.error ??
    fallback
  );
}

export async function getAuthHeaders(): Promise<HeadersInit> {
  const idToken = await getFirebaseIdToken();

  return {
    Accept: "application/json",
    "Content-Type": "application/json",
    Authorization: `Bearer ${idToken}`,
  };
}

export async function parseJsonOrThrow<T>(response: Response): Promise<T> {
  const text = await response.text();

  let body: unknown = null;

  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      throw new Error("APIがJSON以外を返しました。");
    }
  }

  if (!response.ok) {
    throw new Error(
      getResponseErrorMessage(
        body,
        `APIエラーが発生しました。status=${response.status}`,
      ),
    );
  }

  return body as T;
}

export async function parseJsonOrNull<T>(
  response: Response,
): Promise<T | null> {
  const contentType = response.headers.get("content-type") || "";

  if (!contentType.includes("application/json")) {
    return null;
  }

  return (await response.json()) as T;
}