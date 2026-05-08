// frontend/amol/src/features/token-commnet/api/tokenCommentApi.ts

import { getAuth } from "firebase/auth";

import type {
  TokenBlueprintReactionInput,
  TokenBlueprintReviewAggregate,
  TokenComment,
  TokenCommentListResponse,
  TokenCommentPostInput,
  TokenCommentReplyInput,
  TokenCommentVoteInput,
} from "../types/tokenCommentTypes";

const BACKEND_BASE_URL = import.meta.env.VITE_API_BASE_URL;
const TOKEN_BLUEPRINT_BASE_PATH = "/mall/me/token-blueprints";

function normalizeBackendUrl(backendUrl: string): string {
  return backendUrl.replace(/\/+$/, "");
}

function assertBackendBaseUrl(): string {
  if (!BACKEND_BASE_URL) {
    throw new Error("VITE_API_BASE_URL is not configured.");
  }

  return normalizeBackendUrl(BACKEND_BASE_URL);
}

async function getIdToken(): Promise<string> {
  const auth = getAuth();
  const user = auth.currentUser;

  if (!user) {
    throw new Error("ログインが必要です。");
  }

  return user.getIdToken();
}

async function requestJson<T>(
  path: string,
  init?: RequestInit
): Promise<T> {
  const baseUrl = assertBackendBaseUrl();
  const idToken = await getIdToken();

  const response = await fetch(`${baseUrl}${path}`, {
    ...init,
    headers: {
      Accept: "application/json",
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      Authorization: `Bearer ${idToken}`,
      ...init?.headers,
    },
  });

  if (!response.ok) {
    const body = await response.text().catch(() => "");
    throw new Error(`token comment API failed: ${response.status} ${body}`);
  }

  const contentType = response.headers.get("content-type") || "";

  if (!contentType.includes("application/json")) {
    throw new Error("token comment API が JSON 以外を返しました。");
  }

  return response.json() as Promise<T>;
}

