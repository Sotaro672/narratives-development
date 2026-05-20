// frontend/console/tokenBlueprintReview/src/infrastructure/apiTypes.ts
// Raw API DTOs returned by backend.
//
// These represent backend JSON as-is.
// Firestore / backend response field names are treated as the source of truth.

import type {
  ReactionType,
  ActorType,
  AuthorType,
} from "../domain/entity";

export type ApiTokenBlueprintReviewAggregate = {
  TokenBlueprintID: string;
  LikeCount: number;
  DislikeCount: number;
  TopLevelCommentCount: number;
  TotalCommentCount: number;
  PinnedCommentID: string;
  CreatedAt: string;
  UpdatedAt: string;

  // GET /token-blueprint-reviews returns resolved display names in camelCase.
  tokenBlueprintName: string;
  brandName: string;
};

export type ApiTokenBlueprintReaction = {
  TokenBlueprintID: string;
  ActorID: string;
  ActorType: ActorType;
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
  AuthorType: AuthorType;
  IsOwnerComment: boolean;

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
  ActorID: string;
  ActorType: ActorType;
  Type: ReactionType;
  CreatedAt: string;
  UpdatedAt: string;
};