// frontend/amol/src/features/token-commnet/api/tokenCommentApi.ts

import {
  requestJson,
  requestVoid,
} from "../../../lib/http";
import {
  textOrEmpty,
} from "../../../components/utils/textOrEmpty";
import {
  isFiniteNumber,
  isRecord,
} from "../../../components/utils/typeGuards";

import type {
  TokenBlueprintReactionInput,
  TokenBlueprintReviewAggregate,
  TokenComment,
  TokenCommentListResponse,
  TokenCommentPostInput,
  TokenCommentReplyInput,
  TokenCommentVoteInput,
} from "../../shared/types/tokenCommentTypes";

const TOKEN_BLUEPRINT_BASE_PATH =
  "/mall/me/token-blueprints";

function asString(
  value: unknown,
): string {
  return typeof value === "string"
    ? value
    : "";
}

function asNumber(
  value: unknown,
): number {
  return isFiniteNumber(value)
    ? value
    : 0;
}

function asBoolean(
  value: unknown,
): boolean {
  return value === true;
}

function parseTokenBlueprintReviewAggregate(
  value: unknown,
  fallbackTokenBlueprintId = "",
): TokenBlueprintReviewAggregate {
  if (!isRecord(value)) {
    return {
      tokenBlueprintId:
        fallbackTokenBlueprintId,
      likeCount: 0,
      dislikeCount: 0,
      topLevelCommentCount: 0,
      totalCommentCount: 0,
      createdAt: null,
      updatedAt: null,
    };
  }

  return {
    tokenBlueprintId:
      asString(
        value.TokenBlueprintID,
      ) ||
      fallbackTokenBlueprintId,

    likeCount:
      asNumber(
        value.LikeCount,
      ),

    dislikeCount:
      asNumber(
        value.DislikeCount,
      ),

    topLevelCommentCount:
      asNumber(
        value.TopLevelCommentCount,
      ),

    totalCommentCount:
      asNumber(
        value.TotalCommentCount,
      ),

    createdAt:
      textOrEmpty(
        value.CreatedAt,
      ),

    updatedAt:
      textOrEmpty(
        value.UpdatedAt,
      ),
  };
}

function parseTokenComment(
  value: unknown,
): TokenComment | null {
  if (!isRecord(value)) {
    return null;
  }

  return {
    commentId:
      asString(
        value.CommentID,
      ),

    tokenBlueprintId:
      asString(
        value.TokenBlueprintID,
      ),

    parentCommentId:
      asString(
        value.ParentCommentID,
      ),

    rootCommentId:
      asString(
        value.RootCommentID,
      ),

    depth:
      asNumber(
        value.Depth,
      ),

    authorId:
      asString(
        value.AuthorID,
      ),

    authorType:
      asString(
        value.AuthorType,
      ),

    isOwnerComment:
      asBoolean(
        value.IsOwnerComment,
      ),

    body:
      asString(
        value.Body,
      ),

    likeCount:
      asNumber(
        value.LikeCount,
      ),

    dislikeCount:
      asNumber(
        value.DislikeCount,
      ),

    childCount:
      asNumber(
        value.ChildCount,
      ),

    deleted:
      asBoolean(
        value.Deleted,
      ),

    createdAt:
      asString(
        value.CreatedAt,
      ),

    updatedAt:
      asString(
        value.UpdatedAt,
      ),

    authorAvatarName:
      textOrEmpty(
        value.AuthorAvatarName,
      ),

    authorAvatarIcon:
      textOrEmpty(
        value.AuthorAvatarIcon,
      ),

    brandName:
      textOrEmpty(
        value.BrandName,
      ),

    brandIcon:
      textOrEmpty(
        value.BrandIcon,
      ),
  };
}

function parseTokenCommentListResponse(
  value: unknown,
): TokenCommentListResponse {
  if (!isRecord(value)) {
    return {
      items: [],
      page: 1,
      perPage: 20,
      total: 0,
    };
  }

  const items =
    Array.isArray(
      value.items,
    )
      ? value.items
          .map(
            parseTokenComment,
          )
          .filter(
            (
              comment,
            ): comment is TokenComment =>
              comment !== null,
          )
      : [];

  return {
    items,

    page:
      asNumber(
        value.page,
      ) || 1,

    perPage:
      asNumber(
        value.perPage,
      ) || 20,

    total:
      asNumber(
        value.totalCount,
      ),
  };
}

