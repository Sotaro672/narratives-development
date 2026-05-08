// frontend/amol/src/features/wallet/api/avatarApi.ts
import type {
  AvatarResponse,
  AvatarStateResponse,
  PublicAvatarAggregateResponse,
  WalletAvatar,
} from "../types";
import {
  isAvatarResponse,
  isAvatarStateResponse,
  isPublicAvatarAggregateResponse,
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
  state?: AvatarStateResponse | null;
  fallbackAvatarId?: string;
}): WalletAvatar {
  const avatar = params.avatar;
  const state = params.state;

  return {
    avatarId: avatar?.avatarId || state?.avatarId || params.fallbackAvatarId || "",
    avatarName: avatar?.avatarName || "",
    avatarIcon: avatar?.avatarIcon || "",
    profile: avatar?.profile || "",
    followerCount: state?.followerCount ?? 0,
    followingCount: state?.followingCount ?? 0,
  };
}

export async function fetchWalletAvatar({
  backendUrl,
  idToken,
}: FetchWalletPageDataInput): Promise<WalletAvatar> {
  const [avatarResponse, avatarStateResponse] = await Promise.all([
    fetch(`${backendUrl}/mall/me/avatars`, {
      method: "GET",
      headers: {
        Authorization: `Bearer ${idToken}`,
      },
    }),
    fetch(`${backendUrl}/mall/me/avatars/state`, {
      method: "GET",
      headers: {
        Authorization: `Bearer ${idToken}`,
      },
    }),
  ]);

  let avatar: AvatarResponse | null = null;
  let avatarState: AvatarStateResponse | null = null;

  const avatarContentType = avatarResponse.headers.get("content-type") || "";

  if (avatarResponse.ok && avatarContentType.includes("application/json")) {
    const responseBody: unknown = await avatarResponse.json();
    const avatarBody = unwrapData(responseBody);

    if (isAvatarResponse(avatarBody)) {
      avatar = avatarBody;
    }
  }

  const avatarStateContentType =
    avatarStateResponse.headers.get("content-type") || "";

  if (
    avatarStateResponse.ok &&
    avatarStateContentType.includes("application/json")
  ) {
    const responseBody: unknown = await avatarStateResponse.json();
    const avatarStateBody = unwrapData(responseBody);

    if (isAvatarStateResponse(avatarStateBody)) {
      avatarState = avatarStateBody;
    }
  }

  return toWalletAvatar({
    avatar,
    state: avatarState,
  });
}

export async function fetchPublicWalletAvatar({
  backendUrl,
  idToken,
  avatarId,
}: FetchPublicWalletAvatarInput): Promise<WalletAvatar> {
  const encodedAvatarId = encodeURIComponent(avatarId);

  const response = await fetch(
    `${backendUrl}/mall/avatars/${encodedAvatarId}?aggregate=1`,
    {
      method: "GET",
      headers: {
        Authorization: `Bearer ${idToken}`,
      },
    }
  );

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

  if (!isPublicAvatarAggregateResponse(body)) {
    throw new Error("アバター情報APIのレスポンス形式が不正です。");
  }

  const aggregate: PublicAvatarAggregateResponse = body;

  return toWalletAvatar({
    avatar: aggregate.avatar ?? null,
    state: aggregate.state ?? null,
    fallbackAvatarId: avatarId,
  });
}