// frontend/amol/src/features/wallet/api/walletFollowApi.ts
import type { AvatarStateResponse } from "../types";
import type {
  FetchPublicWalletFollowStateInput,
  PublicWalletFollowState,
  PublicWalletFollowUser,
} from "../types/followTypes";

type FollowAvatarInput = {
  backendUrl: string;
  idToken: string;
  targetAvatarId: string;
};

function unwrapData(value: unknown): unknown {
  if (!value || typeof value !== "object") {
    return value;
  }

  const record = value as Record<string, unknown>;

  return record.data ?? value;
}

function extractErrorMessage(value: unknown): string {
  const body = unwrapData(value);

  if (!body || typeof body !== "object") {
    return "";
  }

  const record = body as Record<string, unknown>;
  const error = record.error ?? record.message;

  return typeof error === "string" ? error : "";
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function toStringValue(value: unknown): string {
  return (value ?? "").toString().trim();
}

function toNumberValue(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  if (typeof value === "string") {
    return Number.parseInt(value, 10) || 0;
  }

  return 0;
}

function parsePublicWalletFollowUser(
  value: unknown
): PublicWalletFollowUser | null {
  if (!isRecord(value)) {
    return null;
  }

  const avatarId = toStringValue(value.avatarId);

  if (!avatarId) {
    return null;
  }

  return {
    avatarId,
    avatarName: toStringValue(value.avatarName),
    avatarIcon: toStringValue(value.avatarIcon),
    followedAt: toStringValue(value.followedAt),
  };
}

function parsePublicWalletFollowUsers(value: unknown): PublicWalletFollowUser[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .map((item) => parsePublicWalletFollowUser(item))
    .filter((item): item is PublicWalletFollowUser => item !== null);
}

function parsePublicWalletFollowState(
  value: unknown,
  fallbackAvatarId: string
): PublicWalletFollowState {
  const body = unwrapData(value);

  if (!isRecord(body)) {
    throw new Error("フォロー情報APIのレスポンス形式が不正です。");
  }

  return {
    avatarId: toStringValue(body.avatarId) || fallbackAvatarId,
    followerCount: toNumberValue(body.followerCount),
    followingCount: toNumberValue(body.followingCount),
    postCount: toNumberValue(body.postCount),
    followers: parsePublicWalletFollowUsers(body.followers),
    following: parsePublicWalletFollowUsers(body.following),
    lastActiveAt: toStringValue(body.lastActiveAt),
    updatedAt: toStringValue(body.updatedAt),
  };
}

function isAvatarStateResponse(value: unknown): value is AvatarStateResponse {
  if (!value || typeof value !== "object") {
    return false;
  }

  const state = value as Partial<AvatarStateResponse>;

  return (
    (typeof state.avatarId === "string" ||
      typeof state.avatarId === "undefined") &&
    (typeof state.followerCount === "number" ||
      state.followerCount === null ||
      typeof state.followerCount === "undefined") &&
    (typeof state.followingCount === "number" ||
      state.followingCount === null ||
      typeof state.followingCount === "undefined") &&
    (typeof state.postCount === "number" ||
      state.postCount === null ||
      typeof state.postCount === "undefined")
  );
}

async function requestFollowState(
  method: "POST" | "DELETE",
  { backendUrl, idToken, targetAvatarId }: FollowAvatarInput
): Promise<AvatarStateResponse> {
  const normalizedTargetAvatarId = targetAvatarId.trim();

  if (!normalizedTargetAvatarId) {
    throw new Error("targetAvatarId is required.");
  }

  const response = await fetch(`${backendUrl}/mall/me/avatars/follow`, {
    method,
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json; charset=utf-8",
      Authorization: `Bearer ${idToken}`,
    },
    body: JSON.stringify({
      targetAvatarId: normalizedTargetAvatarId,
    }),
  });

  const contentType = response.headers.get("content-type") || "";
  const responseBody: unknown = contentType.includes("application/json")
    ? await response.json()
    : null;

  if (!response.ok) {
    const message = extractErrorMessage(responseBody);

    throw new Error(
      message ||
        (method === "POST"
          ? "フォローに失敗しました。"
          : "フォロー解除に失敗しました。")
    );
  }

  const body = unwrapData(responseBody);

  if (!isAvatarStateResponse(body)) {
    throw new Error("フォローAPIのレスポンス形式が不正です。");
  }

  return body;
}

export async function fetchPublicWalletFollowState({
  backendUrl,
  idToken,
  avatarId,
}: FetchPublicWalletFollowStateInput): Promise<PublicWalletFollowState> {
  const normalizedAvatarId = avatarId.trim();

  if (!normalizedAvatarId) {
    throw new Error("avatarId is required.");
  }

  const encodedAvatarId = encodeURIComponent(normalizedAvatarId);

  const response = await fetch(
    `${backendUrl}/mall/avatars/${encodedAvatarId}/state`,
    {
      method: "GET",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${idToken}`,
      },
    }
  );

  const contentType = response.headers.get("content-type") || "";
  const responseBody: unknown = contentType.includes("application/json")
    ? await response.json()
    : null;

  if (!response.ok) {
    const message = extractErrorMessage(responseBody);

    throw new Error(message || "フォロー情報の取得に失敗しました。");
  }

  if (!contentType.includes("application/json")) {
    throw new Error("フォロー情報APIがJSON以外を返しました。");
  }

  return parsePublicWalletFollowState(responseBody, normalizedAvatarId);
}

export async function followAvatar(
  input: FollowAvatarInput
): Promise<AvatarStateResponse> {
  return requestFollowState("POST", input);
}

export async function unfollowAvatar(
  input: FollowAvatarInput
): Promise<AvatarStateResponse> {
  return requestFollowState("DELETE", input);
}