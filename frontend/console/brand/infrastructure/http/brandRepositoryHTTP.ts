//frontend\console\brand\src\infrastructure\http\brandRepositoryHTTP.ts
import type { Brand, BrandPatch } from "../../domain/entity/brand";
import { getConsoleApiBase } from "../../../../shell/src/shared/http/apiBase";
import {
  getAuthHeadersOrThrow,
  getAuthJsonHeadersOrThrow,
} from "../../../../shell/src/shared/http/authHeaders";

export interface BrandFilter {
  companyId?: string;
  managerId?: string;
  isActive?: boolean;
  walletAddress?: string;
  deleted?: boolean;
}

export interface BrandSort {
  column?: "name" | "is_active" | "updated_at" | "created_at";
  order?: "asc" | "desc";
}

export interface PageParams {
  page?: number;
  perPage?: number;
}

export interface PageResult<T> {
  items: T[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
}

export type BrandUploadTarget = "brandIcon" | "brandBackgroundImage";

export interface BrandSignedUploadUrlRequest {
  brandId: string;
  target: BrandUploadTarget;
  fileName: string;
  contentType: string;
}

export interface BrandSignedUploadUrlResponse {
  uploadUrl: string;
  publicUrl?: string;
  objectPath?: string;
  method?: "PUT" | "POST";
  headers?: Record<string, string>;
}

const BASE_URL = `${getConsoleApiBase()}/brands`;

async function httpRequest<T>(
  input: string,
  init: RequestInit = {},
): Promise<T> {
  const res = await fetch(input, init);

  if (res.status === 204) return undefined as unknown as T;

  const text = await res.text().catch(() => "");

  if (!res.ok) {
    throw new Error(
      `[BrandRepositoryHTTP] ${res.status} ${res.statusText} :: ${text?.slice(0, 300)}`,
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

async function authed<T>(
  input: string,
  init: RequestInit = {},
  opts?: { json?: boolean },
): Promise<T> {
  const headers = opts?.json
    ? await getAuthJsonHeadersOrThrow()
    : await getAuthHeadersOrThrow();

  return httpRequest<T>(input, {
    ...init,
    headers: {
      ...headers,
      ...(init.headers ?? {}),
    },
  });
}

export class BrandRepositoryHTTP {
  private readonly baseUrl: string;

  constructor(baseUrl: string = BASE_URL) {
    this.baseUrl = String(baseUrl ?? "").replace(/\/+$/g, "");
    if (!this.baseUrl) {
      throw new Error("[BrandRepositoryHTTP] baseUrl is empty.");
    }
  }

  async create(input: Omit<Brand, "createdAt" | "updatedAt">): Promise<Brand> {
    return authed<Brand>(
      this.baseUrl,
      {
        method: "POST",
        body: JSON.stringify(input),
      },
      { json: true },
    );
  }

  async getById(id: string): Promise<Brand> {
    const url = `${this.baseUrl}/${encodeURIComponent(id)}`;
    return authed<Brand>(url, { method: "GET" });
  }

  async exists(id: string): Promise<boolean> {
    const url = `${this.baseUrl}/${encodeURIComponent(id)}`;
    const headers = await getAuthHeadersOrThrow();

    const res = await fetch(url, {
      method: "HEAD",
      headers,
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

    const result = await authed<{ count: number }>(url, { method: "GET" });
    return result.count;
  }

  async update(id: string, patch: BrandPatch): Promise<Brand> {
    const url = `${this.baseUrl}/${encodeURIComponent(id)}`;
    return authed<Brand>(
      url,
      {
        method: "PATCH",
        body: JSON.stringify(patch),
      },
      { json: true },
    );
  }

  async delete(id: string): Promise<void> {
    const url = `${this.baseUrl}/${encodeURIComponent(id)}`;
    await authed<void>(url, { method: "DELETE" });
  }

  async activate(id: string): Promise<Brand> {
    const url = `${this.baseUrl}/${encodeURIComponent(id)}/activate`;
    return authed<Brand>(
      url,
      {
        method: "POST",
      },
      { json: true },
    );
  }

  async deactivate(id: string): Promise<Brand> {
    const url = `${this.baseUrl}/${encodeURIComponent(id)}/deactivate`;
    return authed<Brand>(
      url,
      {
        method: "POST",
      },
      { json: true },
    );
  }

  async list(
    options: {
      filter?: BrandFilter;
      sort?: BrandSort;
      page?: PageParams["page"];
      perPage?: PageParams["perPage"];
    } = {},
  ): Promise<PageResult<Brand>> {
    const { filter = {}, sort = {}, page, perPage } = options;
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

    if (sort.column) params.set("column", sort.column);
    if (sort.order) params.set("order", sort.order);

    if (page != null) params.set("page", String(page));
    if (perPage != null) params.set("perPage", String(perPage));

    const qs = params.toString();
    const url = qs ? `${this.baseUrl}?${qs}` : this.baseUrl;

    const raw = (await authed<any>(url, { method: "GET" })) ?? {};
    const items = (raw.items ?? []) as any[];

    const normalizedItems: Brand[] = items.map((b) => ({
      id: b.id ?? "",
      companyId: b.companyId ?? "",
      name: b.name ?? "",
      description: b.description ?? "",
      websiteUrl: b.websiteUrl ?? "",
      brandIcon: b.brandIcon ?? "",
      brandBackgroundImage: b.brandBackgroundImage ?? "",
      isActive: Boolean(b.isActive ?? false),
      managerId: b.managerId ?? null,
      memberName: b.memberName ?? null,
      walletAddress: b.walletAddress ?? "",
      createdAt: b.createdAt ?? "",
      createdBy: b.createdBy ?? null,
      updatedAt: b.updatedAt ?? null,
      updatedBy: b.updatedBy ?? null,
      deletedAt: b.deletedAt ?? null,
      deletedBy: b.deletedBy ?? null,
    }));

    return {
      items: normalizedItems,
      totalCount: Number(raw.totalCount ?? 0),
      totalPages: Number(raw.totalPages ?? 1),
      page: Number(raw.page ?? page ?? 1),
      perPage: Number(raw.perPage ?? perPage ?? normalizedItems.length ?? 0),
    };
  }

  async save(brand: Brand): Promise<Brand> {
    const id = brand.id;

    if (!id) {
      return this.create({
        ...brand,
        id: "",
      } as Omit<Brand, "createdAt" | "updatedAt">);
    }

    const url = `${this.baseUrl}/${encodeURIComponent(id)}`;
    return authed<Brand>(
      url,
      {
        method: "PUT",
        body: JSON.stringify(brand),
      },
      { json: true },
    );
  }

  async getSignedUploadUrl(
    input: BrandSignedUploadUrlRequest,
  ): Promise<BrandSignedUploadUrlResponse> {
    if (!input.brandId) {
      throw new Error(
        "[BrandRepositoryHTTP] brandId is required before requesting upload URL.",
      );
    }

    return authed<BrandSignedUploadUrlResponse>(
      `${this.baseUrl}/upload-url`,
      {
        method: "POST",
        body: JSON.stringify({
          brandId: input.brandId,
          target: input.target,
          fileName: input.fileName,
          contentType: input.contentType,
        }),
      },
      { json: true },
    );
  }

  async uploadFileToSignedUrl(
    file: File,
    signed: BrandSignedUploadUrlResponse,
  ): Promise<void> {
    const method = signed.method ?? "PUT";
    const headers: Record<string, string> = {
      ...(signed.headers ?? {}),
    };

    if (!headers["Content-Type"] && method === "PUT") {
      headers["Content-Type"] = file.type || "application/octet-stream";
    }

    const res = await fetch(signed.uploadUrl, {
      method,
      headers,
      body: file,
    });

    if (!res.ok) {
      const text = await res.text().catch(() => "");
      throw new Error(
        `[BrandRepositoryHTTP] signed upload failed: ${res.status} ${res.statusText} ${text.slice(0, 300)}`,
      );
    }
  }

  async uploadBrandAsset(params: {
    file: File;
    target: BrandUploadTarget;
    brandId: string;
  }): Promise<{ publicUrl?: string; objectPath?: string }> {
    if (!params.brandId) {
      throw new Error(
        "[BrandRepositoryHTTP] brandId is required before uploading brand asset.",
      );
    }

    const signed = await this.getSignedUploadUrl({
      brandId: params.brandId,
      target: params.target,
      fileName: params.file.name,
      contentType: params.file.type || "application/octet-stream",
    });

    await this.uploadFileToSignedUrl(params.file, signed);

    return {
      publicUrl: signed.publicUrl,
      objectPath: signed.objectPath,
    };
  }
}

export const brandRepositoryHTTP = new BrandRepositoryHTTP();

export async function fetchBrandNameById(brandId: string): Promise<string> {
  const id = brandId ?? "";
  if (!id) return "";

  try {
    const b = await brandRepositoryHTTP.getById(id);
    return b.name ?? "";
  } catch (err) {
    console.warn("[fetchBrandNameById] failed to get brand name", {
      brandId: id,
      err,
    });
    return id;
  }
}

export async function fetchBrandsForCurrentCompany(params?: {
  companyId?: string;
  perPage?: number;
}): Promise<{ id: string; name: string }[]> {
  const perPage = params?.perPage ?? 200;
  const companyId = String(params?.companyId ?? "");

  try {
    const res = await brandRepositoryHTTP.list({
      filter: companyId ? { companyId } : {},
      perPage,
      page: 1,
    });

    return (res.items ?? []).map((b) => ({
      id: String((b as any)?.id ?? ""),
      name: String((b as any)?.name ?? ""),
    }));
  } catch (err) {
    console.warn("[fetchBrandsForCurrentCompany] failed to list brands", {
      companyId,
      err,
    });
    return [];
  }
}