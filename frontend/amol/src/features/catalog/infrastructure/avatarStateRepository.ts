// frontend/amol/src/features/catalog/infrastructure/avatarStateRepository.ts

import type { MeAvatarStateResponse } from "../types";
import { getFirebaseIdToken } from "./authTokenProvider";
import { readResponseErrorMessage } from "./httpErrorReader";

export async function fetchCurrentAvatarId(
  apiBaseUrl: string,
): Promise<string> {
  const idToken = await getFirebaseIdToken();

  const response = await fetch(`${apiBaseUrl}/mall/me/avatars/state`, {
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

  const data = (await response.json()) as Partial<MeAvatarStateResponse>;
  const avatarId = data.avatarId?.trim();

  if (!avatarId) {
    throw new Error("現在のavatarIdが見つかりません。");
  }

  return avatarId;
}