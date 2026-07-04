// frontend/amol/src/features/wallet/api/avatarApi.ts
import type {
  AvatarResponse,
  WalletAvatar,
} from "../types";
import {
  isAvatarResponse,
} from "../utils/guards";

type FetchWalletPageDataInput = {
  backendUrl: string;
  idToken: string;
};

type FetchPublicWalletAvatarInput = {
  backendUrl: string;
  idToken: string;
  avatarId: string;
};

function unwrapData(value: unknown): unknown {
  if (!value || typeof value !== "object") {
    return value;
  }

  const record = value as Record<string, unknown>;

  return record.data ?? value;
}

function getErrorMessageFromBody(value: unknown): string | null {
  const body = unwrapData(value);

  if (!body || typeof body !== "object") {
    return null;
  }

  const record = body as Record<string, unknown>;
  const error = record.error;

  return typeof error === "string" && error ? error : null;
}

function toWalletAvatar(params: {
  avatar?: AvatarResponse | null;
  fallbackAvatarId?: string;
}): WalletAvatar {
  const avatar = params.avatar;

  return {
    avatarId: avatar?.avatarId || params.fallbackAvatarId || "",
    avatarName: avatar?.avatarName || "",
    avatarIcon: avatar?.avatarIcon || "",
    profile: avatar?.profile || "",
    followerCount: 0,
    followingCount: 0,
  };
}

export async function fetchWalletAvatar({
  backendUrl,
  idToken,
}: FetchWalletPageDataInput): Promise<WalletAvatar> {
  const response = await fetch(`${backendUrl}/mall/me/avatars`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
    },
  });

  const contentType = response.headers.get("content-type") || "";

  if (!response.ok) {
    if (contentType.includes("application/json")) {
      const responseBody: unknown = await response.json();
      const error = getErrorMessageFromBody(responseBody);

      if (error) {
        throw new Error(error);
      }
    }

    throw new Error("アバター情報の取得に失敗しました。");
  }

  if (!contentType.includes("application/json")) {
    throw new Error("アバター情報APIがJSON以外を返しました。");
  }

  const responseBody: unknown = await response.json();
  const body = unwrapData(responseBody);

  if (!isAvatarResponse(body)) {
    throw new Error("アバター情報APIのレスポンス形式が不正です。");
  }

  return toWalletAvatar({
    avatar: body,
  });
}

export async function fetchPublicWalletAvatar({
  backendUrl,
  idToken,
  avatarId,
}: FetchPublicWalletAvatarInput): Promise<WalletAvatar> {
  const encodedAvatarId = encodeURIComponent(avatarId);

  const response = await fetch(`${backendUrl}/mall/avatars/${encodedAvatarId}`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
    },
  });

  const contentType = response.headers.get("content-type") || "";

  if (!response.ok) {
    if (contentType.includes("application/json")) {
      const responseBody: unknown = await response.json();
      const error = getErrorMessageFromBody(responseBody);

      if (error) {
        throw new Error(error);
      }
    }

    throw new Error("アバター情報の取得に失敗しました。");
  }

  if (!contentType.includes("application/json")) {
    throw new Error("アバター情報APIがJSON以外を返しました。");
  }

  const responseBody: unknown = await response.json();
  const body = unwrapData(responseBody);

  if (!isAvatarResponse(body)) {
    throw new Error("アバター情報APIのレスポンス形式が不正です。");
  }

  return toWalletAvatar({
    avatar: body,
    fallbackAvatarId: avatarId,
  });
}