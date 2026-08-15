// frontend/console/shell/src/features/tokenBlueprint/infrastructure/repository/tokenBlueprintRepositoryHTTP.ts

import { auth } from "../../../../auth/infrastructure/config/firebaseClient";
import type { ContentFile, TokenBlueprint } from "../../../../shared/types/tokenBlueprint";
import type { PageParams, PageResult } from "../../../../shared/types/common/common";
import { buildConsoleUrl } from "../../../../shared/http/apiBase";
import { getAuthHeaders } from "../../../../shared/http/authHeaders";
import { fetchJSON } from "../../../../shared/http/fetchJSON";

// ---------------------------------------------------------
// HTTP共通処理
// ---------------------------------------------------------

type JsonRequestMethod = "GET" | "POST" | "PUT";

function getActorIdOrEmpty(): string {
  try {
    return auth.currentUser?.uid ?? "";
  } catch {
    return "";
  }
}

function withActorHeader(headers: Record<string, string>): Record<string, string> {
  const actorId = getActorIdOrEmpty();

  if (!actorId) {
    return headers;
  }

  return {
    ...headers,
    "X-Actor-Id": actorId,
  };
}

async function requestJson<T>(
  path: string,
  method: JsonRequestMethod,
  body?: unknown,
): Promise<T> {
  const authHeaders = await getAuthHeaders();

  const headers = withActorHeader({
    ...authHeaders,
    ...(method === "GET" ? {} : { "Content-Type": "application/json" }),
  });

  return fetchJSON<T>(buildConsoleUrl(path), {
    method,
    headers,
    ...(method === "GET"
      ? {}
      : {
          body: JSON.stringify(body ?? {}),
        }),
  });
}

async function requestDelete(path: string): Promise<void> {
  const authHeaders = await getAuthHeaders();
  const headers = withActorHeader(authHeaders);
  const url = buildConsoleUrl(path);

  const response = await fetch(url, {
    method: "DELETE",
    headers,
  });

  if (response.ok) {
    return;
  }

  const text = await response.text();

  if (text) {
    try {
      const data = JSON.parse(text) as {
        error?: unknown;
        message?: unknown;
      };

      const message =
        typeof data.error === "string"
          ? data.error
          : typeof data.message === "string"
            ? data.message
            : "";

      if (message) {
        throw new Error(message);
      }
    } catch (error) {
      if (
        error instanceof Error &&
        error.message !== "Unexpected end of JSON input"
      ) {
        throw error;
      }
    }
  }

  throw new Error(
    text ||
      response.statusText ||
      `HTTP ${response.status}`,
  );
}

// ---------------------------------------------------------
// APIレスポンス型
// ---------------------------------------------------------

export type TokenBlueprintPageResult = PageResult<TokenBlueprint>;

// ---------------------------------------------------------
// API送信payload型
// ---------------------------------------------------------

export type CreateTokenBlueprintPayload = {
  name: string;
  symbol: string;
  brandId: string;
  description: string;
  assigneeId: string;
  iconUrl?: string | null;
  iconObjectPath?: string | null;
  iconFileName?: string | null;
  iconContentType?: string | null;
  iconSize?: number | null;
  contentFiles: ContentFile[];
};

export type UpdateTokenBlueprintPayload = Partial<{
  name: string;
  symbol: string;
  description: string;
  assigneeId: string;
  iconUrl: string | null;
  iconObjectPath: string | null;
  iconFileName: string | null;
  iconContentType: string | null;
  iconSize: number | null;
  contentFiles: ContentFile[];
}>;

// ---------------------------------------------------------
// TokenBlueprint API
// ---------------------------------------------------------

export async function fetchTokenBlueprints(
  params: PageParams = {},
): Promise<TokenBlueprintPageResult> {
  const searchParams = new URLSearchParams();

  if (params.page !== undefined) {
    searchParams.set("page", String(params.page));
  }

  if (params.perPage !== undefined) {
    searchParams.set("perPage", String(params.perPage));
  }

  const query = searchParams.toString();
  const path = query
    ? `/token-blueprints?${query}`
    : "/token-blueprints";

  return requestJson<TokenBlueprintPageResult>(path, "GET");
}

export async function fetchTokenBlueprintById(
  id: string,
): Promise<TokenBlueprint> {
  if (!id) {
    throw new Error("id is required");
  }

  return requestJson<TokenBlueprint>(
    `/token-blueprints/${encodeURIComponent(id)}`,
    "GET",
  );
}

export async function createTokenBlueprint(
  payload: CreateTokenBlueprintPayload,
): Promise<TokenBlueprint> {
  return requestJson<TokenBlueprint>(
    "/token-blueprints",
    "POST",
    payload,
  );
}

export async function updateTokenBlueprint(
  id: string,
  payload: UpdateTokenBlueprintPayload,
): Promise<TokenBlueprint> {
  if (!id) {
    throw new Error("id is required");
  }

  return requestJson<TokenBlueprint>(
    `/token-blueprints/${encodeURIComponent(id)}`,
    "PUT",
    payload,
  );
}

export async function deleteTokenBlueprint(
  id: string,
): Promise<void> {
  if (!id) {
    throw new Error("id is required");
  }

  await requestDelete(
    `/token-blueprints/${encodeURIComponent(id)}`,
  );
}

// ---------------------------------------------------------
// TokenBlueprint content API
// ---------------------------------------------------------

export async function patchTokenBlueprintContentFiles(
  params: {
    tokenBlueprintId: string;
    contentFiles: ContentFile[];
  },
): Promise<TokenBlueprint> {
  if (!params.tokenBlueprintId) {
    throw new Error("tokenBlueprintId is required");
  }

  return requestJson<TokenBlueprint>(
    `/token-blueprints/${encodeURIComponent(params.tokenBlueprintId)}`,
    "PUT",
    {
      contentFiles: params.contentFiles,
    },
  );
}

// ---------------------------------------------------------
// TokenBlueprint icon API
// ---------------------------------------------------------

export async function attachTokenBlueprintIcon(
  params: {
    tokenBlueprintId: string;
    iconUrl: string;
    iconObjectPath: string;
    iconFileName?: string | null;
    iconContentType?: string | null;
    iconSize?: number | null;
  },
): Promise<TokenBlueprint> {
  if (!params.tokenBlueprintId) {
    throw new Error("tokenBlueprintId is required");
  }

  return requestJson<TokenBlueprint>(
    `/token-blueprints/${encodeURIComponent(params.tokenBlueprintId)}`,
    "PUT",
    {
      iconUrl: params.iconUrl,
      iconObjectPath: params.iconObjectPath,
      iconFileName: params.iconFileName,
      iconContentType: params.iconContentType,
      iconSize: params.iconSize,
    },
  );
}