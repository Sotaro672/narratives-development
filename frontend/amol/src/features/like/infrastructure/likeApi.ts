// frontend/amol/src/features/like/infrastructure/likeApi.ts

import { requestJson } from "../../../lib/http";
import type {
  FetchLikesParams,
  LikePage,
  LikeStatus,
  LikeTargetType,
} from "../../shared/types/like";

import {
  DEFAULT_PAGE,
  DEFAULT_PER_PAGE,
} from "../constants";

const LIKE_BASE_PATH = "/mall/me/likes";
const MAX_PER_PAGE = 100;

function normalizePage(value?: number): number {
  if (!Number.isFinite(value) || value == null) {
    return DEFAULT_PAGE;
  }

  return Math.max(DEFAULT_PAGE, Math.trunc(value));
}

function normalizePerPage(value?: number): number {
  if (!Number.isFinite(value) || value == null) {
    return DEFAULT_PER_PAGE;
  }

  return Math.min(MAX_PER_PAGE, Math.max(1, Math.trunc(value)));
}

function requireTargetId(targetId: string): string {
  const normalizedTargetId = targetId.trim();

  if (!normalizedTargetId) {
    throw new Error("お気に入り対象IDが未指定です。");
  }

  return normalizedTargetId;
}

function buildLikeTargetPath(
  targetType: LikeTargetType,
  targetId: string,
): string {
  const normalizedTargetId = requireTargetId(targetId);

  return `${LIKE_BASE_PATH}/${targetType}/${encodeURIComponent(normalizedTargetId)}`;
}

/**
 * 現在のアバターのお気に入り一覧を取得します。
 *
 * GET /mall/me/likes
 *
 * targetType:
 * - undefined = list / resale 両方
 * - list      = 通常販売商品のみ
 * - resale    = 二次流通商品のみ
 */
export async function fetchLikes(
  params: FetchLikesParams = {},
): Promise<LikePage> {
  const page = normalizePage(params.page);
  const perPage = normalizePerPage(params.perPage);

  return requestJson<LikePage>(LIKE_BASE_PATH, {
    method: "GET",
    auth: "required",
    credentials: "include",
    query: {
      targetType: params.targetType,
      page,
      perPage,
    },
    messages: {
      requestErrorMessage: "お気に入り一覧の取得に失敗しました。",
      nonJsonErrorMessage: "お気に入り一覧APIがJSON以外を返しました。",
      invalidJsonErrorMessage: "お気に入り一覧APIのJSON形式が不正です。",
    },
  });
}

/**
 * 指定した対象を現在のアバターがお気に入り登録しているか取得します。
 *
 * GET /mall/me/likes/{targetType}/{targetId}
 */
export async function fetchLikeStatus(
  targetType: LikeTargetType,
  targetId: string,
): Promise<LikeStatus> {
  return requestJson<LikeStatus>(
    buildLikeTargetPath(targetType, targetId),
    {
      method: "GET",
      auth: "required",
      credentials: "include",
      unwrapData: true,
      messages: {
        requestErrorMessage: "お気に入り状態の取得に失敗しました。",
        nonJsonErrorMessage: "お気に入り状態APIがJSON以外を返しました。",
        invalidJsonErrorMessage: "お気に入り状態APIのJSON形式が不正です。",
      },
    },
  );
}

/**
 * 指定した対象を現在のアバターのお気に入りへ追加します。
 *
 * PUT /mall/me/likes/{targetType}/{targetId}
 *
 * backend側で冪等処理されるため、既にお気に入り済みでも liked=true を返します。
 */
export async function addLike(
  targetType: LikeTargetType,
  targetId: string,
): Promise<LikeStatus> {
  return requestJson<LikeStatus>(
    buildLikeTargetPath(targetType, targetId),
    {
      method: "PUT",
      auth: "required",
      credentials: "include",
      unwrapData: true,
      messages: {
        requestErrorMessage: "お気に入りの登録に失敗しました。",
        nonJsonErrorMessage: "お気に入り登録APIがJSON以外を返しました。",
        invalidJsonErrorMessage: "お気に入り登録APIのJSON形式が不正です。",
      },
    },
  );
}

/**
 * 指定した対象を現在のアバターのお気に入りから削除します。
 *
 * DELETE /mall/me/likes/{targetType}/{targetId}
 *
 * backend側で冪等処理されるため、既に削除済みでも liked=false を返します。
 */
export async function removeLike(
  targetType: LikeTargetType,
  targetId: string,
): Promise<LikeStatus> {
  return requestJson<LikeStatus>(
    buildLikeTargetPath(targetType, targetId),
    {
      method: "DELETE",
      auth: "required",
      credentials: "include",
      unwrapData: true,
      messages: {
        requestErrorMessage: "お気に入りの解除に失敗しました。",
        nonJsonErrorMessage: "お気に入り解除APIがJSON以外を返しました。",
        invalidJsonErrorMessage: "お気に入り解除APIのJSON形式が不正です。",
      },
    },
  );
}

// ============================================================
// List
// ============================================================

export async function fetchListLikeStatus(
  listId: string,
): Promise<LikeStatus> {
  return fetchLikeStatus("list", listId);
}

export async function addListLike(
  listId: string,
): Promise<LikeStatus> {
  return addLike("list", listId);
}

export async function removeListLike(
  listId: string,
): Promise<LikeStatus> {
  return removeLike("list", listId);
}

// ============================================================
// Resale
// ============================================================

export async function fetchResaleLikeStatus(
  resaleId: string,
): Promise<LikeStatus> {
  return fetchLikeStatus("resale", resaleId);
}

export async function addResaleLike(
  resaleId: string,
): Promise<LikeStatus> {
  return addLike("resale", resaleId);
}

export async function removeResaleLike(
  resaleId: string,
): Promise<LikeStatus> {
  return removeLike("resale", resaleId);
}