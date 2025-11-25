// frontend/console/brand/src/infrastructure/http/brandRepositoryHTTP.ts
import type { Brand, BrandPatch } from "../../domain/entity/brand";
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

/**
 * BrandFilter
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
 */
export interface BrandSort {
  column?: "name" | "is_active" | "updated_at" | "created_at";
  order?: "asc" | "desc";
}

/** ページング指定 */
export interface PageParams {
  page?: number;
  perPage?: number;
}

/** ページング付き結果 */
export interface PageResult<T> {
  items: T[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
}

// ─────────────────────────────────────────────
// Backend base URL
// ─────────────────────────────────────────────
const RAW_ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined) ?? "";
const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

function sanitizeBase(u: string): string {
  return (u || "").replace(/\/+$/g, "");
}

const ENV_BASE = sanitizeBase(RAW_ENV_BASE);
const FINAL_BASE = sanitizeBase(ENV_BASE || FALLBACK_BASE);

if (!FINAL_BASE) {
  throw new Error(
    "[BrandRepositoryHTTP] BACKEND BASE URL is empty. Set VITE_BACKEND_BASE_URL in .env.local",
  );
}

const BASE_URL = `${FINAL_BASE}/brands`;

/** 素の共通 fetch ラッパ（JSON前提） */
async function httpRequest<T>(input: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(input, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });

  if (res.status === 204) return undefined as unknown as T;

  const text = await res.text().catch(() => "");

  if (!res.ok) {
    throw new Error(
      `[BrandRepositoryHTTP] ${res.status} ${res.statusText} :: ${text?.slice(
        0,
        300,
      )}`,
    );
  }

  const looksLikeHTML = /^\s*<!doctype html>|^\s*<html/i.test(text);
  if (looksLikeHTML) {
    throw new Error(
      `[BrandRepositoryHTTP] response is not JSON (HTML received). ` +
        `BASE_URL の設定を確認してください。received head: ${text.slice(0, 120)}`,
    );
  }

  try {
    return text ? (JSON.parse(text) as T) : (undefined as unknown as T);
  } catch {
    throw new Error(
      `[BrandRepositoryHTTP] JSON parse error. head: ${text.slice(0, 120)}`,
    );
  }
}

/** 認証付きラッパ：Authorization: Bearer <ID_TOKEN> を自動付与 */
async function authed<T>(input: string, init: RequestInit = {}): Promise<T> {
  const token = await auth.currentUser?.getIdToken();
  if (!token) {
    throw new Error("[BrandRepositoryHTTP] Not authenticated (no ID token).");
  }
  return httpRequest<T>(input, {
    ...init,
    headers: {
      ...(init.headers ?? {}),
      Authorization: `Bearer ${token}`,
    },
  });
}

// ─────────────────────────────────────────────
// Repository 本体
// ─────────────────────────────────────────────
export class BrandRepositoryHTTP {
  private readonly baseUrl: string;

  constructor(baseUrl: string = BASE_URL) {
    this.baseUrl = sanitizeBase(baseUrl);
    if (!this.baseUrl) {
      throw new Error(
        "[BrandRepositoryHTTP] baseUrl is empty. Check VITE_BACKEND_BASE_URL or FALLBACK_BASE.",
      );
    }
  }

  // Create
  async create(input: Omit<Brand, "createdAt" | "updatedAt">): Promise<Brand> {
    return authed<Brand>(this.baseUrl, {
      method: "POST",
      body: JSON.stringify(input),
    });
  }

  // GetByID
  async getById(id: string): Promise<Brand> {
    const url = `${this.baseUrl}/${encodeURIComponent(id)}`;
    return authed<Brand>(url, { method: "GET" });
  }

