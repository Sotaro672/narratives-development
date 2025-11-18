// frontend/console/brand/src/infrastructure/http/brandRepositoryHTTP.ts

import type { Brand, BrandPatch } from "../../domain/entity/brand";

/**
 * BrandFilter
 * backend/internal/domain/brand/filter.go に対応する想定のフロント側フィルタ。
 */
export interface BrandFilter {
  companyId?: string;
  managerId?: string;
  isActive?: boolean;
  walletAddress?: string;
  deleted?: boolean;
}

/**
 * BrandSort
 * backend/internal/domain/brand/sort.go に対応する想定のソート指定。
 * column は backend 側の mapBrandSort と対応するキーを利用。
 */
export interface BrandSort {
  column?: "name" | "is_active" | "updated_at" | "created_at";
  order?: "asc" | "desc";
}

/**
 * ページング指定
 */
export interface PageParams {
  page?: number; // backend: Page.Number
  perPage?: number; // backend: Page.PerPage
}

/**
 * ページング付き結果
 * backend/internal/domain/common/page.go の PageResult<T> に対応する想定。
 */
export interface PageResult<T> {
  items: T[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
}

// ─────────────────────────────────────────────
// Backend base URL（.env 未設定でも空文字フォールバック）
// ─────────────────────────────────────────────
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE = "";
const BASE_URL = `${ENV_BASE || FALLBACK_BASE}/brands`;

/**
 * 共通の fetch ラッパ。
 * 必要に応じて Firebase Auth などで Authorization ヘッダを追加する実装に差し替えてください。
 */
async function httpRequest<T>(
  input: string,
  init: RequestInit = {},
): Promise<T> {
  const res = await fetch(input, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });

  if (!res.ok) {
    // ここでは簡易に Error を投げるだけにしておき、
    // 詳細なエラー処理は呼び出し側でラップしてください。
    const text = await res.text().catch(() => "");
    throw new Error(
      `BrandRepositoryHTTP request failed: ${res.status} ${res.statusText} ${text}`,
    );
  }

  // 空レスポンスの場合（204 No Contentなど）は undefined を返す
  if (res.status === 204) {
    return undefined as unknown as T;
  }

  return (await res.json()) as T;
}

// ─────────────────────────────────────────────
// Repository 本体
// ─────────────────────────────────────────────

/**
 * BrandRepositoryHTTP
 * backend 側の BrandRepository に対応する HTTP 実装。
 *
 * - エンドポイントのパスやクエリパラメータは仮の設計です。
 *   実際のバックエンド API 実装に合わせて修正してください。
 */
export class BrandRepositoryHTTP {
  private readonly baseUrl: string;

  constructor(baseUrl: string = BASE_URL) {
    this.baseUrl = baseUrl.replace(/\/+$/g, "");
  }

  // ==============================
  // Create
  // ==============================

  /**
   * Brand 作成。
   * backend: POST /brands
   *
   * @param input Brand 作成に必要なフィールド（id は省略可）
   */
  async create(input: Omit<Brand, "createdAt" | "updatedAt">): Promise<Brand> {
    return httpRequest<Brand>(this.baseUrl, {
      method: "POST",
      body: JSON.stringify(input),
    });
  }

  // ==============================
  // GetByID
  // ==============================

  /**
   * ID での単一取得。
   * backend: GET /brands/{id}
   */
  async getById(id: string): Promise<Brand> {
    const url = `${this.baseUrl}/${encodeURIComponent(id)}`;
    return httpRequest<Brand>(url, {
      method: "GET",
    });
  }

  // ==============================
  // Exists
  // ==============================

  /**
   * 存在確認。
   * backend: HEAD /brands/{id} または GET /brands/{id} を利用する想定。
   * 実際の API に合わせて実装を変更してください。
   */
  async exists(id: string): Promise<boolean> {
    const url = `${this.baseUrl}/${encodeURIComponent(id)}`;

    const res = await fetch(url, {
      method: "HEAD",
    });

    if (res.status === 404) return false;
    if (!res.ok) {
      const text = await res.text().catch(() => "");
      throw new Error(
        `BrandRepositoryHTTP.exists failed: ${res.status} ${res.statusText} ${text}`,
      );
    }
    return true;
  }

  // ==============================
  // Count
  // ==============================

