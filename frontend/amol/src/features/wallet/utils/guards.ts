// frontend/amol/src/features/wallet/utils/guards.ts

import { isRecord } from "../../shared/utils/typeGuards";
import type { AvatarStateResponse } from "../types";

export function isAvatarStateResponse(
  value: unknown,
): value is AvatarStateResponse {
  if (!isRecord(value)) {
    return false;
  }

  const state =
    value as Partial<AvatarStateResponse>;

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