  // Exists
  async exists(id: string): Promise<boolean> {
    const url = `${this.baseUrl}/${encodeURIComponent(id)}`;
    const token = await auth.currentUser?.getIdToken();
    if (!token) throw new Error("[BrandRepositoryHTTP] Not authenticated.");

    const res = await fetch(url, {
      method: "HEAD",
      headers: { Authorization: `Bearer ${token}` },
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

  // Count
  async count(filter: BrandFilter = {}): Promise<number> {
    const params = new URLSearchParams();
    if (filter.companyId) params.set("companyId", filter.companyId);
    if (filter.managerId) params.set("managerId", filter.managerId);
    if (typeof filter.isActive === "boolean")
      params.set("isActive", String(filter.isActive));
    if (filter.walletAddress) params.set("walletAddress", filter.walletAddress);
    if (typeof filter.deleted === "boolean")
      params.set("deleted", String(filter.deleted));

    const url =
      params.toString().length > 0
        ? `${this.baseUrl}/count?${params.toString()}`
        : `${this.baseUrl}/count`;

    const result = await authed<{ count: number }>(url, { method: "GET" });
    return result.count;
  }

  // Update (partial)
  async update(id: string, patch: BrandPatch): Promise<Brand> {
    const url = `${this.baseUrl}/${encodeURIComponent(id)}`;
    return authed<Brand>(url, {
      method: "PATCH",
      body: JSON.stringify(patch),
    });
  }

  // Delete
  async delete(id: string): Promise<void> {
    const url = `${this.baseUrl}/${encodeURIComponent(id)}`;
    await authed<void>(url, { method: "DELETE" });
  }

  // List
  async list(options: {
    filter?: BrandFilter;
    sort?: BrandSort;
    page?: PageParams["page"];
    perPage?: PageParams["perPage"];
  } = {}): Promise<PageResult<Brand>> {
    const { filter = {}, sort = {}, page, perPage } = options;
    const params = new URLSearchParams();

    if (filter.companyId) params.set("companyId", filter.companyId);
    if (filter.managerId) params.set("managerId", filter.managerId);
    if (typeof filter.isActive === "boolean")
      params.set("isActive", String(filter.isActive));
    if (filter.walletAddress) params.set("walletAddress", filter.walletAddress);
    if (typeof filter.deleted === "boolean")
      params.set("deleted", String(filter.deleted));

    if (sort.column) params.set("column", sort.column);
    if (sort.order) params.set("order", sort.order);

    if (page != null) params.set("page", String(page));
    if (perPage != null) params.set("perPage", String(perPage));

    const qs = params.toString();
    const url = qs ? `${this.baseUrl}?${qs}` : this.baseUrl;

    // ★ Go 側が Items / TotalCount ... のように先頭大文字で返してくるケースも考慮して正規化
    const raw = (await authed<any>(url, { method: "GET" })) ?? {};

    const rawItems = (raw.items ?? raw.Items ?? []) as any[];

    const normalizedItems: Brand[] = rawItems.map((b) => ({
      id: b.id ?? b.ID ?? "",
      companyId: b.companyId ?? b.CompanyID ?? "",
      name: b.name ?? b.Name ?? "",
      description: b.description ?? b.Description ?? "",
      websiteUrl: b.websiteUrl ?? b.URL ?? b.Url ?? "",
      isActive: Boolean(b.isActive ?? b.IsActive ?? true),
      manager: b.manager ?? b.Manager ?? b.managerId ?? b.ManagerID ?? undefined,
      walletAddress: b.walletAddress ?? b.WalletAddress ?? "",
      createdAt: b.createdAt ?? b.CreatedAt ?? "",
      createdBy: b.createdBy ?? b.CreatedBy ?? undefined,
      updatedAt: b.updatedAt ?? b.UpdatedAt ?? undefined,
      updatedBy: b.updatedBy ?? b.UpdatedBy ?? undefined,
      deletedAt: b.deletedAt ?? b.DeletedAt ?? undefined,
      deletedBy: b.deletedBy ?? b.DeletedBy ?? undefined,
    }));

    const totalCount = Number(
      raw.totalCount ?? raw.TotalCount ?? normalizedItems.length ?? 0,
    );
    const totalPages = Number(raw.totalPages ?? raw.TotalPages ?? 1);
    const pageNum = Number(raw.page ?? raw.Page ?? page ?? 1);
    const perPageNum = Number(
      raw.perPage ?? raw.PerPage ?? perPage ?? normalizedItems.length ?? 0,
    );

    return {
      items: normalizedItems,
      totalCount,
      totalPages,
      page: pageNum,
      perPage: perPageNum,
    };
  }

  // Save (Upsert)
  async save(brand: Brand): Promise<Brand> {
    const trimmedId = brand.id?.trim();
    if (!trimmedId) {
      return this.create({
        ...brand,
        id: "",
      } as Omit<Brand, "createdAt" | "updatedAt">);
    }
    const url = `${this.baseUrl}/${encodeURIComponent(trimmedId)}`;
    return authed<Brand>(url, {
      method: "PUT",
      body: JSON.stringify(brand),
    });
  }

  // Reset (dev)
  async reset(): Promise<void> {
    const url = `${this.baseUrl}/reset`;
    await authed<void>(url, { method: "POST" });
  }
}

export const brandRepositoryHTTP = new BrandRepositoryHTTP();

/**
 * brandId → brandName 変換ヘルパー
 * backend の brand.Service.GetNameByID と同等の役割
 */
export async function fetchBrandNameById(brandId: string): Promise<string> {
  const id = brandId?.trim();
  if (!id) return "";

  try {
    const b = await brandRepositoryHTTP.getById(id);
    return (b.name ?? "").trim(); // formatBrandName 相当
  } catch (err) {
    console.warn("[fetchBrandNameById] failed to get brand name", {
      brandId: id,
      err,
    });
    // 失敗時は ID をそのまま表示するか、空文字にするかは運用ポリシー次第
    return id;
  }
}
