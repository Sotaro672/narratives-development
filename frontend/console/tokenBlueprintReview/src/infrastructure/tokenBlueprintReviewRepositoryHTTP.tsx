// frontend/console/tokenBlueprintReview/src/infrastructure/tokenBlueprintReviewRepositoryHTTP.tsx
import type {
  TokenBlueprintReviewAggregate,
  Comment,
  ReactionType,
} from "../domain/entity";

import type {
  ApiTokenBlueprintReviewAggregate,
  ApiComment,
} from "./apiTypes";

import {
  fromApiTokenBlueprintReviewAggregate,
  fromApiComment,
} from "./mappers";

import { API_BASE } from "../../../shell/src/shared/http/apiBase";
import { getAuthJsonHeaders } from "../../../shell/src/shared/http/authHeaders";

/**
 * console(brand) 用 TokenBlueprintReview HTTP repository
 *
 * 役割:
 * - aggregate 一覧取得
 * - comments 一覧取得
 * - brand からの comment / reply / comment reaction
 */

async function apiGetJson<T>(path: string): Promise<T> {
  const headers = await getAuthJsonHeaders();

  const res = await fetch(`${API_BASE}${path}`, {
    method: "GET",
    headers: {
      ...headers,
      Accept: "application/json",
    },
    credentials: "include",
  });

  const text = await res.text().catch(() => "");
  if (!res.ok) {
    throw new Error(text || `GET ${path} failed: ${res.status}`);
  }

  if (!text) return {} as T;

  try {
    return JSON.parse(text) as T;
  } catch {
    throw new Error(text);
  }
}

async function apiSendJson<T>(method: "POST" | "DELETE", path: string, body?: unknown): Promise<T> {
  const headers = await getAuthJsonHeaders();

  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers: {
      ...headers,
      Accept: "application/json",
      ...(method === "POST" ? { "Content-Type": "application/json" } : {}),
    },
    body: method === "POST" ? JSON.stringify(body ?? {}) : undefined,
    credentials: "include",
  });

  const text = await res.text().catch(() => "");
  if (!res.ok) {
    throw new Error(text || `${method} ${path} failed: ${res.status}`);
  }

  if (!text) return {} as T;

  try {
    return JSON.parse(text) as T;
  } catch {
    throw new Error(text);
  }
}

// ============================================================
// Response DTOs
// ============================================================

type ListTokenBlueprintReviewAggregatesResponse = {
  items: ApiTokenBlueprintReviewAggregate[];
};

export type ListTokenBlueprintCommentsResponse = {
  items: ApiComment[];
  tokenBlueprintName?: string;
  brandName?: string;
};

type CreateBrandCommentResponse = {
  item: ApiComment;
};

type ReactToCommentResponse = {
  item: ApiComment;
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
 */
export async function listTokenBlueprintReviewAggregatesByCompanyId(
  _companyId: string,
): Promise<TokenBlueprintReviewAggregate[]> {
  const data = await apiGetJson<ListTokenBlueprintReviewAggregatesResponse>(
    "/token-blueprint-reviews",
  );

  const rawItems = Array.isArray((data as { items?: unknown[] })?.items)
    ? ((data as { items?: unknown[] }).items ?? [])
    : [];

  return (rawItems as ApiTokenBlueprintReviewAggregate[]).map(
    fromApiTokenBlueprintReviewAggregate,
  );
}

// ============================================================
// Comments
// ============================================================

/**
 * backend: GET /token-blueprint-reviews/{tokenBlueprintId}/comments
 *
 * IMPORTANT:
 * - この関数は detail 表示用に top-level + replies を含む全 comments を返す前提。
 * - backend が top-level のみ返す実装のままだと reply は画面表示できない。
 */
export async function listTokenBlueprintCommentsByTokenBlueprintId(
  tokenBlueprintId: string,
): Promise<{
  items: Comment[];
  tokenBlueprintName?: string;
  brandName?: string;
}> {
  const id = String(tokenBlueprintId || "");
  if (!id) return { items: [] };

  const data = await apiGetJson<ListTokenBlueprintCommentsResponse>(
    `/token-blueprint-reviews/${encodeURIComponent(id)}/comments`,
  );

  const rawItems = Array.isArray((data as { items?: unknown[] })?.items)
    ? ((data as { items?: unknown[] }).items ?? [])
    : [];

  return {
    items: (rawItems as ApiComment[]).map(fromApiComment),
    tokenBlueprintName:
      data?.tokenBlueprintName != null && String(data.tokenBlueprintName) !== ""
        ? String(data.tokenBlueprintName)
        : undefined,
    brandName:
      data?.brandName != null && String(data.brandName) !== ""
        ? String(data.brandName)
        : undefined,
  };
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
  const id = String(tokenBlueprintId || "").trim();
  const content = String(body || "").trim();

  if (!id) {
    throw new Error("tokenBlueprintId is required");
  }
  if (!content) {
    throw new Error("body is required");
  }

  const req: CreateBrandCommentRequest = {
    body: content,
    ...(options?.commentId ? { commentId: options.commentId } : {}),
    ...(options?.parentCommentId ? { parentCommentId: options.parentCommentId } : {}),
  };

  const data = await apiSendJson<CreateBrandCommentResponse>(
    "POST",
    `/token-blueprint-reviews/${encodeURIComponent(id)}/comments`,
    req,
  );

  if (!data?.item) {
    throw new Error("comment response item is missing");
  }

  return fromApiComment(data.item);
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
  const id = String(tokenBlueprintId || "").trim();
  const parentId = String(parentCommentId || "").trim();
  const content = String(body || "").trim();

  if (!id) {
    throw new Error("tokenBlueprintId is required");
  }
  if (!parentId) {
    throw new Error("parentCommentId is required");
  }
  if (!content) {
    throw new Error("body is required");
  }

  const req: CreateBrandCommentRequest = {
    body: content,
    ...(options?.commentId ? { commentId: options.commentId } : {}),
  };

  const data = await apiSendJson<CreateBrandCommentResponse>(
    "POST",
    `/token-blueprint-reviews/${encodeURIComponent(id)}/comments/${encodeURIComponent(parentId)}/replies`,
    req,
  );

  if (!data?.item) {
    throw new Error("reply response item is missing");
  }

  return fromApiComment(data.item);
}

/**
 * backend: DELETE /token-blueprint-reviews/{tokenBlueprintId}/comments/{commentId}
 */
export async function deleteBrandComment(
  tokenBlueprintId: string,
  commentId: string,
): Promise<void> {
  const id = String(tokenBlueprintId || "").trim();
  const cid = String(commentId || "").trim();

  if (!id) {
    throw new Error("tokenBlueprintId is required");
  }
  if (!cid) {
    throw new Error("commentId is required");
  }

  await apiSendJson<void>(
    "DELETE",
    `/token-blueprint-reviews/${encodeURIComponent(id)}/comments/${encodeURIComponent(cid)}`,
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
  const id = String(tokenBlueprintId || "").trim();
  const cid = String(commentId || "").trim();

  if (!id) {
    throw new Error("tokenBlueprintId is required");
  }
  if (!cid) {
    throw new Error("commentId is required");
  }

  const data = await apiSendJson<ReactToCommentResponse>(
    "POST",
    `/token-blueprint-reviews/${encodeURIComponent(id)}/comments/${encodeURIComponent(cid)}/reactions`,
    { type } satisfies ReactAsBrandRequest,
  );

  if (!data?.item) {
    throw new Error("comment reaction response item is missing");
  }

  return fromApiComment(data.item);
}