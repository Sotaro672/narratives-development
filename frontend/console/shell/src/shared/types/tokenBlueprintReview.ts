// frontend\console\shell\src\shared\types\tokenBlueprintReview.ts
// Domain models for TokenBlueprint Review (frontend)
//
// Policy:
// - Keep only app-internal models and domain-level constraints here.
// - No HTTP code.
// - No API raw DTOs.
// - No API -> domain mappers.

export type ReactionType = "comment" | "like" | "dislike";
export type AuthorType = "avatar" | "brand";

export type TokenBlueprintReviewAggregate = {
  tokenBlueprintId: string;
  tokenBlueprintName: string;
  brandName: string;
  likeCount: number;
  dislikeCount: number;
  topLevelCommentCount: number;
  totalCommentCount: number;
  pinnedCommentId: string;
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
  authorAvatarName: string;
  authorAvatarIcon: string | null;
  brandName: string;
  brandIcon: string | null;
};