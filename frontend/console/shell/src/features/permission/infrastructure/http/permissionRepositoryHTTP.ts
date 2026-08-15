// frontend/console/shell/src/features/permission/infrastructure/http/permissionRepositoryHTTP.ts

import { buildConsoleUrl } from "../../../../shared/http/apiBase";
import { getAuthHeaders } from "../../../../shared/http/authHeaders";
import type {
  PageRequest,
  PageResult,
  Sort,
} from "../../../../shared/types/common/common";
import type {
  Permission,
  PermissionCategory,
} from "../../../../shared/types/permission";

// ─────────────────────────────────────────────
// Backend API URL
// ─────────────────────────────────────────────

const PERMISSIONS_URL = buildConsoleUrl("/permissions");

// ─────────────────────────────────────────────
// Filter
// ─────────────────────────────────────────────

/**
 * backend/internal/domain/permission.Filter に対応する。
 */
export type PermissionFilter = {
  searchQuery?: string;
  categories?: PermissionCategory[];
};

/**
 * Permission一覧取得時のオプション。
 */
export type ListPermissionOptions = {
  filter?: PermissionFilter;
  sort?: Sort;
  page?: PageRequest;
};

// ─────────────────────────────────────────────
// Error response
// ─────────────────────────────────────────────

type ErrorResponse = {
  error?: unknown;
  message?: unknown;
};

class PermissionHttpError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "PermissionHttpError";
    this.status = status;
  }
}

// ─────────────────────────────────────────────
// HTTP utilities
// ─────────────────────────────────────────────

async function requestJson<T>(
  input: string,
  init: RequestInit = {},
): Promise<T> {
  const authHeaders: Record<string, string> = await getAuthHeaders();
  const headers = new Headers(init.headers);

  for (const [key, value] of Object.entries(authHeaders)) {
    headers.set(key, value);
  }

  const response = await fetch(input, {
    ...init,
    headers,
  });

  const text = await response.text().catch(() => "");

  if (!response.ok) {
    throw new PermissionHttpError(
      response.status,
      resolveErrorMessage(response, text),
    );
  }

  if (!text) {
    return undefined as T;
  }

  const looksLikeHTML = /^\s*<!doctype html>|^\s*<html/i.test(text);

  if (looksLikeHTML) {
    throw new Error(
      "[PermissionRepositoryHTTP] response is not JSON (HTML received). " +
        `VITE_BACKEND_BASE_URL の設定を確認してください。received head: ${text.slice(0, 120)}`,
    );
  }

  try {
    return JSON.parse(text) as T;
  } catch {
    throw new Error(
      `[PermissionRepositoryHTTP] JSON parse error. head: ${text.slice(0, 120)}`,
    );
  }
}

function resolveErrorMessage(
  response: Response,
  text: string,
): string {
  const fallbackMessage =
    `[PermissionRepositoryHTTP] ${response.status} ${response.statusText}`;

  if (!text) {
    return fallbackMessage;
  }

  try {
    const body = JSON.parse(text) as ErrorResponse;
    const backendMessage = body.error ?? body.message;

    if (typeof backendMessage === "string" && backendMessage) {
      return backendMessage;
    }
  } catch {
    // JSON形式ではない場合はfallbackを使用する。
  }

  return `${fallbackMessage} :: ${text.slice(0, 300)}`;
}

// ─────────────────────────────────────────────
// Query utilities
// ─────────────────────────────────────────────

function buildQuery(
  params: Record<string, string | number | undefined | null>,
): string {
  const searchParams = new URLSearchParams();

  for (const [key, value] of Object.entries(params)) {
    if (
      value === undefined ||
      value === null ||
      value === ""
    ) {
      continue;
    }

    searchParams.set(key, String(value));
  }

  const query = searchParams.toString();
  return query ? `?${query}` : "";
}

// ─────────────────────────────────────────────
// Repository
// ─────────────────────────────────────────────

export class PermissionRepositoryHTTP {
  private readonly baseUrl: string;

  constructor(baseUrl: string = PERMISSIONS_URL) {
    this.baseUrl = baseUrl.replace(/\/+$/g, "");
  }

  /**
   * Permission一覧を取得する。
   *
   * GET /permissions
   */
  async list(
    options: ListPermissionOptions = {},
  ): Promise<PageResult<Permission>> {
    const { filter, sort, page } = options;

    const query = buildQuery({
      page: page?.number,
      perPage: page?.perPage,
      sort: sort?.column,
      order: sort?.order,
      search: filter?.searchQuery,
      categories:
        filter?.categories?.length
          ? filter.categories.join(",")
          : undefined,
    });

    return requestJson<PageResult<Permission>>(
      `${this.baseUrl}${query}`,
      {
        method: "GET",
      },
    );
  }

  /**
   * IDを指定してPermissionを取得する。
   *
   * GET /permissions/:id
   */
  async getById(id: string): Promise<Permission> {
    const permissionId = id.trim();

    if (!permissionId) {
      throw new Error("permission id is required");
    }

    try {
      return await requestJson<Permission>(
        `${this.baseUrl}/${encodeURIComponent(permissionId)}`,
        {
          method: "GET",
        },
      );
    } catch (error: unknown) {
      if (
        error instanceof PermissionHttpError &&
        error.status === 404
      ) {
        throw new Error("permission not found");
      }

      throw error;
    }
  }
}