// frontend/console/shell/src/features/tokenBlueprintReview/application/tokenBlueprintReviewDetailService.tsx

import type { TokenBlueprint } from "../../../shared/types/tokenBlueprint";
import type {
  Comment,
  TokenBlueprintReviewAggregate,
  ReactionType,
} from "../../../shared/types/tokenBlueprintReview";
import type {
  ReportReason,
  ReportResponse,
} from "../../../shared/types/report";

import {
  listTokenBlueprintCommentsByTokenBlueprintId,
  listTokenBlueprintReviewAggregates,
  createBrandComment,
  createBrandReply,
  deleteBrandComment,
  reactToCommentAsBrand,
  reportTokenBlueprintCommentAsBrand,
} from "../infrastructure/tokenBlueprintReviewRepositoryHTTP";

import { fetchTokenBlueprintById } from "../../tokenBlueprint/infrastructure/repository/tokenBlueprintRepositoryHTTP";

/**
 * 詳細取得。
 * tokenBlueprintReviewId = tokenBlueprintId を前提とする。
 */
export async function fetchTokenBlueprintReviewDetail(
  id: string,
): Promise<TokenBlueprint> {
  const normalizedId = id.trim();

  if (!normalizedId) {
    throw new Error("id is required");
  }

  return fetchTokenBlueprintById(normalizedId);
}

/**
 * detail 用 comments 取得。
 * backend BFF は top-level comment と replies を含む comments 全件を返す。
 */
export async function fetchTokenBlueprintCommentsForDetail(
  tokenBlueprintId: string,
): Promise<{
  items: Comment[];
  tokenBlueprintName: string;
  brandName: string;
  page?: number;
  perPage?: number;
  totalCount?: number;
}> {
  const normalizedTokenBlueprintId = tokenBlueprintId.trim();

  if (!normalizedTokenBlueprintId) {
    throw new Error("tokenBlueprintId is required");
  }

  return listTokenBlueprintCommentsByTokenBlueprintId(
    normalizedTokenBlueprintId,
  );
}

/**
 * detail 用 aggregate 取得。
 * backend は companyId を認証 context から解決する。
 */
export async function fetchTokenBlueprintAggregateForDetail(
  tokenBlueprintId: string,
): Promise<TokenBlueprintReviewAggregate | null> {
  const normalizedTokenBlueprintId = tokenBlueprintId.trim();

  if (!normalizedTokenBlueprintId) {
    throw new Error("tokenBlueprintId is required");
  }

  const rows = await listTokenBlueprintReviewAggregates();

  return (
    rows.find(
      (row) =>
        row.tokenBlueprintId === normalizedTokenBlueprintId,
    ) ?? null
  );
}

/**
 * brand 側 top-level comment 作成。
 */
export async function postBrandComment(
  tokenBlueprintId: string,
  body: string,
  options?: {
    commentId?: string;
    parentCommentId?: string;
  },
): Promise<Comment> {
  return createBrandComment(
    tokenBlueprintId,
    body,
    options,
  );
}

/**
 * brand 側 reply 作成。
 */
export async function postBrandReply(
  tokenBlueprintId: string,
  parentCommentId: string,
  body: string,
  options?: {
    commentId?: string;
  },
): Promise<Comment> {
  return createBrandReply(
    tokenBlueprintId,
    parentCommentId,
    body,
    options,
  );
}

/**
 * brand 側 comment 削除。
 */
export async function removeBrandComment(
  tokenBlueprintId: string,
  commentId: string,
): Promise<void> {
  return deleteBrandComment(
    tokenBlueprintId,
    commentId,
  );
}

/**
 * brand 側 comment reaction。
 */
export async function reactBrandToComment(
  tokenBlueprintId: string,
  commentId: string,
  type: ReactionType,
): Promise<Comment> {
  return reactToCommentAsBrand(
    tokenBlueprintId,
    commentId,
    type,
  );
}

/**
 * brand 側 comment 通報。
 */
export async function reportBrandTokenBlueprintComment(
  tokenBlueprintId: string,
  commentId: string,
  reason: ReportReason,
  detail?: string,
): Promise<ReportResponse> {
  const normalizedTokenBlueprintId =
    tokenBlueprintId.trim();

  const normalizedCommentId =
    commentId.trim();

  const normalizedDetail =
    detail?.trim() ?? "";

  if (!normalizedTokenBlueprintId) {
    throw new Error(
      "tokenBlueprintId is required",
    );
  }

  if (!normalizedCommentId) {
    throw new Error(
      "commentId is required",
    );
  }

  return reportTokenBlueprintCommentAsBrand({
    tokenBlueprintId:
      normalizedTokenBlueprintId,
    commentId:
      normalizedCommentId,
    reason,
    ...(normalizedDetail
      ? { detail: normalizedDetail }
      : {}),
  });
}