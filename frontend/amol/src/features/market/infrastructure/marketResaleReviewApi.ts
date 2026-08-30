// frontend/amol/src/features/market/infrastructure/marketResaleReviewApi.ts

import { requestJson } from "../../../lib/http";
import { MARKET_RESALES_PATH } from "../constants/marketPaths";

import type {
  ResaleInteractionSummary,
  ResaleReviewComment,
  ResaleReviewCommentPage,
} from "../../shared/types/resaleReview";

type ApiDataResponse<T> = {
  data: T;
};

type CreateMarketResaleCommentResponse = {
  data: ResaleReviewComment;
  interaction: ResaleInteractionSummary;
};

export type CreateMarketResaleCommentResult = {
  comment: ResaleReviewComment;
  interaction: ResaleInteractionSummary;
};

export type FetchMarketResaleCommentsParams = {
  resaleId: string;
  page?: number;
  perPage?: number;
};

export type CreateMarketResaleCommentParams = {
  resaleId: string;
  body: string;
};

export type DeleteMarketResaleCommentParams = {
  resaleId: string;
  commentId: string;
};

function requireResaleId(resaleId: string): string {
  const normalizedResaleId = resaleId.trim();

  if (!normalizedResaleId) {
    throw new Error("マーケット出品IDが未指定です。");
  }

  return normalizedResaleId;
}

function requireCommentId(commentId: string): string {
  const normalizedCommentId = commentId.trim();

  if (!normalizedCommentId) {
    throw new Error("コメントIDが未指定です。");
  }

  return normalizedCommentId;
}

function buildMarketResaleReviewPath(
  resaleId: string,
  suffix: "interactions" | "like" | "comments",
): string {
  const normalizedResaleId = requireResaleId(resaleId);

  return `${MARKET_RESALES_PATH}/${encodeURIComponent(normalizedResaleId)}/${suffix}`;
}

export async function fetchMarketResaleInteractions(
  resaleId: string,
): Promise<ResaleInteractionSummary> {
  const result = await requestJson<ApiDataResponse<ResaleInteractionSummary>>(
    buildMarketResaleReviewPath(resaleId, "interactions"),
    {
      method: "GET",
      auth: "required",
      credentials: "include",
      messages: {
        requestErrorMessage: "いいね情報の取得に失敗しました。",
        nonJsonErrorMessage: "いいね情報APIがJSON以外を返しました。",
        invalidJsonErrorMessage: "いいね情報APIのレスポンスが不正です。",
      },
    },
  );

  return result.data;
}

export async function addMarketResaleLike(
  resaleId: string,
): Promise<ResaleInteractionSummary> {
  const result = await requestJson<ApiDataResponse<ResaleInteractionSummary>>(
    buildMarketResaleReviewPath(resaleId, "like"),
    {
      method: "PUT",
      auth: "required",
      credentials: "include",
      messages: {
        requestErrorMessage: "いいねの登録に失敗しました。",
        nonJsonErrorMessage: "いいね登録APIがJSON以外を返しました。",
        invalidJsonErrorMessage: "いいね登録APIのレスポンスが不正です。",
      },
    },
  );

  return result.data;
}

export async function removeMarketResaleLike(
  resaleId: string,
): Promise<ResaleInteractionSummary> {
  const result = await requestJson<ApiDataResponse<ResaleInteractionSummary>>(
    buildMarketResaleReviewPath(resaleId, "like"),
    {
      method: "DELETE",
      auth: "required",
      credentials: "include",
      messages: {
        requestErrorMessage: "いいねの解除に失敗しました。",
        nonJsonErrorMessage: "いいね解除APIがJSON以外を返しました。",
        invalidJsonErrorMessage: "いいね解除APIのレスポンスが不正です。",
      },
    },
  );

  return result.data;
}

export async function fetchMarketResaleComments(
  params: FetchMarketResaleCommentsParams,
): Promise<ResaleReviewCommentPage> {
  const page = Math.max(
    1,
    Math.trunc(params.page ?? 1),
  );
  const perPage = Math.min(
    100,
    Math.max(
      1,
      Math.trunc(params.perPage ?? 20),
    ),
  );

  return requestJson<ResaleReviewCommentPage>(
    buildMarketResaleReviewPath(params.resaleId, "comments"),
    {
      method: "GET",
      auth: "required",
      credentials: "include",
      query: {
        page,
        perPage,
      },
      messages: {
        requestErrorMessage: "コメントの取得に失敗しました。",
        nonJsonErrorMessage: "コメント一覧APIがJSON以外を返しました。",
        invalidJsonErrorMessage: "コメント一覧APIのレスポンスが不正です。",
      },
    },
  );
}

export async function createMarketResaleComment(
  params: CreateMarketResaleCommentParams,
): Promise<CreateMarketResaleCommentResult> {
  const resaleId = requireResaleId(params.resaleId);

  if (!params.body || !/\S/u.test(params.body)) {
    throw new Error("コメントを入力してください。");
  }

  const result = await requestJson<CreateMarketResaleCommentResponse>(
    `${MARKET_RESALES_PATH}/${encodeURIComponent(resaleId)}/comments`,
    {
      method: "POST",
      auth: "required",
      credentials: "include",
      json: {
        body: params.body,
      },
      messages: {
        requestErrorMessage: "コメントの投稿に失敗しました。",
        nonJsonErrorMessage: "コメント投稿APIがJSON以外を返しました。",
        invalidJsonErrorMessage: "コメント投稿APIのレスポンスが不正です。",
      },
    },
  );

  return {
    comment: result.data,
    interaction: result.interaction,
  };
}

export async function deleteMarketResaleComment(
  params: DeleteMarketResaleCommentParams,
): Promise<ResaleInteractionSummary> {
  const resaleId = requireResaleId(params.resaleId);
  const commentId = requireCommentId(params.commentId);

  const result = await requestJson<ApiDataResponse<ResaleInteractionSummary>>(
    `${MARKET_RESALES_PATH}/${encodeURIComponent(resaleId)}/comments/${encodeURIComponent(commentId)}`,
    {
      method: "DELETE",
      auth: "required",
      credentials: "include",
      messages: {
        requestErrorMessage: "コメントの削除に失敗しました。",
        nonJsonErrorMessage: "コメント削除APIがJSON以外を返しました。",
        invalidJsonErrorMessage: "コメント削除APIのレスポンスが不正です。",
      },
    },
  );

  return result.data;
}