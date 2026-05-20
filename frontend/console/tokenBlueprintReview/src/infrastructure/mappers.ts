// frontend/console/tokenBlueprintReview/src/infrastructure/mappers.ts
// API DTO -> domain model mappers

import type {
  Comment,
  CommentReaction,
  TokenBlueprintReaction,
  TokenBlueprintReviewAggregate,
  ReactionType,
  AuthorType,
  ActorType,
} from "../domain/entity";

import {
  validateReactionType,
  validateAuthorType,
  validateActorType,
} from "../domain/entity";

import type {
  ApiComment,
  ApiCommentReaction,
  ApiTokenBlueprintReaction,
  ApiTokenBlueprintReviewAggregate,
} from "./apiTypes";

function resolveCommentAuthorType(authorType: AuthorType): AuthorType {
  validateAuthorType(authorType);
  return authorType;
}

function resolveActorType(actorType: ActorType): ActorType {
  validateActorType(actorType);
  return actorType;
}

function resolveReactionType(type: ReactionType): ReactionType {
  validateReactionType(type);
  return type;
}

export function fromApiTokenBlueprintReviewAggregate(
  a: ApiTokenBlueprintReviewAggregate,
): TokenBlueprintReviewAggregate {
  return {
    tokenBlueprintId: a.TokenBlueprintID,
    tokenBlueprintName: a.tokenBlueprintName,
    brandName: a.brandName,
    likeCount: a.LikeCount,
    dislikeCount: a.DislikeCount,
    topLevelCommentCount: a.TopLevelCommentCount,
    totalCommentCount: a.TotalCommentCount,
    pinnedCommentId: a.PinnedCommentID,
    createdAt: a.CreatedAt,
    updatedAt: a.UpdatedAt,
  };
}

export function fromApiTokenBlueprintReaction(
  a: ApiTokenBlueprintReaction,
): TokenBlueprintReaction {
  return {
    tokenBlueprintId: a.TokenBlueprintID,
    actorId: a.ActorID,
    actorType: resolveActorType(a.ActorType),
    type: resolveReactionType(a.Type),
    createdAt: a.CreatedAt,
    updatedAt: a.UpdatedAt,
  };
}

export function fromApiComment(a: ApiComment): Comment {
  return {
    commentId: a.CommentID,
    tokenBlueprintId: a.TokenBlueprintID,
    parentCommentId: a.ParentCommentID,
    rootCommentId: a.RootCommentID,
    depth: a.Depth,

    authorId: a.AuthorID,
    authorType: resolveCommentAuthorType(a.AuthorType),
    isOwnerComment: a.IsOwnerComment,

    body: a.Body,
    likeCount: a.LikeCount,
    dislikeCount: a.DislikeCount,
    childCount: a.ChildCount,
    deleted: a.Deleted,

    createdAt: a.CreatedAt,
    updatedAt: a.UpdatedAt,

    authorAvatarName: a.AuthorAvatarName,
    authorAvatarIcon: a.AuthorAvatarIcon,

    brandName: a.BrandName,
    brandIcon: a.BrandIcon,
  };
}

export function fromApiCommentReaction(a: ApiCommentReaction): CommentReaction {
  return {
    tokenBlueprintId: a.TokenBlueprintID,
    commentId: a.CommentID,
    actorId: a.ActorID,
    actorType: resolveActorType(a.ActorType),
    type: resolveReactionType(a.Type),
    createdAt: a.CreatedAt,
    updatedAt: a.UpdatedAt,
  };
}