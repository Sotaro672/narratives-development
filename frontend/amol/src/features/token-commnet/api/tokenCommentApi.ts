// frontend/amol/src/features/token-commnet/api/tokenCommentApi.ts

import {
  requestJson,
  requestVoid,
} from "../../../lib/http";

import type {
  TokenBlueprintReactionInput,
  TokenBlueprintReviewAggregate,
  TokenComment,
  TokenCommentListResponse,
  TokenCommentPostInput,
  TokenCommentReplyInput,
  TokenCommentVoteInput,
} from "../../shared/types/tokenCommentTypes";

const TOKEN_BLUEPRINT_BASE_PATH = "/mall/me/token-blueprints";

type TokenBlueprintReviewAggregateResponse = {
  TokenBlueprintID: string;
  LikeCount: number;
  DislikeCount: number;
  TopLevelCommentCount: number;
  TotalCommentCount: number;
  CreatedAt: string;
  UpdatedAt: string;
};

type TokenCommentResponse = {
  CommentID: string;
  TokenBlueprintID: string;
  ParentCommentID: string;
  RootCommentID: string;
  Depth: number;
  AuthorID: string;
  AuthorType: string;
  IsOwnerComment: boolean;
  Body: string;
  LikeCount: number;
  DislikeCount: number;
  ChildCount: number;
  Deleted: boolean;
  CreatedAt: string;
  UpdatedAt: string;
  AuthorAvatarName: string;
  AuthorAvatarIcon: string | null;
  BrandName: string;
  BrandIcon: string | null;
};

type TokenCommentListResponseBody = {
  items: TokenCommentResponse[];
  page: number;
  perPage: number;
  totalCount: number;
};

function toTokenBlueprintReviewAggregate(
  value: TokenBlueprintReviewAggregateResponse,
): TokenBlueprintReviewAggregate {
  return {
    tokenBlueprintId: value.TokenBlueprintID,
    likeCount: value.LikeCount,
    dislikeCount: value.DislikeCount,
    topLevelCommentCount: value.TopLevelCommentCount,
    totalCommentCount: value.TotalCommentCount,
    createdAt: value.CreatedAt,
    updatedAt: value.UpdatedAt,
  };
}

function toTokenComment(
  value: TokenCommentResponse,
): TokenComment {
  return {
    commentId: value.CommentID,
    tokenBlueprintId: value.TokenBlueprintID,
    parentCommentId: value.ParentCommentID,
    rootCommentId: value.RootCommentID,
    depth: value.Depth,
    authorId: value.AuthorID,
    authorType: value.AuthorType,
    isOwnerComment: value.IsOwnerComment,
    body: value.Body,
    likeCount: value.LikeCount,
    dislikeCount: value.DislikeCount,
    childCount: value.ChildCount,
    deleted: value.Deleted,
    createdAt: value.CreatedAt,
    updatedAt: value.UpdatedAt,
    authorAvatarName: value.AuthorAvatarName,
    authorAvatarIcon: value.AuthorAvatarIcon,
    brandName: value.BrandName,
    brandIcon: value.BrandIcon,
  };
}

function toTokenCommentListResponse(
  value: TokenCommentListResponseBody,
): TokenCommentListResponse {
  return {
    items: value.items.map(toTokenComment),
    page: value.page,
    perPage: value.perPage,
    total: value.totalCount,
  };
}

function encodePathSegment(value: string): string {
  return encodeURIComponent(value);
}

export async function fetchTokenBlueprintReviewAggregate(
  tokenBlueprintId: string,
): Promise<TokenBlueprintReviewAggregate> {
  const body = await requestJson<TokenBlueprintReviewAggregateResponse>(
    `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(tokenBlueprintId)}/reviews/aggregate`,
    {
      method: "GET",
      auth: "required",
      messages: {
        requestErrorMessage: "token comment API failed.",
        nonJsonErrorMessage: "token comment API が JSON 以外を返しました。",
      },
    },
  );

  return toTokenBlueprintReviewAggregate(body);
}

export async function upsertTokenBlueprintReaction({
  tokenBlueprintId,
  type,
}: TokenBlueprintReactionInput): Promise<void> {
  await requestVoid(
    `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(tokenBlueprintId)}/reactions`,
    {
      method: "POST",
      auth: "required",
      json: { type },
    },
  );
}

export async function fetchTokenComments(
  tokenBlueprintId: string,
): Promise<TokenCommentListResponse> {
  const body = await requestJson<TokenCommentListResponseBody>(
    `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(tokenBlueprintId)}/comments`,
    {
      method: "GET",
      auth: "required",
      messages: {
        requestErrorMessage: "token comment API failed.",
        nonJsonErrorMessage: "token comment API が JSON 以外を返しました。",
      },
    },
  );

  return toTokenCommentListResponse(body);
}

export async function postTokenComment({
  tokenBlueprintId,
  body,
}: TokenCommentPostInput): Promise<void> {
  await requestVoid(
    `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(tokenBlueprintId)}/comments`,
    {
      method: "POST",
      auth: "required",
      json: { body },
    },
  );
}

export async function postTokenCommentReply({
  tokenBlueprintId,
  parentCommentId,
  body,
}: TokenCommentReplyInput): Promise<void> {
  await requestVoid(
    `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(tokenBlueprintId)}/comments/${encodePathSegment(parentCommentId)}/replies`,
    {
      method: "POST",
      auth: "required",
      json: { body },
    },
  );
}

export async function likeTokenComment({
  tokenBlueprintId,
  commentId,
}: TokenCommentVoteInput): Promise<void> {
  await requestVoid(
    `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(tokenBlueprintId)}/comments/${encodePathSegment(commentId)}/reactions`,
    {
      method: "POST",
      auth: "required",
      json: { type: "like" },
    },
  );
}

export async function dislikeTokenComment({
  tokenBlueprintId,
  commentId,
}: TokenCommentVoteInput): Promise<void> {
  await requestVoid(
    `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(tokenBlueprintId)}/comments/${encodePathSegment(commentId)}/reactions`,
    {
      method: "POST",
      auth: "required",
      json: { type: "dislike" },
    },
  );
}