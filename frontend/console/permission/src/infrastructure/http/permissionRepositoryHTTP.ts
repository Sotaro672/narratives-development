// frontend/console/permission/src/infrastructure/http/permissionRepositoryHTTP.ts

import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";
import type { Permission } from "../../domain/entity/permission";
import type {
  Page,
  PageResult,
  Sort,
} from "../../../../shell/src/shared/types/common/common";

// ─────────────────────────────────────────────
// Backend base URL（.env 未設定でも Cloud Run にフォールバック）
// ─────────────────────────────────────────────
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  window.location.origin === "https://narratives.jp"
    ? "https://narratives-backend-871263659099.asia-northeast1.run.app"
    : "http://localhost:8080";

export const BACKEND_BASE_URL = ENV_BASE || FALLBACK_BASE;

// ─────────────────────────────────────────────
// Filter 型（backend/internal/domain/permission.Filter に対応）
// ─────────────────────────────────────────────

export type PermissionFilter = {
  // FilterCommon 相当
  searchQuery?: string;
  // カテゴリフィルタ（CategoryWallet などの string 値）
  categories?: string[];
};

// 一覧 API 呼び出し時のオプション
export type ListPermissionOptions = {
  filter?: PermissionFilter;
  sort?: Sort;
  page?: Page;
};

// ─────────────────────────────────────────────
// HTTP Utility
// ─────────────────────────────────────────────

// 認証付き fetch ヘルパ
async function authedFetch(input: RequestInfo, init: RequestInit = {}): Promise<Response> {
  const user = auth.currentUser;
  const headers = new Headers(init.headers || {});

  if (user) {
    const token = await user.getIdToken();
    headers.set("Authorization", `Bearer ${token}`);
  }

  // JSON 前提のヘッダ（必要に応じて上書き）
  if (!headers.has("Content-Type") && init.body) {
    headers.set("Content-Type", "application/json");
  }

  return fetch(input, {
    ...init,
    headers,
  });
}

// クエリストリング生成
function buildQuery(params: Record<string, string | number | undefined | null>): string {
  const usp = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === null || value === "") return;
    usp.set(key, String(value));
  });
  const qs = usp.toString();
  return qs ? `?${qs}` : "";
}

// ─────────────────────────────────────────────
// Repository 実装
// ─────────────────────────────────────────────

export class PermissionRepositoryHTTP {
  private baseUrl: string;

  constructor(baseUrl: string = BACKEND_BASE_URL) {
    this.baseUrl = baseUrl.replace(/\/+$/g, "");
  }

  // 一覧取得（GET /permissions）
  async list(options: ListPermissionOptions = {}): Promise<PageResult<Permission>> {
    const { filter, sort, page } = options;

    const qs = buildQuery({
      page: (page as any)?.number ?? (page as any)?.page,
      perPage: (page as any)?.perPage,
      sort: sort?.column,
      order: sort?.order,
      search: filter?.searchQuery,
      // categories は CSV で渡す（例: "wallet,brand,member"）
      categories: filter?.categories?.length
        ? filter.categories.join(",")
        : undefined,
    });

    const url = `${this.baseUrl}/permissions${qs}`;
    const res = await authedFetch(url, {
      method: "GET",
    });

    if (!res.ok) {
      // エラーレスポンスのメッセージをできるだけ拾う
      let message = `Failed to load permissions: ${res.status} ${res.statusText}`;
      try {
        const body = await res.json();
        if ((body as any)?.error) {
          message = String((body as any).error);
        }
      } catch {
        // ignore JSON parse error
      }
      throw new Error(message);
    }

    const raw = await res.json();

    // items を camelCase に正規化（Go 側の Permission は ID/Name/Category/Description）
    const rawItems = (raw.items ?? raw.Items ?? []) as any[];
    const normalizedItems: Permission[] = rawItems.map((it) => ({
      id: it.id ?? it.ID ?? "",
      name: it.name ?? it.Name ?? "",
      category: it.category ?? it.Category ?? "",
      description: it.description ?? it.Description ?? "",
    }));

    const data: PageResult<Permission> = {
      items: normalizedItems,
      totalCount: Number(raw.totalCount ?? raw.TotalCount ?? 0),
      totalPages: Number(raw.totalPages ?? raw.TotalPages ?? 1),
      page: Number(raw.page ?? raw.Page ?? 1),
      // ← ここを括弧で囲ってエラー回避
      perPage: Number(
        raw.perPage ?? raw.PerPage ?? (normalizedItems.length || 10),
      ),
    };

    return data;
  }

  // 単体取得（GET /permissions/:id）
  async getById(id: string): Promise<Permission> {
    const trimmed = id.trim();
    if (!trimmed) {
      throw new Error("permission id is required");
    }

    const url = `${this.baseUrl}/permissions/${encodeURIComponent(trimmed)}`;
    const res = await authedFetch(url, {
      method: "GET",
    });

    if (res.status === 404) {
      throw new Error("permission not found");
    }

    if (!res.ok) {
      let message = `Failed to load permission: ${res.status} ${res.statusText}`;
      try {
        const body = await res.json();
        if ((body as any)?.error) {
          message = String((body as any).error);
        }
      } catch {
        // ignore
      }
      throw new Error(message);
    }

    const raw = await res.json();
    const data: Permission = {
      id: raw.id ?? raw.ID ?? "",
      name: raw.name ?? raw.Name ?? "",
      category: raw.category ?? raw.Category ?? "",
      description: raw.description ?? raw.Description ?? "",
    };

    return data;
  }
}