  /**
   * 件数カウント。
   * backend: GET /brands/count?companyId=... などを想定。
   * 実際のエンドポイント仕様に合わせて調整してください。
   */
  async count(filter: BrandFilter = {}): Promise<number> {
    const params = new URLSearchParams();

    if (filter.companyId) params.set("companyId", filter.companyId);
    if (filter.managerId) params.set("managerId", filter.managerId);
    if (typeof filter.isActive === "boolean") {
      params.set("isActive", String(filter.isActive));
    }
    if (filter.walletAddress) params.set("walletAddress", filter.walletAddress);
    if (typeof filter.deleted === "boolean") {
      params.set("deleted", String(filter.deleted));
    }

    const url =
      params.toString().length > 0
        ? `${this.baseUrl}/count?${params.toString()}`
        : `${this.baseUrl}/count`;

    const result = await httpRequest<{ count: number }>(url, {
      method: "GET",
    });
    return result.count;
  }

  // ==============================
  // Update (partial)
  // ==============================

  /**
   * BrandPatch による部分更新。
   * backend: PATCH /brands/{id}
   */
  async update(id: string, patch: BrandPatch): Promise<Brand> {
    const url = `${this.baseUrl}/${encodeURIComponent(id)}`;
    return httpRequest<Brand>(url, {
      method: "PATCH",
      body: JSON.stringify(patch),
    });
  }

  // ==============================
  // Delete (hard delete)
  // ==============================

  /**
   * 完全削除。
   * backend: DELETE /brands/{id}
   */
  async delete(id: string): Promise<void> {
    const url = `${this.baseUrl}/${encodeURIComponent(id)}`;
    await httpRequest<void>(url, {
      method: "DELETE",
    });
  }

  // ==============================
  // List (filter/sort/pagination)
  // ==============================

  /**
   * 一覧取得（ページング付き）。
   * backend: GET /brands?companyId=...&managerId=...&isActive=...&walletAddress=...&column=created_at&order=desc&page=1&perPage=50
   */
  async list(options: {
    filter?: BrandFilter;
    sort?: BrandSort;
    page?: PageParams["page"];
    perPage?: PageParams["perPage"];
  } = {}): Promise<PageResult<Brand>> {
    const { filter = {}, sort = {}, page, perPage } = options;
    const params = new URLSearchParams();

    // Filter
    if (filter.companyId) params.set("companyId", filter.companyId);
    if (filter.managerId) params.set("managerId", filter.managerId);
    if (typeof filter.isActive === "boolean") {
      params.set("isActive", String(filter.isActive));
    }
    if (filter.walletAddress) params.set("walletAddress", filter.walletAddress);
    if (typeof filter.deleted === "boolean") {
      params.set("deleted", String(filter.deleted));
    }

    // Sort
    if (sort.column) params.set("column", sort.column);
    if (sort.order) params.set("order", sort.order);

    // Page
    if (page != null) params.set("page", String(page));
    if (perPage != null) params.set("perPage", String(perPage));

    const qs = params.toString();
    const url = qs ? `${this.baseUrl}?${qs}` : this.baseUrl;

    // backend の PageResult struct に合わせたレスポンス形式を想定
    const result = await httpRequest<{
      items: Brand[];
      totalCount: number;
      totalPages: number;
      page: number;
      perPage: number;
    }>(url, {
      method: "GET",
    });

    return {
      items: result.items ?? [],
      totalCount: result.totalCount ?? result.items?.length ?? 0,
      totalPages: result.totalPages ?? 1,
      page: result.page ?? page ?? 1,
      perPage: result.perPage ?? perPage ?? result.items?.length ?? 0,
    };
  }

  // ==============================
  // Save (Upsert)
  // ==============================

  /**
   * Upsert 的な保存。
   * backend: PUT /brands/{id} を想定（ID が空の場合は POST /brands 相当）。
   */
  async save(brand: Brand): Promise<Brand> {
    const trimmedId = brand.id?.trim();

    if (!trimmedId) {
      // ID が空なら create 扱い
      return this.create({
        ...brand,
        id: "", // backend が ID を採番する前提
      } as Omit<Brand, "createdAt" | "updatedAt">);
    }

    const url = `${this.baseUrl}/${encodeURIComponent(trimmedId)}`;
    return httpRequest<Brand>(url, {
      method: "PUT",
      body: JSON.stringify(brand),
    });
  }

  // ==============================
  // Reset (development/testing)
  // ==============================

  /**
   * 開発・テスト用の全削除リセット。
   * backend: POST /brands/reset を想定。
   */
  async reset(): Promise<void> {
    const url = `${this.baseUrl}/reset`;
    await httpRequest<void>(url, {
      method: "POST",
    });
  }
}

// デフォルトインスタンス（簡易利用用）
export const brandRepositoryHTTP = new BrandRepositoryHTTP();
