// frontend/console/shell/src/features/tokenBlueprint/infrastructure/repository/tokenBlueprintRepositoryHTTP.ts

import { auth } from "../../../../auth/infrastructure/config/firebaseClient";

import type { TokenBlueprint } from "../../../../shared/types/tokenBlueprint";

import { buildConsoleUrl } from "../../../../shared/http/apiBase";

import {
  getAuthHeadersOrThrow,
  getAuthJsonHeadersOrThrow,
} from "../../../../shared/http/authHeaders";

import { fetchJSON } from "../../../../shared/http/fetchJSON";

import type { ContentFileDTO } from "../dto/tokenBlueprint.dto";

import {
  normalizeTokenBlueprint,
} from "../dto/tokenBlueprint.mapper";

// ---------------------------------------------------------
// HTTP共通処理
// ---------------------------------------------------------

type JsonRequestMethod =
  | "GET"
  | "POST"
  | "PUT";

function getActorIdOrEmpty(): string {
  try {
    return (
      auth.currentUser?.uid?.trim?.() ??
      ""
    );
  } catch {
    return "";
  }
}

function withActorHeader(
  headers: Record<string, string>,
): Record<string, string> {
  const actorId =
    getActorIdOrEmpty();

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
  const baseHeaders =
    method === "GET"
      ? await getAuthHeadersOrThrow()
      : await getAuthJsonHeadersOrThrow();

  const headers =
    withActorHeader(
      baseHeaders,
    );

  return fetchJSON<T>(
    buildConsoleUrl(path),
    {
      method,
      headers,
      ...(method === "GET"
        ? {}
        : {
            body: JSON.stringify(
              body ?? {},
            ),
          }),
    },
  );
}

async function requestDelete(
  path: string,
): Promise<void> {
  const baseHeaders =
    await getAuthHeadersOrThrow();

  const headers =
    withActorHeader(
      baseHeaders,
    );

  const url =
    buildConsoleUrl(path);

  const response =
    await fetch(
      url,
      {
        method: "DELETE",
        headers,
      },
    );

  if (response.ok) {
    return;
  }

  const text =
    await response.text();

  if (text) {
    try {
      const data =
        JSON.parse(text) as {
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
        throw new Error(
          message,
        );
      }
    } catch (error) {
      if (error instanceof Error) {
        if (
          error.message !==
          "Unexpected end of JSON input"
        ) {
          throw error;
        }
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

export interface TokenBlueprintPageResult {
  items: TokenBlueprint[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
}

// ---------------------------------------------------------
// API送信payload型
// ---------------------------------------------------------

export type CreateTokenBlueprintPayload = {
  name: string;
  symbol: string;

  brandId: string;
  companyId: string;

  description: string;

  assigneeId: string;
  createdBy: string;

  iconUrl?: string | null;
  iconObjectPath?: string | null;
  iconFileName?: string | null;
  iconContentType?: string | null;
  iconSize?: number | null;

  contentFiles: ContentFileDTO[];
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

  contentFiles: ContentFileDTO[];
}>;

// ---------------------------------------------------------
// TokenBlueprint API
// ---------------------------------------------------------

export async function fetchTokenBlueprints(
  params?: {
    page?: number;
    perPage?: number;
  },
): Promise<TokenBlueprintPageResult> {
  const searchParams =
    new URLSearchParams();

  if (params?.page !== undefined) {
    searchParams.set(
      "page",
      String(params.page),
    );
  }

  if (params?.perPage !== undefined) {
    searchParams.set(
      "perPage",
      String(params.perPage),
    );
  }

  const query =
    searchParams.toString();

  const path =
    query
      ? `/token-blueprints?${query}`
      : "/token-blueprints";

  return requestJson<TokenBlueprintPageResult>(
    path,
    "GET",
  );
}

export async function fetchTokenBlueprintById(
  id: string,
): Promise<TokenBlueprint> {
  if (!id) {
    throw new Error(
      "id is required",
    );
  }

  return requestJson<TokenBlueprint>(
    `/token-blueprints/${encodeURIComponent(id)}`,
    "GET",
  );
}

export async function createTokenBlueprint(
  payload: CreateTokenBlueprintPayload,
): Promise<TokenBlueprint> {
  const raw =
    await requestJson<unknown>(
      "/token-blueprints",
      "POST",
      payload,
    );

  return normalizeTokenBlueprint(
    raw,
  );
}

export async function updateTokenBlueprint(
  id: string,
  payload: UpdateTokenBlueprintPayload,
): Promise<TokenBlueprint> {
  if (!id) {
    throw new Error(
      "id is required",
    );
  }

  const raw =
    await requestJson<unknown>(
      `/token-blueprints/${encodeURIComponent(id)}`,
      "PUT",
      payload,
    );

  return normalizeTokenBlueprint(
    raw,
  );
}

export async function deleteTokenBlueprint(
  id: string,
): Promise<void> {
  if (!id) {
    throw new Error(
      "id is required",
    );
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
    contentFiles: ContentFileDTO[];
  },
): Promise<TokenBlueprint> {
  if (!params.tokenBlueprintId) {
    throw new Error(
      "tokenBlueprintId is required",
    );
  }

  const raw =
    await requestJson<unknown>(
      `/token-blueprints/${encodeURIComponent(
        params.tokenBlueprintId,
      )}`,
      "PUT",
      {
        contentFiles:
          params.contentFiles,
      },
    );

  return normalizeTokenBlueprint(
    raw,
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
    throw new Error(
      "tokenBlueprintId is required",
    );
  }

  const raw =
    await requestJson<unknown>(
      `/token-blueprints/${encodeURIComponent(
        params.tokenBlueprintId,
      )}`,
      "PUT",
      {
        iconUrl:
          params.iconUrl,

        iconObjectPath:
          params.iconObjectPath,

        iconFileName:
          params.iconFileName,

        iconContentType:
          params.iconContentType,

        iconSize:
          params.iconSize,
      },
    );

  return normalizeTokenBlueprint(
    raw,
  );
}