function encodePathSegment(
  value: string,
): string {
  return encodeURIComponent(
    value,
  );
}

export async function fetchTokenBlueprintReviewAggregate(
  tokenBlueprintId: string,
): Promise<TokenBlueprintReviewAggregate> {
  if (!tokenBlueprintId) {
    return {
      tokenBlueprintId: "",
      likeCount: 0,
      dislikeCount: 0,
      topLevelCommentCount: 0,
      totalCommentCount: 0,
      createdAt: null,
      updatedAt: null,
    };
  }

  const body =
    await requestJson<unknown>(
      `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(
        tokenBlueprintId,
      )}/reviews/aggregate`,
      {
        method: "GET",
        auth: "required",
        messages: {
          requestErrorMessage:
            "token comment API failed.",
          nonJsonErrorMessage:
            "token comment API が JSON 以外を返しました。",
        },
      },
    );

  return parseTokenBlueprintReviewAggregate(
    body,
    tokenBlueprintId,
  );
}

export async function upsertTokenBlueprintReaction({
  tokenBlueprintId,
  type,
}: TokenBlueprintReactionInput): Promise<void> {
  if (
    !tokenBlueprintId ||
    !type
  ) {
    return;
  }

  await requestVoid(
    `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(
      tokenBlueprintId,
    )}/reactions`,
    {
      method: "POST",
      auth: "required",
      json: {
        type,
      },
    },
  );
}

export async function fetchTokenComments(
  tokenBlueprintId: string,
): Promise<TokenCommentListResponse> {
  if (!tokenBlueprintId) {
    return {
      items: [],
      page: 1,
      perPage: 20,
      total: 0,
    };
  }

  const body =
    await requestJson<unknown>(
      `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(
        tokenBlueprintId,
      )}/comments`,
      {
        method: "GET",
        auth: "required",
        messages: {
          requestErrorMessage:
            "token comment API failed.",
          nonJsonErrorMessage:
            "token comment API が JSON 以外を返しました。",
        },
      },
    );

  return parseTokenCommentListResponse(
    body,
  );
}

export async function postTokenComment({
  tokenBlueprintId,
  body,
}: TokenCommentPostInput): Promise<void> {
  const trimmedBody =
    body.trim();

  if (
    !tokenBlueprintId ||
    !trimmedBody
  ) {
    return;
  }

  await requestVoid(
    `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(
      tokenBlueprintId,
    )}/comments`,
    {
      method: "POST",
      auth: "required",
      json: {
        body: trimmedBody,
      },
    },
  );
}

export async function postTokenCommentReply({
  tokenBlueprintId,
  parentCommentId,
  body,
}: TokenCommentReplyInput): Promise<void> {
  const trimmedBody =
    body.trim();

  if (
    !tokenBlueprintId ||
    !parentCommentId ||
    !trimmedBody
  ) {
    return;
  }

  await requestVoid(
    `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(
      tokenBlueprintId,
    )}/comments/${encodePathSegment(
      parentCommentId,
    )}/replies`,
    {
      method: "POST",
      auth: "required",
      json: {
        body: trimmedBody,
      },
    },
  );
}

export async function likeTokenComment({
  tokenBlueprintId,
  commentId,
}: TokenCommentVoteInput): Promise<void> {
  if (
    !tokenBlueprintId ||
    !commentId
  ) {
    return;
  }

  await requestVoid(
    `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(
      tokenBlueprintId,
    )}/comments/${encodePathSegment(
      commentId,
    )}/reactions`,
    {
      method: "POST",
      auth: "required",
      json: {
        type: "like",
      },
    },
  );
}

export async function dislikeTokenComment({
  tokenBlueprintId,
  commentId,
}: TokenCommentVoteInput): Promise<void> {
  if (
    !tokenBlueprintId ||
    !commentId
  ) {
    return;
  }

  await requestVoid(
    `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(
      tokenBlueprintId,
    )}/comments/${encodePathSegment(
      commentId,
    )}/reactions`,
    {
      method: "POST",
      auth: "required",
      json: {
        type: "dislike",
      },
    },
  );
}