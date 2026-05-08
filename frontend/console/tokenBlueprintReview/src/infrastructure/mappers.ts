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
  ReactionComment,
  validateReactionType,
  validateAuthorType,
  validateActorType,
  AuthorTypeAvatar,
  AuthorTypeBrand,
  ActorTypeAvatar,
  ActorTypeBrand,
} from "../domain/entity";

import type {
  ApiComment,
  ApiCommentReaction,
  ApiTokenBlueprintReaction,
  ApiTokenBlueprintReviewAggregate,
} from "./apiTypes";

function toOptionalString(v: unknown): string | undefined {
  return v != null && String(v) !== "" ? String(v) : undefined;
}

function resolveCommentAuthorType(a: ApiComment): AuthorType {
  const raw = toOptionalString(a?.AuthorType);

  if (raw === AuthorTypeAvatar || raw === AuthorTypeBrand) {
    validateAuthorType(raw);
    return raw;
  }

  return AuthorTypeAvatar;
}

function resolveActorType(raw: unknown): ActorType {
  const v = toOptionalString(raw);

  if (v === ActorTypeAvatar || v === ActorTypeBrand) {
    validateActorType(v);
    return v;
  }

  return ActorTypeAvatar;
}

export function fromApiTokenBlueprintReviewAggregate(
  a: ApiTokenBlueprintReviewAggregate,
): TokenBlueprintReviewAggregate {
  const raw = a as ApiTokenBlueprintReviewAggregate & {
    PinnedCommentID?: string;
    pinnedCommentId?: string;
  };

  return {
    tokenBlueprintId: String(a?.TokenBlueprintID ?? ""),
    tokenBlueprintName:
      a?.tokenBlueprintName != null && String(a.tokenBlueprintName) !== ""
        ? String(a.tokenBlueprintName)
        : undefined,
    brandName:
      a?.brandName != null && String(a.brandName) !== ""
        ? String(a.brandName)
        : undefined,
    likeCount: Number(a?.LikeCount ?? 0),
    dislikeCount: Number(a?.DislikeCount ?? 0),
    topLevelCommentCount: Number(a?.TopLevelCommentCount ?? 0),
    totalCommentCount: Number(a?.TotalCommentCount ?? 0),
    pinnedCommentId: String(raw?.PinnedCommentID ?? raw?.pinnedCommentId ?? ""),
    createdAt: String(a?.CreatedAt ?? ""),
    updatedAt: String(a?.UpdatedAt ?? ""),
  };
}

export function fromApiTokenBlueprintReaction(
  a: ApiTokenBlueprintReaction,
): TokenBlueprintReaction {
  const type = (a?.Type ?? ReactionComment) as ReactionType;
  validateReactionType(type);

  const raw = a as ApiTokenBlueprintReaction & {
    ActorID?: string;
    ActorType?: string;
    actorId?: string;
    actorType?: string;
    AvatarID?: string;
  };

  return {
    tokenBlueprintId: String(a?.TokenBlueprintID ?? ""),
    actorId: String(raw?.ActorID ?? raw?.actorId ?? raw?.AvatarID ?? ""),
    actorType: resolveActorType(raw?.ActorType ?? raw?.actorType),
    type,
    createdAt: String(a?.CreatedAt ?? ""),
    updatedAt: String(a?.UpdatedAt ?? ""),
  };
}

export function fromApiComment(a: ApiComment): Comment {
  const authorType = resolveCommentAuthorType(a);

  const raw = a as ApiComment & {
    IsOwnerComment?: boolean;
    isOwnerComment?: boolean;
  };

  return {
    commentId: String(a?.CommentID ?? ""),
    tokenBlueprintId: String(a?.TokenBlueprintID ?? ""),
    parentCommentId: String(a?.ParentCommentID ?? ""),
    rootCommentId: String(a?.RootCommentID ?? ""),
    depth: Number(a?.Depth ?? 0),

    authorId: String(a?.AuthorID ?? ""),
    authorType,
    isOwnerComment: Boolean(raw?.IsOwnerComment ?? raw?.isOwnerComment ?? false),

    body: String(a?.Body ?? ""),
    likeCount: Number(a?.LikeCount ?? 0),
    dislikeCount: Number(a?.DislikeCount ?? 0),
    childCount: Number(a?.ChildCount ?? 0),
    deleted: Boolean(a?.Deleted ?? false),

    createdAt: String(a?.CreatedAt ?? ""),
    updatedAt: String(a?.UpdatedAt ?? ""),

    authorAvatarName: toOptionalString(a?.AuthorAvatarName),
    authorAvatarIcon: toOptionalString(a?.AuthorAvatarIcon),

    brandName: toOptionalString(a?.BrandName),
    brandIcon: toOptionalString(a?.BrandIcon),
  };
}

export function fromApiCommentReaction(a: ApiCommentReaction): CommentReaction {
  const type = (a?.Type ?? ReactionComment) as ReactionType;
  validateReactionType(type);

  const raw = a as ApiCommentReaction & {
    ActorID?: string;
    ActorType?: string;
    actorId?: string;
    actorType?: string;
    AvatarID?: string;
  };

  return {
    tokenBlueprintId: String(a?.TokenBlueprintID ?? ""),
    commentId: String(a?.CommentID ?? ""),
    actorId: String(raw?.ActorID ?? raw?.actorId ?? raw?.AvatarID ?? ""),
    actorType: resolveActorType(raw?.ActorType ?? raw?.actorType),
    type,
    createdAt: String(a?.CreatedAt ?? ""),
    updatedAt: String(a?.UpdatedAt ?? ""),
  };
}