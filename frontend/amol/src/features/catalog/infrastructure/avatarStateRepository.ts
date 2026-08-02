// frontend/amol/src/features/catalog/infrastructure/avatarStateRepository.ts

import { getApiBaseUrl } from "../../../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../../../lib/authToken";
import { getMyAvatar } from "../../avatar/api/avatarApi";

export async function fetchCurrentAvatarId(
  apiBaseUrl = getApiBaseUrl(),
): Promise<string> {
  const idToken = await getFirebaseIdToken();

  const avatar = await getMyAvatar({
    backendUrl: apiBaseUrl,
    idToken,
  });

  const avatarId = avatar?.avatarId.trim() ?? "";

  if (!avatarId) {
    throw new Error("現在のavatarIdが見つかりません。");
  }

  return avatarId;
}