async function requestNoContent(
  path: string,
  init?: RequestInit
): Promise<void> {
  const baseUrl = assertBackendBaseUrl();
  const idToken = await getIdToken();

  const response = await fetch(`${baseUrl}${path}`, {
    ...init,
    headers: {
      Accept: "application/json",
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      Authorization: `Bearer ${idToken}`,
      ...init?.headers,
    },
  });

  if (!response.ok) {
    const body = await response.text().catch(() => "");
    throw new Error(`token comment API failed: ${response.status} ${body}`);
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function pick(
  value: Record<string, unknown>,
  keys: readonly string[]
): unknown {
  for (const key of keys) {
    if (Object.prototype.hasOwnProperty.call(value, key)) {
      return value[key];
    }
  }

  return undefined;
}

function asString(value: unknown): string {
  if (value === null || value === undefined) {
    return "";
  }

  if (typeof value === "string") {
    return value;
  }

  return String(value);
}

function asNullableString(value: unknown): string | null {
  if (value === null || value === undefined) {
    return null;
  }

  if (typeof value === "string") {
    return value;
  }

  return String(value);
}

function asNumber(value: unknown): number {
  if (typeof value === "number") {
    return Number.isFinite(value) ? value : 0;
  }

  if (typeof value === "string") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }

  return 0;
}

function asBoolean(value: unknown): boolean {
  if (typeof value === "boolean") {
    return value;
  }

  if (typeof value === "number") {
    return value !== 0;
  }

  if (typeof value === "string") {
    const normalized = value.trim().toLowerCase();
    return normalized === "true" || normalized === "1";
  }

  return false;
}

function parseTokenBlueprintReviewAggregate(
  value: unknown,
  fallbackTokenBlueprintId = ""
): TokenBlueprintReviewAggregate {
  if (!isRecord(value)) {
    return {
      tokenBlueprintId: fallbackTokenBlueprintId,
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
      asString(pick(value, ["tokenBlueprintId", "TokenBlueprintID"])) ||
      fallbackTokenBlueprintId,
    likeCount: asNumber(pick(value, ["likeCount", "LikeCount"])),
    dislikeCount: asNumber(pick(value, ["dislikeCount", "DislikeCount"])),
    topLevelCommentCount: asNumber(
      pick(value, ["topLevelCommentCount", "TopLevelCommentCount"])
    ),
    totalCommentCount: asNumber(
      pick(value, ["totalCommentCount", "TotalCommentCount"])
    ),
    createdAt: asNullableString(pick(value, ["createdAt", "CreatedAt"])),
    updatedAt: asNullableString(pick(value, ["updatedAt", "UpdatedAt"])),
  };
}

function parseTokenComment(value: unknown): TokenComment | null {
  if (!isRecord(value)) {
    return null;
  }

  return {
    commentId: asString(pick(value, ["commentId", "CommentID"])),
    tokenBlueprintId: asString(
      pick(value, ["tokenBlueprintId", "TokenBlueprintID"])
    ),
    parentCommentId: asString(
      pick(value, ["parentCommentId", "ParentCommentID"])
    ),
    rootCommentId: asString(pick(value, ["rootCommentId", "RootCommentID"])),
    depth: asNumber(pick(value, ["depth", "Depth"])),
    authorId: asString(pick(value, ["authorId", "AuthorID"])),
    authorType: asString(pick(value, ["authorType", "AuthorType"])),
    isOwnerComment: asBoolean(
      pick(value, ["isOwnerComment", "IsOwnerComment"])
    ),
    body: asString(pick(value, ["body", "Body"])),
    likeCount: asNumber(pick(value, ["likeCount", "LikeCount"])),
    dislikeCount: asNumber(pick(value, ["dislikeCount", "DislikeCount"])),
    childCount: asNumber(pick(value, ["childCount", "ChildCount"])),
    deleted: asBoolean(pick(value, ["deleted", "Deleted"])),
    createdAt: asString(pick(value, ["createdAt", "CreatedAt"])),
    updatedAt: asString(pick(value, ["updatedAt", "UpdatedAt"])),
    authorAvatarName: asNullableString(
      pick(value, ["authorAvatarName", "AuthorAvatarName"])
    ),
    authorAvatarIcon: asNullableString(
      pick(value, ["authorAvatarIcon", "AuthorAvatarIcon"])
    ),
    brandName: asNullableString(pick(value, ["brandName", "BrandName"])),
    brandIcon: asNullableString(pick(value, ["brandIcon", "BrandIcon"])),
  };
}

function parseTokenCommentListResponse(
  value: unknown
): TokenCommentListResponse {
  if (!isRecord(value)) {
    return {
      items: [],
      page: 1,
      perPage: 20,
      total: 0,
      tokenBlueprintName: null,
      brandName: null,
    };
  }

  const rawItems = pick(value, ["items", "Items"]);
  const items = Array.isArray(rawItems)
    ? rawItems
        .map(parseTokenComment)
        .filter((comment): comment is TokenComment => comment !== null)
    : [];

  return {
    items,
    page: asNumber(pick(value, ["page", "Page"])) || 1,
    perPage: asNumber(pick(value, ["perPage", "PerPage"])) || 20,
    total: asNumber(
      pick(value, ["totalCount", "TotalCount", "total", "Total"])
    ),
    tokenBlueprintName: asNullableString(
      pick(value, ["tokenBlueprintName", "TokenBlueprintName"])
    ),
    brandName: asNullableString(pick(value, ["brandName", "BrandName"])),
  };
}

function encodePathSegment(value: string): string {
  return encodeURIComponent(value);
}

export async function fetchTokenBlueprintReviewAggregate(
  tokenBlueprintId: string
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

  const body = await requestJson<unknown>(
    `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(
      tokenBlueprintId
    )}/reviews/aggregate`,
    {
      method: "GET",
    }
  );

  return parseTokenBlueprintReviewAggregate(body, tokenBlueprintId);
}

export async function upsertTokenBlueprintReaction({
  tokenBlueprintId,
  type,
}: TokenBlueprintReactionInput): Promise<void> {
  if (!tokenBlueprintId || !type) {
    return;
  }

  await requestNoContent(
    `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(
      tokenBlueprintId
    )}/reactions`,
    {
      method: "POST",
      body: JSON.stringify({
        type,
      }),
    }
  );
}

export async function fetchTokenComments(
  tokenBlueprintId: string
): Promise<TokenCommentListResponse> {
  if (!tokenBlueprintId) {
    return {
      items: [],
      page: 1,
      perPage: 20,
      total: 0,
      tokenBlueprintName: null,
      brandName: null,
    };
  }

  const body = await requestJson<unknown>(
    `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(
      tokenBlueprintId
    )}/comments`,
    {
      method: "GET",
    }
  );

  return parseTokenCommentListResponse(body);
}

export async function postTokenComment({
  tokenBlueprintId,
  body,
}: TokenCommentPostInput): Promise<void> {
  const trimmedBody = body.trim();

  if (!tokenBlueprintId || !trimmedBody) {
    return;
  }

  await requestNoContent(
    `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(
      tokenBlueprintId
    )}/comments`,
    {
      method: "POST",
      body: JSON.stringify({
        body: trimmedBody,
      }),
    }
  );
}

export async function postTokenCommentReply({
  tokenBlueprintId,
  parentCommentId,
  body,
}: TokenCommentReplyInput): Promise<void> {
  const trimmedBody = body.trim();

  if (!tokenBlueprintId || !parentCommentId || !trimmedBody) {
    return;
  }

  await requestNoContent(
    `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(
      tokenBlueprintId
    )}/comments/${encodePathSegment(parentCommentId)}/replies`,
    {
      method: "POST",
      body: JSON.stringify({
        body: trimmedBody,
      }),
    }
  );
}

export async function likeTokenComment({
  tokenBlueprintId,
  commentId,
}: TokenCommentVoteInput): Promise<void> {
  if (!tokenBlueprintId || !commentId) {
    return;
  }

  await requestNoContent(
    `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(
      tokenBlueprintId
    )}/comments/${encodePathSegment(commentId)}/reactions`,
    {
      method: "POST",
      body: JSON.stringify({
        type: "like",
      }),
    }
  );
}

export async function dislikeTokenComment({
  tokenBlueprintId,
  commentId,
}: TokenCommentVoteInput): Promise<void> {
  if (!tokenBlueprintId || !commentId) {
    return;
  }

  await requestNoContent(
    `${TOKEN_BLUEPRINT_BASE_PATH}/${encodePathSegment(
      tokenBlueprintId
    )}/comments/${encodePathSegment(commentId)}/reactions`,
    {
      method: "POST",
      body: JSON.stringify({
        type: "dislike",
      }),
    }
  );
}