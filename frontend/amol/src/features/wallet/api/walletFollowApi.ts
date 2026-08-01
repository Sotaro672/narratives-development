// frontend/amol/src/features/wallet/api/walletFollowApi.ts

import {
  isFiniteNumber,
  isRecord,
} from "../../shared/utils/typeGuards";
import type { AvatarStateResponse } from "../types";
import type { FetchPublicWalletFollowStateInput } from "../types/followTypes";
import { isAvatarStateResponse } from "../utils/guards";

type FollowAvatarInput = {
  backendUrl: string;
  idToken: string;
  targetAvatarId: string;
};

type PublicWalletFollowUser = {
  avatarId: string;
  avatarName: string;
  avatarIcon: string;
  followedAt: string;
};

type PublicWalletFollowState = {
  avatarId: string;
  followerCount: number;
  followingCount: number;
  postCount: number;
  followers: PublicWalletFollowUser[];
  following: PublicWalletFollowUser[];
  lastActiveAt: string;
  updatedAt: string;
};

function unwrapData(value: unknown): unknown {
  if (!isRecord(value)) {
    return value;
  }

  return value.data ?? value;
}

function extractErrorMessage(
  value: unknown,
): string {
  const body = unwrapData(value);

  if (!isRecord(body)) {
    return "";
  }

  const error =
    body.error ??
    body.message;

  return typeof error === "string"
    ? error
    : "";
}

function toStringValue(
  value: unknown,
): string {
  return String(value ?? "").trim();
}

function toNumberValue(
  value: unknown,
): number {
  if (isFiniteNumber(value)) {
    return value;
  }

  if (typeof value === "string") {
    return Number.parseInt(value, 10) || 0;
  }

  return 0;
}

function parsePublicWalletFollowUser(
  value: unknown,
): PublicWalletFollowUser | null {
  if (!isRecord(value)) {
    return null;
  }

  const avatarId =
    toStringValue(value.avatarId);

  if (!avatarId) {
    return null;
  }

  return {
    avatarId,
    avatarName:
      toStringValue(value.avatarName),
    avatarIcon:
      toStringValue(value.avatarIcon),
    followedAt:
      toStringValue(value.followedAt),
  };
}

function parsePublicWalletFollowUsers(
  value: unknown,
): PublicWalletFollowUser[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .map(parsePublicWalletFollowUser)
    .filter(
      (
        item,
      ): item is PublicWalletFollowUser =>
        item !== null,
    );
}

function parsePublicWalletFollowState(
  value: unknown,
  fallbackAvatarId: string,
): PublicWalletFollowState {
  const body = unwrapData(value);

  if (!isRecord(body)) {
    throw new Error(
      "フォロー情報APIのレスポンス形式が不正です。",
    );
  }

  return {
    avatarId:
      toStringValue(body.avatarId) ||
      fallbackAvatarId,
    followerCount:
      toNumberValue(body.followerCount),
    followingCount:
      toNumberValue(body.followingCount),
    postCount:
      toNumberValue(body.postCount),
    followers:
      parsePublicWalletFollowUsers(
        body.followers,
      ),
    following:
      parsePublicWalletFollowUsers(
        body.following,
      ),
    lastActiveAt:
      toStringValue(body.lastActiveAt),
    updatedAt:
      toStringValue(body.updatedAt),
  };
}

async function requestFollowState(
  method: "POST" | "DELETE",
  {
    backendUrl,
    idToken,
    targetAvatarId,
  }: FollowAvatarInput,
): Promise<AvatarStateResponse> {
  const normalizedTargetAvatarId =
    targetAvatarId.trim();

  if (!normalizedTargetAvatarId) {
    throw new Error(
      "targetAvatarId is required.",
    );
  }

  const response = await fetch(
    `${backendUrl}/mall/me/avatars/follow`,
    {
      method,
      headers: {
        Accept: "application/json",
        "Content-Type":
          "application/json; charset=utf-8",
        Authorization:
          `Bearer ${idToken}`,
      },
      body: JSON.stringify({
        targetAvatarId:
          normalizedTargetAvatarId,
      }),
    },
  );

  const contentType =
    response.headers.get(
      "content-type",
    ) || "";

  const responseBody: unknown =
    contentType.includes(
      "application/json",
    )
      ? await response.json()
      : null;

  if (!response.ok) {
    const message =
      extractErrorMessage(responseBody);

    throw new Error(
      message ||
        (method === "POST"
          ? "フォローに失敗しました。"
          : "フォロー解除に失敗しました。"),
    );
  }

  const body = unwrapData(responseBody);

  if (!isAvatarStateResponse(body)) {
    throw new Error(
      "フォローAPIのレスポンス形式が不正です。",
    );
  }

  return body;
}

export async function fetchPublicWalletFollowState({
  backendUrl,
  idToken,
  avatarId,
}: FetchPublicWalletFollowStateInput): Promise<PublicWalletFollowState> {
  const normalizedAvatarId =
    avatarId.trim();

  if (!normalizedAvatarId) {
    throw new Error(
      "avatarId is required.",
    );
  }

  const encodedAvatarId =
    encodeURIComponent(
      normalizedAvatarId,
    );

  const response = await fetch(
    `${backendUrl}/mall/avatars/${encodedAvatarId}/state`,
    {
      method: "GET",
      headers: {
        Accept: "application/json",
        Authorization:
          `Bearer ${idToken}`,
      },
    },
  );

  const contentType =
    response.headers.get(
      "content-type",
    ) || "";

  const responseBody: unknown =
    contentType.includes(
      "application/json",
    )
      ? await response.json()
      : null;

  if (!response.ok) {
    const message =
      extractErrorMessage(responseBody);

    throw new Error(
      message ||
        "フォロー情報の取得に失敗しました。",
    );
  }

  if (
    !contentType.includes(
      "application/json",
    )
  ) {
    throw new Error(
      "フォロー情報APIがJSON以外を返しました。",
    );
  }

  return parsePublicWalletFollowState(
    responseBody,
    normalizedAvatarId,
  );
}

export async function followAvatar(
  input: FollowAvatarInput,
): Promise<AvatarStateResponse> {
  return requestFollowState(
    "POST",
    input,
  );
}

export async function unfollowAvatar(
  input: FollowAvatarInput,
): Promise<AvatarStateResponse> {
  return requestFollowState(
    "DELETE",
    input,
  );
}