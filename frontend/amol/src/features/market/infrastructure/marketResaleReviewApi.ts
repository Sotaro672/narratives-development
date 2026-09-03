// frontend/amol/src/features/market/infrastructure/marketResaleReviewApi.ts

import { requestJson } from "../../../lib/http";
import { MARKET_RESALES_PATH } from "../constants/marketPaths";

import type {
  ResaleReviewComment,
  ResaleReviewCommentPage,
} from "../../shared/types/resaleReview";

type CreateMarketResaleCommentResponse = {
  data: ResaleReviewComment;
};

export type CreateMarketResaleCommentResult = {
  comment: ResaleReviewComment;
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

function buildMarketResaleCommentsPath(resaleId: string): string {
  const normalizedResaleId = requireResaleId(resaleId);

  return `${MARKET_RESALES_PATH}/${encodeURIComponent(normalizedResaleId)}/comments`;
}

export async function fetchMarketResaleComments(
  params: FetchMarketResaleCommentsParams,
): Promise<ResaleReviewCommentPage> {
  const page = Math.max(1, Math.trunc(params.page ?? 1));
  const perPage = Math.min(100, Math.max(1, Math.trunc(params.perPage ?? 20)));

  return requestJson<ResaleReviewCommentPage>(
    buildMarketResaleCommentsPath(params.resaleId),
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
    buildMarketResaleCommentsPath(resaleId),
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
  };
}

export async function deleteMarketResaleComment(
  params: DeleteMarketResaleCommentParams,
): Promise<void> {
  const resaleId = requireResaleId(params.resaleId);
  const commentId = requireCommentId(params.commentId);

  await requestJson<unknown>(
    `${buildMarketResaleCommentsPath(resaleId)}/${encodeURIComponent(commentId)}`,
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
}