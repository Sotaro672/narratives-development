// frontend/amol/src/features/avatar/api/avatarApi.ts

import { HttpError, requestJson } from "../../../lib/http";
import type {
  AvatarMutationResponse,
  AvatarPayloadBase,
  CreateAvatarPayload,
  MyAvatarResponse,
} from "../../shared/types/avatar";

export async function getMyAvatar(): Promise<MyAvatarResponse | null> {
  try {
    return await requestJson<MyAvatarResponse>("/mall/me/avatars", {
      method: "GET",
      auth: "required",
    });
  } catch (error) {
    if (error instanceof HttpError && error.status === 404) return null;
    throw error;
  }
}

export async function getPublicAvatar({ avatarId }: { avatarId: string }): Promise<MyAvatarResponse | null> {
  if (!avatarId) throw new Error("avatarIdが指定されていません。");

  try {
    return await requestJson<MyAvatarResponse>(
      `/mall/avatars/${encodeURIComponent(avatarId)}`,
      { method: "GET" },
    );
  } catch (error) {
    if (error instanceof HttpError && error.status === 404) return null;
    throw error;
  }
}

export async function createAvatar({
  payload,
}: {
  payload: CreateAvatarPayload;
}): Promise<AvatarMutationResponse> {
  return requestJson<AvatarMutationResponse>("/mall/avatars", {
    method: "POST",
    auth: "required",
    json: payload,
  });
}

export async function updateAvatar({
  payload,
}: {
  payload: AvatarPayloadBase;
}): Promise<AvatarMutationResponse> {
  return requestJson<AvatarMutationResponse>("/mall/me/avatars", {
    method: "PATCH",
    auth: "required",
    json: payload,
  });
}