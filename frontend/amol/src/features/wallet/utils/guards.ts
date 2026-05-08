// frontend/amol/src/features/wallet/utils/guards.ts
import type {
  AvatarResponse,
  AvatarStateResponse,
  PublicAvatarAggregateResponse,
} from "../types";

export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

export function isAvatarResponse(value: unknown): value is AvatarResponse {
  if (!isRecord(value)) return false;

  const avatar = value as Partial<AvatarResponse>;

  return (
    (typeof avatar.avatarId === "string" ||
      typeof avatar.avatarId === "undefined") &&
    (typeof avatar.userId === "string" ||
      typeof avatar.userId === "undefined") &&
    (typeof avatar.avatarName === "string" ||
      typeof avatar.avatarName === "undefined") &&
    (typeof avatar.avatarIcon === "string" ||
      avatar.avatarIcon === null ||
      typeof avatar.avatarIcon === "undefined") &&
    (typeof avatar.walletAddress === "string" ||
      typeof avatar.walletAddress === "undefined") &&
    (typeof avatar.profile === "string" ||
      avatar.profile === null ||
      typeof avatar.profile === "undefined")
  );
}

export function isAvatarStateResponse(
  value: unknown
): value is AvatarStateResponse {
  if (!isRecord(value)) return false;

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

export function isPublicAvatarAggregateResponse(
  value: unknown
): value is PublicAvatarAggregateResponse {
  if (!isRecord(value)) return false;

  const aggregate = value as Partial<PublicAvatarAggregateResponse>;

  return (
    (isAvatarResponse(aggregate.avatar) ||
      aggregate.avatar === null ||
      typeof aggregate.avatar === "undefined") &&
    (isAvatarStateResponse(aggregate.state) ||
      aggregate.state === null ||
      typeof aggregate.state === "undefined")
  );
}