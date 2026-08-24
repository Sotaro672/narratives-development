// frontend/console/shell/src/features/brand/infrastructure/http/brandRepositoryHTTP.ts

import type { Brand, BrandPatch } from "../../../../shared/types/brand";
import type { PageParams, PageResult } from "../../../../shared/types/common/common";
import { buildConsoleUrl } from "../../../../shared/http/apiBase";
import { getAuthHeaders } from "../../../../shared/http/authHeaders";

/**
 * POST /brands のリクエストボディ。
 *
 * Brand は必ず Company 配下の Account に接続して作成するため、
 * accountId は必須。
 *
 * 新規BrandはBackend側で必ず isActive=true として作成するため、
 * isActiveは作成リクエストへ含めない。
 *
 * 次の項目もBackend側で生成または後から設定されるため含めない。
 * - id
 * - memberName
 * - walletAddress
 * - createdAt
 * - updatedAt
 * - updatedBy
 * - deletedAt
 * - deletedBy
 */
export interface CreateBrandInput {
  companyId: string;
  accountId: string;
  name: string;
  description: string;
  websiteUrl?: string;
  brandIcon?: string;
  brandBackgroundImage?: string;
  managerId?: string | null;
  createdBy?: string | null;
}

/**
 * Brand に接続する Account を変更するための入力。
 *
 * 1つの Account を複数 Brand が共有することを許容する。
 * Account の Company 所有権検証はBackend側で行う。
 */
export interface UpdateBrandAccountInput {
  accountId: string;
}

const BASE_URL = buildConsoleUrl("/brands");

async function httpRequest<T>(input: string, init: RequestInit = {}): Promise<T> {
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

  const looksLikeHTML = /^\s*<!doctype html>|^\s*<html/i.test(text);

  if (looksLikeHTML) {
    throw new Error(
      "[BrandRepositoryHTTP] response is not JSON (HTML received). " +
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
  const authHeaders = await getAuthHeaders();

  return httpRequest<T>(input, {
    ...init,
    headers: {
      ...authHeaders,
      ...(opts?.json ? { "Content-Type": "application/json" } : {}),
      ...(init.headers ?? {}),
    },
  });
}

export class BrandRepositoryHTTP {
  private readonly baseUrl: string;

  constructor(baseUrl: string = BASE_URL) {
    this.baseUrl = baseUrl.replace(/\/+$/g, "");

    if (!this.baseUrl) {
      throw new Error("[BrandRepositoryHTTP] baseUrl is empty.");
    }
  }

  async create(input: CreateBrandInput): Promise<Brand> {
    const accountId = input.accountId.trim();

    if (!accountId) {
      throw new Error("[BrandRepositoryHTTP] accountId is required.");
    }

    return authed<Brand>(
      this.baseUrl,
      {
        method: "POST",
        body: JSON.stringify({
          ...input,
          accountId,
        }),
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

  /**
   * Brand に接続している Account を変更します。
   *
   * Backend側で以下を検証します。
   * - Account が存在すること
   * - Brand と Account が同一 Company に属すること
   * - Account が deleted ではないこと
   *
   * 同一 Account を複数 Brand が参照することは許容します。
   */
  async updateAccount(
    id: string,
    input: UpdateBrandAccountInput,
  ): Promise<Brand> {
    const accountId = input.accountId.trim();

    if (!accountId) {
      throw new Error("[BrandRepositoryHTTP] accountId is required.");
    }

    const url = `${this.baseUrl}/${encodeURIComponent(id)}`;

    return authed<Brand>(
      url,
      {
        method: "PATCH",
        body: JSON.stringify({
          accountId,
        }),
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

  async list(options: PageParams = {}): Promise<PageResult<Brand>> {
    const { page, perPage } = options;
    const params = new URLSearchParams();

    if (page != null) {
      params.set("page", String(page));
    }

    if (perPage != null) {
      params.set("perPage", String(perPage));
    }

    const query = params.toString();
    const url = query ? `${this.baseUrl}?${query}` : this.baseUrl;

    return authed<PageResult<Brand>>(url, {
      method: "GET",
    });
  }
}

export const brandRepositoryHTTP = new BrandRepositoryHTTP();

export async function fetchBrandNameById(brandId: string): Promise<string> {
  if (!brandId) {
    return "";
  }

  try {
    const brand = await brandRepositoryHTTP.getById(brandId);
    return brand.name;
  } catch {
    return brandId;
  }
}

export async function fetchBrandsForCurrentCompany(
  params?: { perPage?: number },
): Promise<{ id: string; name: string }[]> {
  const perPage = params?.perPage ?? 200;

  try {
    const result = await brandRepositoryHTTP.list({
      page: 1,
      perPage,
    });

    return result.items.map((brand) => ({
      id: brand.id,
      name: brand.name,
    }));
  } catch {
    return [];
  }
}