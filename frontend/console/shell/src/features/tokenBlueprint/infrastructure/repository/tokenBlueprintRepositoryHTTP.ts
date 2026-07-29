// frontend/console/shell/src/features/tokenBlueprint/infrastructure/repository/tokenBlueprintRepositoryHTTP.ts

import type { TokenBlueprint } from "../../../../shared/types/tokenBlueprint";
import type { ContentFileDTO } from "../dto/tokenBlueprint.dto";

import {
  apiDelete,
  apiGet,
  apiPostJson,
  apiPutJson,
} from "../http/client";

import { handleJsonResponse } from "../http/json";

import {
  normalizePageResult,
  normalizeTokenBlueprint,
} from "../dto/tokenBlueprint.mapper";

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
  const url = new URL(
    "/token-blueprints",
    "http://local",
  );

  if (params?.page !== undefined) {
    url.searchParams.set(
      "page",
      String(params.page),
    );
  }

  if (params?.perPage !== undefined) {
    url.searchParams.set(
      "perPage",
      String(params.perPage),
    );
  }

  const response = await apiGet(
    url.pathname + url.search,
  );

  const raw =
    await handleJsonResponse<unknown>(
      response,
    );

  return normalizePageResult(raw);
}

export async function fetchTokenBlueprintById(
  id: string,
): Promise<TokenBlueprint> {
  if (!id) {
    throw new Error(
      "id is required",
    );
  }

  const response = await apiGet(
    `/token-blueprints/${encodeURIComponent(id)}`,
  );

  const raw =
    await handleJsonResponse<unknown>(
      response,
    );

  return normalizeTokenBlueprint(raw);
}

export async function createTokenBlueprint(
  payload: CreateTokenBlueprintPayload,
): Promise<TokenBlueprint> {
  const response = await apiPostJson(
    "/token-blueprints",
    payload,
  );

  const raw =
    await handleJsonResponse<unknown>(
      response,
    );

  return normalizeTokenBlueprint(raw);
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

  const response = await apiPutJson(
    `/token-blueprints/${encodeURIComponent(id)}`,
    payload,
  );

  const raw =
    await handleJsonResponse<unknown>(
      response,
    );

  return normalizeTokenBlueprint(raw);
}

export async function deleteTokenBlueprint(
  id: string,
): Promise<void> {
  if (!id) {
    throw new Error(
      "id is required",
    );
  }

  const response = await apiDelete(
    `/token-blueprints/${encodeURIComponent(id)}`,
  );

  await handleJsonResponse<unknown>(
    response,
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

  const response = await apiPutJson(
    `/token-blueprints/${encodeURIComponent(
      params.tokenBlueprintId,
    )}`,
    {
      contentFiles: params.contentFiles,
    },
  );

  const raw =
    await handleJsonResponse<unknown>(
      response,
    );

  return normalizeTokenBlueprint(raw);
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

  const response = await apiPutJson(
    `/token-blueprints/${encodeURIComponent(
      params.tokenBlueprintId,
    )}`,
    {
      iconUrl: params.iconUrl,
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

  const raw =
    await handleJsonResponse<unknown>(
      response,
    );

  return normalizeTokenBlueprint(raw);
}