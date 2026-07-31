// frontend/console/tokenBlueprintReview/src/infrastructure/mappers.ts
// API DTO -> domain model mappers

import type {
  AuthorType,
  Comment,
  TokenBlueprintReviewAggregate,
} from "../../../shared/types/tokenBlueprintReview";

import { validateAuthorType } from "../../../shared/types/tokenBlueprintReview";

import type {
  ApiComment,
  ApiTokenBlueprintReviewAggregate,
} from "./apiTypes";

function resolveCommentAuthorType(
  authorType: AuthorType,
): AuthorType {
  validateAuthorType(authorType);
  return authorType;
}

export function fromApiTokenBlueprintReviewAggregate(
  aggregate: ApiTokenBlueprintReviewAggregate,
): TokenBlueprintReviewAggregate {
  return {
    tokenBlueprintId: aggregate.TokenBlueprintID,
    tokenBlueprintName: aggregate.tokenBlueprintName,
    brandName: aggregate.brandName,
    likeCount: aggregate.LikeCount,
    dislikeCount: aggregate.DislikeCount,
    topLevelCommentCount: aggregate.TopLevelCommentCount,
    totalCommentCount: aggregate.TotalCommentCount,
    pinnedCommentId: aggregate.PinnedCommentID,
    createdAt: aggregate.CreatedAt,
    updatedAt: aggregate.UpdatedAt,
  };
}

export function fromApiComment(
  comment: ApiComment,
): Comment {
  return {
    commentId: comment.CommentID,
    tokenBlueprintId: comment.TokenBlueprintID,
    parentCommentId: comment.ParentCommentID,
    rootCommentId: comment.RootCommentID,
    depth: comment.Depth,

    authorId: comment.AuthorID,
    authorType: resolveCommentAuthorType(
      comment.AuthorType,
    ),
    isOwnerComment: comment.IsOwnerComment,

    body: comment.Body,
    likeCount: comment.LikeCount,
    dislikeCount: comment.DislikeCount,
    childCount: comment.ChildCount,
    deleted: comment.Deleted,

    createdAt: comment.CreatedAt,
    updatedAt: comment.UpdatedAt,

    authorAvatarName: comment.AuthorAvatarName,
    authorAvatarIcon: comment.AuthorAvatarIcon,

    brandName: comment.BrandName,
    brandIcon: comment.BrandIcon,
  };
}