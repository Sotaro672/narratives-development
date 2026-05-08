// frontend/console/tokenBlueprintReview/src/infrastructure/apiTypes.ts
// Raw API DTOs returned by backend.
//
// These represent backend JSON as-is.

import type { ReactionType, ActorType } from "../domain/entity";

export type ApiTokenBlueprintReviewAggregate = {
  TokenBlueprintID: string;
  LikeCount: number;
  DislikeCount: number;
  TopLevelCommentCount: number;
  TotalCommentCount: number;
  PinnedCommentID?: string;
  CreatedAt: string;
  UpdatedAt: string;

  // backend may attach these in camelCase
  tokenBlueprintName?: string;
  brandName?: string;
  pinnedCommentId?: string;
};

export type ApiTokenBlueprintReaction = {
  TokenBlueprintID: string;
  ActorID?: string;
  ActorType?: ActorType;
  AvatarID?: string;
  Type: ReactionType;
  CreatedAt: string;
  UpdatedAt: string;
};

export type ApiComment = {
  CommentID: string;
  TokenBlueprintID: string;
  ParentCommentID: string;
  RootCommentID: string;
  Depth: number;

  AuthorID: string;
  AuthorType: "avatar" | "brand";
  IsOwnerComment?: boolean;
  isOwnerComment?: boolean;

  AuthorAvatarName?: string;
  AuthorAvatarIcon?: string;

  BrandName?: string;
  BrandIcon?: string;

  Body: string;
  LikeCount: number;
  DislikeCount: number;
  ChildCount: number;
  Deleted: boolean;

  CreatedAt: string;
  UpdatedAt: string;
};

export type ApiCommentReaction = {
  TokenBlueprintID: string;
  CommentID: string;
  ActorID?: string;
  ActorType?: ActorType;
  AvatarID?: string;
  Type: ReactionType;
  CreatedAt: string;
  UpdatedAt: string;
};