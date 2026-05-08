// frontend/console/tokenBlueprintReview/src/application/tokenBlueprintReviewDetailService.tsx

import type { TokenBlueprint } from "../../../tokenBlueprint/src/domain/entity/tokenBlueprint";
import type {
  Comment,
  TokenBlueprintReviewAggregate,
  ReactionType,
} from "../domain/entity";

import {
  listTokenBlueprintCommentsByTokenBlueprintId,
  listTokenBlueprintReviewAggregatesByCompanyId,
  createBrandComment,
  createBrandReply,
  deleteBrandComment,
  reactToCommentAsBrand,
} from "../infrastructure/tokenBlueprintReviewRepositoryHTTP";

import { fetchTokenBlueprintById } from "../../../tokenBlueprint/src/infrastructure/repository/tokenBlueprintRepositoryHTTP";

/**
 * 詳細取得（リポジトリのラッパー）
 * - review 側では「tokenBlueprintReviewId = tokenBlueprintId（docId同一）」前提で取得する
 */
export async function fetchTokenBlueprintReviewDetail(id: string): Promise<TokenBlueprint> {
  if (!id) {
    throw new Error("id is empty");
  }
  return fetchTokenBlueprintById(id);
}

/**
 * detail 用 comments 取得
 *
 * NOTE:
 * - 親コメントと reply 表示のため、backend / repository 側では
 *   top-level のみではなく reply を含む comments 全件を返す必要がある。
 */
export async function fetchTokenBlueprintCommentsForDetail(
  tokenBlueprintId: string,
): Promise<{ items: Comment[]; tokenBlueprintName?: string; brandName?: string }> {
  if (!tokenBlueprintId) return { items: [] };

  return listTokenBlueprintCommentsByTokenBlueprintId(tokenBlueprintId);
}

/**
 * detail 用 aggregate（companyId から一覧取得して該当IDを抽出）
 * backend: GET /token-blueprint-reviews
 *
 * NOTE:
 * - backend は companyId を auth context で見ている前提だが、
 *   既存実装に合わせて companyId を「呼び出しトリガ」として受け取る。
 */
export async function fetchTokenBlueprintAggregateForDetail(
  companyId: string,
  tokenBlueprintId: string,
): Promise<TokenBlueprintReviewAggregate | null> {
  if (!companyId || !tokenBlueprintId) return null;

  const rows = await listTokenBlueprintReviewAggregatesByCompanyId(companyId);
  return rows.find((r) => r.tokenBlueprintId === tokenBlueprintId) ?? null;
}

/**
 * brand 側 top-level comment 作成
 */
export async function postBrandComment(
  tokenBlueprintId: string,
  body: string,
  options?: {
    commentId?: string;
    parentCommentId?: string;
  },
): Promise<Comment> {
  return createBrandComment(tokenBlueprintId, body, options);
}

/**
 * brand 側 reply 作成
 */
export async function postBrandReply(
  tokenBlueprintId: string,
  parentCommentId: string,
  body: string,
  options?: {
    commentId?: string;
  },
): Promise<Comment> {
  return createBrandReply(tokenBlueprintId, parentCommentId, body, options);
}

/**
 * brand 側 comment 削除
 */
export async function removeBrandComment(
  tokenBlueprintId: string,
  commentId: string,
): Promise<void> {
  return deleteBrandComment(tokenBlueprintId, commentId);
}

/**
 * brand 側 comment reaction
 */
export async function reactBrandToComment(
  tokenBlueprintId: string,
  commentId: string,
  type: ReactionType,
): Promise<Comment> {
  return reactToCommentAsBrand(tokenBlueprintId, commentId, type);
}