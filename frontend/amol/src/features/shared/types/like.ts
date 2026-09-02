// frontend/amol/src/features/shared/types/like.ts

import type { PageResult } from "../pageResult";

export type LikeTargetType = "list" | "resale";

export type LikeEntity = {
  avatarId: string;
  targetType: LikeTargetType;
  targetId: string;
  createdAt: string;
};

export type LikeStatus = {
  targetType: LikeTargetType;
  targetId: string;
  liked: boolean;
};

export type LikePage = PageResult<LikeEntity>;

export type FetchLikesParams = {
  targetType?: LikeTargetType;
  page?: number;
  perPage?: number;
};