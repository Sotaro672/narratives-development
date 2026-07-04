// frontend/amol/src/features/catalog/infrastructure/avatarStateRepository.ts

import { getApiBaseUrl } from "../../../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../../../lib/authToken";
import { readResponseErrorMessage } from "./httpErrorReader";

type MeAvatarResponse = {
  avatarId?: string;
};

function unwrapData(value: unknown): unknown {
  if (!value || typeof value !== "object") {
    return value;
  }

  const record = value as Record<string, unknown>;

  return record.data ?? value;
}

function isMeAvatarResponse(value: unknown): value is MeAvatarResponse {
  if (!value || typeof value !== "object") {
    return false;
  }

  const record = value as Record<string, unknown>;

  return typeof record.avatarId === "string";
}

export async function fetchCurrentAvatarId(
  apiBaseUrl = getApiBaseUrl(),
): Promise<string> {
  const idToken = await getFirebaseIdToken();
  const base = apiBaseUrl.replace(/\/+$/, "");

  const response = await fetch(`${base}/mall/me/avatars`, {
    method: "GET",
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${idToken}`,
    },
    credentials: "include",
  });

  if (!response.ok) {
    const message = await readResponseErrorMessage(response);
    throw new Error(message || "現在のアバター情報の取得に失敗しました。");
  }

  const contentType = response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    throw new Error("現在のアバター情報APIがJSON以外を返しました。");
  }

  const responseBody: unknown = await response.json();
  const data = unwrapData(responseBody);

  if (!isMeAvatarResponse(data)) {
    throw new Error("現在のアバター情報APIのレスポンス形式が不正です。");
  }

  const avatarId = data.avatarId?.trim();

  if (!avatarId) {
    throw new Error("現在のavatarIdが見つかりません。");
  }

  return avatarId;
}