// frontend/amol/src/features/resale/api/resaleReviewApi.ts

import {
  fetchResaleWithAuth,
  type ApiDataResponse,
} from "./resaleHttpClient";

import type {
  ResaleInteractionSummary,
  ResaleReviewComment,
  ResaleReviewCommentPage,
} from "../../shared/types/resaleReview";

export type FetchMyResaleCommentsParams = {
  resaleId: string;
  page?: number;
  perPage?: number;
};

export type CreateMyResaleCommentParams = {
  resaleId: string;
  body: string;
};

export type DeleteMyResaleCommentParams = {
  resaleId: string;
  commentId: string;
};

export type CreateMyResaleCommentResult = {
  comment: ResaleReviewComment;
  interaction: ResaleInteractionSummary;
};

type CreateMyResaleCommentResponse = {
  data: ResaleReviewComment;
  interaction: ResaleInteractionSummary;
};

function requireResaleId(resaleId: string): string {
  const normalizedResaleId = resaleId.trim();

  if (!normalizedResaleId) {
    throw new Error("resaleId is required");
  }

  return normalizedResaleId;
}

function requireCommentId(commentId: string): string {
  const normalizedCommentId = commentId.trim();

  if (!normalizedCommentId) {
    throw new Error("commentId is required");
  }

  return normalizedCommentId;
}

export async function fetchMyResaleComments(
  params: FetchMyResaleCommentsParams,
): Promise<ResaleReviewCommentPage> {
  const resaleId = requireResaleId(params.resaleId);
  const page = Math.max(1, Math.trunc(params.page ?? 1));
  const perPage = Math.min(
    200,
    Math.max(1, Math.trunc(params.perPage ?? 20)),
  );

  return fetchResaleWithAuth<ResaleReviewCommentPage>(
    `/mall/me/resales/${encodeURIComponent(resaleId)}/comments`,
    {
      method: "GET",
      query: {
        page,
        perPage,
      },
    },
  );
}

export async function createMyResaleComment(
  params: CreateMyResaleCommentParams,
): Promise<CreateMyResaleCommentResult> {
  const resaleId = requireResaleId(params.resaleId);

  if (!params.body || !/\S/u.test(params.body)) {
    throw new Error("コメントを入力してください。");
  }

  const result = await fetchResaleWithAuth<CreateMyResaleCommentResponse>(
    `/mall/me/resales/${encodeURIComponent(resaleId)}/comments`,
    {
      method: "POST",
      json: {
        body: params.body,
      },
    },
  );

  return {
    comment: result.data,
    interaction: result.interaction,
  };
}

export async function deleteMyResaleComment(
  params: DeleteMyResaleCommentParams,
): Promise<ResaleInteractionSummary> {
  const resaleId = requireResaleId(params.resaleId);
  const commentId = requireCommentId(params.commentId);

  const result = await fetchResaleWithAuth<
    ApiDataResponse<ResaleInteractionSummary>
  >(
    `/mall/me/resales/${encodeURIComponent(resaleId)}/comments/${encodeURIComponent(commentId)}`,
    {
      method: "DELETE",
    },
  );

  return result.data;
}