// frontend/console/shell/src/features/brand/infrastructure/http/brandRepositoryHTTP.ts

import type {
  Brand,
  BrandPatch,
} from "../../../../shared/types/brand";
import { getConsoleApiBase } from "../../../../shared/http/apiBase";
import {
  getAuthHeadersOrThrow,
  getAuthJsonHeadersOrThrow,
} from "../../../../shared/http/authHeaders";

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

const BASE_URL = `${getConsoleApiBase()}/brands`;

async function httpRequest<T>(
  input: string,
  init: RequestInit = {},
): Promise<T> {
  const res = await fetch(input, init);

  if (res.status === 204) {
    return undefined as unknown as T;
  }

  const text = await res.text().catch(() => "");

  if (!res.ok) {
    throw new Error(
      `[BrandRepositoryHTTP] ${res.status} ${res.statusText} :: ${text.slice(0, 300)}`,
    );
  }

  const looksLikeHTML =
    /^\s*<!doctype html>|^\s*<html/i.test(text);

  if (looksLikeHTML) {
    throw new Error(
      `[BrandRepositoryHTTP] response is not JSON (HTML received). ` +
        `BASE_URL の設定を確認してください。received head: ${text.slice(0, 120)}`,
    );
  }

  try {
    return text
      ? (JSON.parse(text) as T)
      : (undefined as unknown as T);
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
    this.baseUrl = String(baseUrl ?? "").replace(
      /\/+$/g,
      "",
    );

    if (!this.baseUrl) {
      throw new Error(
        "[BrandRepositoryHTTP] baseUrl is empty.",
      );
    }
  }

  async create(
    input: Omit<Brand, "createdAt" | "updatedAt">,
  ): Promise<Brand> {
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

    return authed<Brand>(url, {
      method: "GET",
    });
  }

  async update(
    id: string,
    patch: BrandPatch,
  ): Promise<Brand> {
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

    await authed<void>(url, {
      method: "DELETE",
    });
  }

  async list(
    options: PageParams = {},
  ): Promise<PageResult<Brand>> {
    const { page, perPage } = options;
    const params = new URLSearchParams();

    if (page != null) {
      params.set("page", String(page));
    }

    if (perPage != null) {
      params.set("perPage", String(perPage));
    }

    const query = params.toString();
    const url = query
      ? `${this.baseUrl}?${query}`
      : this.baseUrl;

    const raw =
      (await authed<any>(url, {
        method: "GET",
      })) ?? {};

    const items = (raw.items ?? []) as any[];

    const normalizedItems: Brand[] = items.map(
      (brand) => ({
        id: brand.id ?? "",
        companyId: brand.companyId ?? "",
        name: brand.name ?? "",
        description: brand.description ?? "",
        websiteUrl: brand.websiteUrl ?? "",
        brandIcon: brand.brandIcon ?? "",
        brandBackgroundImage:
          brand.brandBackgroundImage ?? "",
        isActive: Boolean(brand.isActive ?? false),
        managerId: brand.managerId ?? null,
        memberName: brand.memberName ?? null,
        walletAddress: brand.walletAddress ?? "",
        createdAt: brand.createdAt ?? "",
        createdBy: brand.createdBy ?? null,
        updatedAt: brand.updatedAt ?? null,
        updatedBy: brand.updatedBy ?? null,
        deletedAt: brand.deletedAt ?? null,
        deletedBy: brand.deletedBy ?? null,
      }),
    );

    return {
      items: normalizedItems,
      totalCount: Number(raw.totalCount ?? 0),
      totalPages: Number(raw.totalPages ?? 1),
      page: Number(raw.page ?? page ?? 1),
      perPage: Number(
        raw.perPage ??
          perPage ??
          normalizedItems.length,
      ),
    };
  }
}

export const brandRepositoryHTTP =
  new BrandRepositoryHTTP();

export async function fetchBrandNameById(
  brandId: string,
): Promise<string> {
  const id = brandId ?? "";

  if (!id) {
    return "";
  }

  try {
    const brand =
      await brandRepositoryHTTP.getById(id);

    return brand.name ?? "";
  } catch {
    return id;
  }
}

export async function fetchBrandsForCurrentCompany(
  params?: {
    perPage?: number;
  },
): Promise<{ id: string; name: string }[]> {
  const perPage = params?.perPage ?? 200;

  try {
    const result = await brandRepositoryHTTP.list({
      page: 1,
      perPage,
    });

    return result.items.map((brand) => ({
      id: String(brand.id ?? ""),
      name: String(brand.name ?? ""),
    }));
  } catch {
    return [];
  }
}