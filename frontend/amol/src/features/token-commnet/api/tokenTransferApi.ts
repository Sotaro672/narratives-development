// frontend/amol/src/features/token-commnet/api/tokenTransferApi.ts
import { getAuth } from "firebase/auth";

import type {
  FetchTokenTransferFollowStateParams,
  TokenTransferFollowState,
  TokenTransferTargetAvatar,
  TransferTokenToAvatarParams,
  TransferTokenToAvatarResponse,
} from "../types/tokenTransferTypes";

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function unwrapData(value: unknown): unknown {
  if (!isRecord(value)) {
    return value;
  }

  return isRecord(value.data) ? value.data : value;
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

function toBooleanValue(value: unknown): boolean {
  if (typeof value === "boolean") {
    return value;
  }

  if (typeof value === "number") {
    return value !== 0;
  }

  const text = toStringValue(value).toLowerCase();

  return text === "true" || text === "1" || text === "yes";
}

function normalizeBackendUrl(backendUrl: string): string {
  return backendUrl.trim().replace(/\/+$/, "");
}

async function readJsonOrNull(response: Response): Promise<unknown> {
  const contentType = response.headers.get("content-type") || "";

  if (!contentType.includes("application/json")) {
    return null;
  }

  return response.json();
}

function extractErrorMessage(responseBody: unknown): string {
  if (!isRecord(responseBody)) {
    return "";
  }

  return toStringValue(responseBody.error || responseBody.message);
}

function parseTargetAvatar(value: unknown): TokenTransferTargetAvatar | null {
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

function parseTargetAvatars(value: unknown): TokenTransferTargetAvatar[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .map((item) => parseTargetAvatar(item))
    .filter((item): item is TokenTransferTargetAvatar => item !== null);
}

function parseFollowState(
  value: unknown,
  fallbackAvatarId: string
): TokenTransferFollowState {
  const body = unwrapData(value);

  if (!isRecord(body)) {
    throw new Error("フォロー情報APIのレスポンス形式が不正です。");
  }

  return {
    avatarId: toStringValue(body.avatarId) || fallbackAvatarId,
    followerCount: toNumberValue(body.followerCount),
    followingCount: toNumberValue(body.followingCount),
    followers: parseTargetAvatars(body.followers),
    following: parseTargetAvatars(body.following),
    updatedAt: toStringValue(body.updatedAt),
  };
}

function parseTransferResponse(value: unknown): TransferTokenToAvatarResponse {
  const body = unwrapData(value);

  if (!isRecord(body)) {
    throw new Error("トークン移譲APIのレスポンス形式が不正です。");
  }

  return {
    avatarId: toStringValue(body.avatarId),
    targetAvatarId: toStringValue(body.targetAvatarId),
    productId: toStringValue(body.productId),
    txSignature: toStringValue(body.txSignature),
    fromWallet: toStringValue(body.fromWallet),
    toWallet: toStringValue(body.toWallet),
    updatedToAddress: toBooleanValue(body.updatedToAddress),
    mintAddress: toStringValue(body.mintAddress),
    tokenBlueprintId: toStringValue(body.tokenBlueprintId),
  };
}

export async function getCurrentIdToken(): Promise<string> {
  const auth = getAuth();
  const user = auth.currentUser;

  if (!user) {
    throw new Error("ログインが必要です。");
  }

  return user.getIdToken();
}

export async function fetchTokenTransferFollowState({
  backendUrl,
  idToken,
  avatarId,
}: FetchTokenTransferFollowStateParams): Promise<TokenTransferFollowState> {
  const normalizedBackendUrl = normalizeBackendUrl(backendUrl);
  const normalizedAvatarId = avatarId.trim();

  if (!normalizedBackendUrl) {
    throw new Error("backendUrl が空です。");
  }

  if (!normalizedAvatarId) {
    throw new Error("avatarId が空です。");
  }

  const encodedAvatarId = encodeURIComponent(normalizedAvatarId);

  const response = await fetch(
    `${normalizedBackendUrl}/mall/avatars/${encodedAvatarId}/state`,
    {
      method: "GET",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${idToken}`,
      },
    }
  );

  const responseBody = await readJsonOrNull(response);

  if (!response.ok) {
    const message = extractErrorMessage(responseBody);

    if (message) {
      throw new Error(message);
    }

    throw new Error("フォロー情報の取得に失敗しました。");
  }

  return parseFollowState(responseBody, normalizedAvatarId);
}

export async function transferTokenToAvatar({
  backendUrl,
  idToken,
  productId,
  targetAvatarId,
}: TransferTokenToAvatarParams): Promise<TransferTokenToAvatarResponse> {
  const normalizedBackendUrl = normalizeBackendUrl(backendUrl);
  const normalizedProductId = productId.trim();
  const normalizedTargetAvatarId = targetAvatarId.trim();

  if (!normalizedBackendUrl) {
    throw new Error("backendUrl が空です。");
  }

  if (!normalizedProductId) {
    throw new Error("productId が空です。");
  }

  if (!normalizedTargetAvatarId) {
    throw new Error("渡す相手を選択してください。");
  }

  const response = await fetch(`${normalizedBackendUrl}/mall/me/contents/share`, {
    method: "POST",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      Authorization: `Bearer ${idToken}`,
    },
    body: JSON.stringify({
      productId: normalizedProductId,
      targetAvatarId: normalizedTargetAvatarId,
    }),
  });

  const responseBody = await readJsonOrNull(response);

  if (!response.ok) {
    const message = extractErrorMessage(responseBody);

    if (message) {
      throw new Error(message);
    }

    if (response.status === 401) {
      throw new Error("ログインが必要です。");
    }

    if (response.status === 503) {
      throw new Error("アバター情報を取得できませんでした。");
    }

    if (response.status === 404) {
      throw new Error("トークンまたはアバターが見つかりませんでした。");
    }

    throw new Error("トークンの移譲に失敗しました。");
  }

  return parseTransferResponse(responseBody);
}