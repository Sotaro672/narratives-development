// frontend/amol/src/features/avatar/api/avatarApi.ts

import {
  HttpError,
  requestJson,
} from "../../../lib/http";

import type {
  AvatarMutationResponse,
  CreateAvatarPayload,
  MyAvatarResponse,
  UpdateAvatarPayload,
} from "../../shared/types/avatar";

type AuthedRequestParams = {
  backendUrl: string;
  idToken: string;
};

type GetPublicAvatarParams = AuthedRequestParams & {
  avatarId: string;
};

export async function getMyAvatar({
  backendUrl,
  idToken,
}: AuthedRequestParams): Promise<MyAvatarResponse | null> {
  void backendUrl;
  void idToken;

  try {
    return await requestJson<MyAvatarResponse>(
      "/mall/me/avatars",
      {
        method: "GET",
        auth: "required",
        unwrapData: true,
      },
    );
  } catch (error) {
    if (
      error instanceof HttpError &&
      error.status === 404
    ) {
      return null;
    }

    throw error;
  }
}

export async function getPublicAvatar({
  backendUrl,
  idToken,
  avatarId,
}: GetPublicAvatarParams): Promise<MyAvatarResponse | null> {
  void backendUrl;
  void idToken;

  const normalizedAvatarId =
    avatarId.trim();

  if (!normalizedAvatarId) {
    throw new Error(
      "avatarIdが指定されていません。",
    );
  }

  try {
    const avatar =
      await requestJson<MyAvatarResponse>(
        `/mall/avatars/${encodeURIComponent(
          normalizedAvatarId,
        )}`,
        {
          method: "GET",
          auth: "required",
          unwrapData: true,
        },
      );

    return {
      ...avatar,
      avatarId:
        avatar.avatarId ||
        normalizedAvatarId,
    };
  } catch (error) {
    if (
      error instanceof HttpError &&
      error.status === 404
    ) {
      return null;
    }

    throw error;
  }
}

export async function createAvatar({
  backendUrl,
  idToken,
  payload,
}: AuthedRequestParams & {
  payload: CreateAvatarPayload;
}): Promise<AvatarMutationResponse> {
  void backendUrl;
  void idToken;

  return requestJson<AvatarMutationResponse>(
    "/mall/avatars",
    {
      method: "POST",
      auth: "required",
      json: payload,
      unwrapData: true,
    },
  );
}

export async function updateAvatar({
  backendUrl,
  idToken,
  avatarId,
  payload,
}: AuthedRequestParams & {
  avatarId: string;
  payload: UpdateAvatarPayload;
}): Promise<AvatarMutationResponse> {
  void backendUrl;
  void idToken;

  return requestJson<AvatarMutationResponse>(
    "/mall/me/avatars",
    {
      method: "PATCH",
      auth: "required",
      json: payload,
      unwrapData: true,
      fallbackValue: {
        avatarId,
      } as AvatarMutationResponse,
    },
  );
}