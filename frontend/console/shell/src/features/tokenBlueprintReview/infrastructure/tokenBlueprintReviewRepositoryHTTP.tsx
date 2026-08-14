// frontend/console/shell/src/features/tokenBlueprintReview/infrastructure/tokenBlueprintReviewRepositoryHTTP.tsx

import type {
  TokenBlueprintReviewAggregate,
  Comment,
  ReactionType,
} from "../../../shared/types/tokenBlueprintReview";

import { API_BASE } from "../../../shared/http/apiBase";
import { getAuthJsonHeaders } from "../../../shared/http/authHeaders";

/**
 * console(brand) 用 TokenBlueprintReview HTTP repository
 *
 * backend BFF の response を正とし、
 * frontend 側では mapper / normalizer / alias を持たない。
 */

async function apiGetJson<T>(path: string): Promise<T> {
  const headers = await getAuthJsonHeaders();
  const response = await fetch(`${API_BASE}${path}`, {
    method: "GET",
    headers: { ...headers, Accept: "application/json" },
    credentials: "include",
  });

  const text = await response.text();
  if (!response.ok) {
    throw new Error(text || `GET ${path} failed: ${response.status}`);
  }
  if (!text) {
    throw new Error(`GET ${path} returned an empty response`);
  }

  return JSON.parse(text) as T;
}

async function apiPostJson<T>(path: string, body: unknown): Promise<T> {
  const headers = await getAuthJsonHeaders();
  const response = await fetch(`${API_BASE}${path}`, {
    method: "POST",
    headers: {
      ...headers,
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
    credentials: "include",
  });

  const text = await response.text();
  if (!response.ok) {
    throw new Error(text || `POST ${path} failed: ${response.status}`);
  }
  if (!text) {
    throw new Error(`POST ${path} returned an empty response`);
  }

  return JSON.parse(text) as T;
}

async function apiDelete(path: string): Promise<void> {
  const headers = await getAuthJsonHeaders();
  const response = await fetch(`${API_BASE}${path}`, {
    method: "DELETE",
    headers: { ...headers, Accept: "application/json" },
    credentials: "include",
  });

  if (response.ok) {
    return;
  }

  const text = await response.text();
  throw new Error(text || `DELETE ${path} failed: ${response.status}`);
}

// ============================================================
// Response DTOs
// ============================================================

type ListTokenBlueprintReviewAggregatesResponse = {
  items: TokenBlueprintReviewAggregate[];
};

export type ListTokenBlueprintCommentsResponse = {
  items: Comment[];
  tokenBlueprintName: string;
  brandName: string;
  page?: number;
  perPage?: number;
  totalCount?: number;
};

type CommentResponse = {
  item: Comment;
};

// ============================================================
// Request DTOs
// ============================================================

type CreateBrandCommentRequest = {
  commentId?: string;
  parentCommentId?: string;
  body: string;
};

type ReactAsBrandRequest = {
  type: ReactionType;
};

// ============================================================
// Aggregates
// ============================================================

/**
 * backend: GET /token-blueprint-reviews
 *
 * companyId は backend の認証 context から解決する。
 */
export async function listTokenBlueprintReviewAggregates(): Promise<
  TokenBlueprintReviewAggregate[]
> {
  const response =
    await apiGetJson<ListTokenBlueprintReviewAggregatesResponse>(
      "/token-blueprint-reviews",
    );

  return response.items;
}

// ============================================================
// Comments
// ============================================================

/**
 * backend: GET /token-blueprint-reviews/{tokenBlueprintId}/comments
 *
 * detail 表示用に top-level comment と replies を含む
 * comments 全件を取得する。
 */
export async function listTokenBlueprintCommentsByTokenBlueprintId(
  tokenBlueprintId: string,
): Promise<ListTokenBlueprintCommentsResponse> {
  if (!tokenBlueprintId) {
    throw new Error("tokenBlueprintId is required");
  }

  return apiGetJson<ListTokenBlueprintCommentsResponse>(
    `/token-blueprint-reviews/${encodeURIComponent(tokenBlueprintId)}/comments`,
  );
}

/**
 * backend: POST /token-blueprint-reviews/{tokenBlueprintId}/comments
 */
export async function createBrandComment(
  tokenBlueprintId: string,
  body: string,
  options?: {
    commentId?: string;
    parentCommentId?: string;
  },
): Promise<Comment> {
  if (!tokenBlueprintId) {
    throw new Error("tokenBlueprintId is required");
  }
  if (!body) {
    throw new Error("body is required");
  }

  const request: CreateBrandCommentRequest = {
    body,
    ...(options?.commentId ? { commentId: options.commentId } : {}),
    ...(options?.parentCommentId
      ? { parentCommentId: options.parentCommentId }
      : {}),
  };

  const response = await apiPostJson<CommentResponse>(
    `/token-blueprint-reviews/${encodeURIComponent(tokenBlueprintId)}/comments`,
    request,
  );

  return response.item;
}

/**
 * backend: POST /token-blueprint-reviews/{tokenBlueprintId}/comments/{commentId}/replies
 */
export async function createBrandReply(
  tokenBlueprintId: string,
  parentCommentId: string,
  body: string,
  options?: {
    commentId?: string;
  },
): Promise<Comment> {
  if (!tokenBlueprintId) {
    throw new Error("tokenBlueprintId is required");
  }
  if (!parentCommentId) {
    throw new Error("parentCommentId is required");
  }
  if (!body) {
    throw new Error("body is required");
  }

  const request: CreateBrandCommentRequest = {
    body,
    ...(options?.commentId ? { commentId: options.commentId } : {}),
  };

  const response = await apiPostJson<CommentResponse>(
    `/token-blueprint-reviews/${encodeURIComponent(tokenBlueprintId)}/comments/${encodeURIComponent(parentCommentId)}/replies`,
    request,
  );

  return response.item;
}

/**
 * backend: DELETE /token-blueprint-reviews/{tokenBlueprintId}/comments/{commentId}
 */
export async function deleteBrandComment(
  tokenBlueprintId: string,
  commentId: string,
): Promise<void> {
  if (!tokenBlueprintId) {
    throw new Error("tokenBlueprintId is required");
  }
  if (!commentId) {
    throw new Error("commentId is required");
  }

  await apiDelete(
    `/token-blueprint-reviews/${encodeURIComponent(tokenBlueprintId)}/comments/${encodeURIComponent(commentId)}`,
  );
}

// ============================================================
// Comment reactions
// ============================================================

/**
 * backend: POST /token-blueprint-reviews/{tokenBlueprintId}/comments/{commentId}/reactions
 */
export async function reactToCommentAsBrand(
  tokenBlueprintId: string,
  commentId: string,
  type: ReactionType,
): Promise<Comment> {
  if (!tokenBlueprintId) {
    throw new Error("tokenBlueprintId is required");
  }
  if (!commentId) {
    throw new Error("commentId is required");
  }

  const request: ReactAsBrandRequest = { type };
  const response = await apiPostJson<CommentResponse>(
    `/token-blueprint-reviews/${encodeURIComponent(tokenBlueprintId)}/comments/${encodeURIComponent(commentId)}/reactions`,
    request,
  );

  return response.item;
}