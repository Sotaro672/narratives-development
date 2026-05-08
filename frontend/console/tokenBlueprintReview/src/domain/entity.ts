// frontend/console/tokenBlueprintReview/src/domain/entity.ts
// Domain models for TokenBlueprint Review (frontend)
//
// Policy:
// - Keep only app-internal models and domain-level constraints here.
// - No HTTP code.
// - No API raw DTOs.
// - No API -> domain mappers.

export const ErrInvalidReactionType = "invalid reaction type" as const;
export const ErrInvalidAuthorType = "invalid author type" as const;
export const ErrInvalidActorType = "invalid actor type" as const;

export type ReactionType = "comment" | "like" | "dislike";

export const ReactionComment: ReactionType = "comment";
export const ReactionLike: ReactionType = "like";
export const ReactionDislike: ReactionType = "dislike";

export function validateReactionType(t: ReactionType): void {
  if (t !== ReactionComment && t !== ReactionLike && t !== ReactionDislike) {
    throw new Error(ErrInvalidReactionType);
  }
}

export type AuthorType = "avatar" | "brand";

export const AuthorTypeAvatar: AuthorType = "avatar";
export const AuthorTypeBrand: AuthorType = "brand";

export function validateAuthorType(t: AuthorType): void {
  if (t !== AuthorTypeAvatar && t !== AuthorTypeBrand) {
    throw new Error(ErrInvalidAuthorType);
  }
}

export type ActorType = "avatar" | "brand";

export const ActorTypeAvatar: ActorType = "avatar";
export const ActorTypeBrand: ActorType = "brand";

export function validateActorType(t: ActorType): void {
  if (t !== ActorTypeAvatar && t !== ActorTypeBrand) {
    throw new Error(ErrInvalidActorType);
  }
}

// ---------------------------
// Domain models (camelCase)
// ---------------------------

export type TokenBlueprintReviewAggregate = {
  tokenBlueprintId: string;
  tokenBlueprintName?: string;
  brandName?: string;
  likeCount: number;
  dislikeCount: number;
  topLevelCommentCount: number;
  totalCommentCount: number;
  pinnedCommentId: string;
  createdAt: string;
  updatedAt: string;
};

export type TokenBlueprintReaction = {
  tokenBlueprintId: string;
  actorId: string;
  actorType: ActorType;
  type: ReactionType;
  createdAt: string;
  updatedAt: string;
};

export type Comment = {
  commentId: string;
  tokenBlueprintId: string;
  parentCommentId: string;
  rootCommentId: string;
  depth: number;

  authorId: string;
  authorType: AuthorType;
  isOwnerComment: boolean;

  body: string;
  likeCount: number;
  dislikeCount: number;
  childCount: number;

  deleted: boolean;

  createdAt: string;
  updatedAt: string;

  authorAvatarName?: string;
  authorAvatarIcon?: string;

  brandName?: string;
  brandIcon?: string;
};

export type CommentReaction = {
  tokenBlueprintId: string;
  commentId: string;
  actorId: string;
  actorType: ActorType;
  type: ReactionType;
  createdAt: string;
  updatedAt: string;
};