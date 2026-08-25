// frontend/console/shell/src/features/tokenBlueprint/infrastructure/repository/tokenBlueprintCreateOperationRepositoryHTTP.ts

import { buildConsoleUrl } from "../../../../shared/http/apiBase";

import { fetchJSON } from "../../../../shared/http/fetchJSON";

import type { ContentType } from "../../../../shared/types/tokenBlueprint";

const TOKEN_BLUEPRINT_CREATE_OPERATION_BASE_PATH =
  "/token-blueprints/create-operations";

type JsonRequestMethod =
  | "GET"
  | "POST"
  | "PUT";

export type TokenBlueprintCreateOperationStatus =
  | "waiting_upload"
  | "queued"
  | "processing"
  | "completed"
  | "failed_retryable"
  | "failed_fatal";

export type TokenBlueprintCreateOperationIcon = {

  fileName: string;

  contentType: string;

  size: number;

  url?: string;

  objectPath?: string;

  uploaded: boolean;

  uploadedAt?: string;

};

export type TokenBlueprintCreateOperationContent = {

  id: string;

  name: string;

  type: ContentType;

  contentType: string;

  size: number;

  url?: string;

  objectPath?: string;

  uploaded: boolean;

  uploadedAt?: string;

};

export type TokenBlueprintCreateOperation = {

  id: string;

  tokenBlueprintId: string;

  status: TokenBlueprintCreateOperationStatus;

  resumeStatus?: TokenBlueprintCreateOperationStatus;

  icon?: TokenBlueprintCreateOperationIcon;

  contents: TokenBlueprintCreateOperationContent[];

  expectedUploadCount: number;

  completedUploadCount: number;

  retryCount: number;

  maxRetries: number;

  lastError?: string;

  version: number;

  createdAt: string;

  updatedAt: string;

  failedAt?: string;

  completedAt?: string;

};

export type StartTokenBlueprintCreateOperationIcon = {

  fileName: string;

  contentType: string;

  size: number;

};

export type StartTokenBlueprintCreateOperationContent = {

  id: string;

  name: string;

  type: ContentType;

  contentType: string;

  size: number;

};

export type StartTokenBlueprintCreateOperationInput = {

  operationId?: string;

  idempotencyKey: string;

  name: string;

  symbol: string;

  brandId: string;

  description?: string;

  assigneeId: string;

  icon?: StartTokenBlueprintCreateOperationIcon;

  contents?: StartTokenBlueprintCreateOperationContent[];

  maxRetries?: number;

};

export type RegisterTokenBlueprintCreateOperationIconInput = {

  url: string;

  objectPath: string;

  fileName: string;

  contentType: string;

  size: number;

};

export type RegisterTokenBlueprintCreateOperationContentInput = {

  url: string;

  objectPath: string;

  name: string;

  contentType: string;

  size: number;

};

function requireOperationId(
  operationId: string,
): string {

  const value = operationId.trim();

  if (!value) {

    throw new Error(
      "operationId is required",
    );

  }

  return value;

}

function requireContentId(
  contentId: string,
): string {

  const value = contentId.trim();

  if (!value) {

    throw new Error(
      "contentId is required",
    );

  }

  return value;

}

function requireIdempotencyKey(
  idempotencyKey: string,
): string {

  const value = idempotencyKey.trim();

  if (!value) {

    throw new Error(
      "idempotencyKey is required",
    );

  }

  return value;

}

async function requestJson<T>(
  path: string,
  method: JsonRequestMethod,
  body?: unknown,
  headers?: Record<string, string>,
): Promise<T> {

  return fetchJSON<T>(
    buildConsoleUrl(path),
    {

      method,

      auth: "required",

      headers: {

        ...(method === "GET"
          ? {}
          : {
              "Content-Type": "application/json",
            }),

        ...headers,

      },

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

export async function startTokenBlueprintCreateOperation(
  input: StartTokenBlueprintCreateOperationInput,
): Promise<TokenBlueprintCreateOperation> {

  const idempotencyKey =
    requireIdempotencyKey(
      input.idempotencyKey,
    );

  return requestJson<TokenBlueprintCreateOperation>(
    TOKEN_BLUEPRINT_CREATE_OPERATION_BASE_PATH,
    "POST",
    {

      operationId:
        input.operationId,

      idempotencyKey,

      name:
        input.name,

      symbol:
        input.symbol,

      brandId:
        input.brandId,

      description:
        input.description ?? "",

      assigneeId:
        input.assigneeId,

      icon:
        input.icon,

      contents:
        input.contents ?? [],

      maxRetries:
        input.maxRetries,

    },
    {

      "Idempotency-Key":
        idempotencyKey,

    },
  );

}

export async function fetchTokenBlueprintCreateOperation(
  operationId: string,
): Promise<TokenBlueprintCreateOperation> {

  const id =
    requireOperationId(
      operationId,
    );

  return requestJson<TokenBlueprintCreateOperation>(
    `${TOKEN_BLUEPRINT_CREATE_OPERATION_BASE_PATH}/${encodeURIComponent(id)}`,
    "GET",
  );

}

export async function registerTokenBlueprintCreateOperationIcon(
  operationId: string,
  input: RegisterTokenBlueprintCreateOperationIconInput,
): Promise<TokenBlueprintCreateOperation> {

  const id =
    requireOperationId(
      operationId,
    );

  return requestJson<TokenBlueprintCreateOperation>(
    `${TOKEN_BLUEPRINT_CREATE_OPERATION_BASE_PATH}/${encodeURIComponent(id)}/icon`,
    "PUT",
    {

      url:
        input.url,

      objectPath:
        input.objectPath,

      fileName:
        input.fileName,

      contentType:
        input.contentType,

      size:
        input.size,

    },
  );

}

export async function registerTokenBlueprintCreateOperationContent(
  operationId: string,
  contentId: string,
  input: RegisterTokenBlueprintCreateOperationContentInput,
): Promise<TokenBlueprintCreateOperation> {

  const id =
    requireOperationId(
      operationId,
    );

  const normalizedContentId =
    requireContentId(
      contentId,
    );

  return requestJson<TokenBlueprintCreateOperation>(
    `${TOKEN_BLUEPRINT_CREATE_OPERATION_BASE_PATH}/${encodeURIComponent(id)}/contents/${encodeURIComponent(normalizedContentId)}`,
    "PUT",
    {

      url:
        input.url,

      objectPath:
        input.objectPath,

      name:
        input.name,

      contentType:
        input.contentType,

      size:
        input.size,

    },
  );

}

export async function commitTokenBlueprintCreateOperation(
  operationId: string,
): Promise<TokenBlueprintCreateOperation> {

  const id =
    requireOperationId(
      operationId,
    );

  return requestJson<TokenBlueprintCreateOperation>(
    `${TOKEN_BLUEPRINT_CREATE_OPERATION_BASE_PATH}/${encodeURIComponent(id)}/commit`,
    "POST",
    {},
  );

}