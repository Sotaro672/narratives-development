// frontend/console/member/src/infrastructure/http/memberRepositoryHTTP.ts
/// <reference types="vite/client" />

import type {
  MemberRepository,
  MemberFilter,
  MemberSort,
} from "../../domain/repository/memberRepository";
import type {
  Page,
  PageResult,
  CursorPage,
  CursorPageResult,
  SaveOptions,
} from "../../../../shell/src/shared/types/common/common";
import type { Member, MemberPatch } from "../../domain/entity/member";

// ✅ shared http (shell)
import { buildConsoleUrl } from "../../../../shell/src/shared/http/apiBase";
import {
  getAuthHeaders,
  getAuthJsonHeaders,
} from "../../../../shell/src/shared/http/authHeaders";
import { fetchJSON } from "../../../../shell/src/shared/http/fetchJSON";
import { withQuery } from "../../../../shell/src/shared/http/queryString";

export class MemberRepositoryHTTP implements MemberRepository {
  async getById(id: string): Promise<Member | null> {
    const headers = await getAuthHeaders();
    const url = buildConsoleUrl(`/members/${encodeURIComponent(id)}`);

    const res = await fetch(url, { headers });
    if (res.status === 404) return null;

    const ct = res.headers.get("content-type") ?? "";
    if (!ct.includes("application/json")) {
      const text = await res.text().catch(() => "");
      throw new Error(`Unexpected content-type: ${ct}\n${text.slice(0, 200)}`);
    }
    if (!res.ok) throw new Error(await res.text().catch(() => `HTTP ${res.status}`));

    return (await res.json()) as Member;
  }

  async list(page: Page, filter?: MemberFilter): Promise<PageResult<Member>> {
    const headers = await getAuthHeaders();

    const pageNumber = page.number && page.number > 0 ? page.number : 1;
    const perPage = page.perPage && page.perPage > 0 ? page.perPage : 50;

    const url = withQuery(buildConsoleUrl("/members"), {
      q: filter?.searchQuery,
      brandIds: filter?.brandIds,
      status: filter?.status,
      page: pageNumber,
      perPage,
      sort: "updatedAt",
      order: "desc",
    });

    const data = await fetchJSON<unknown>(url, { headers });

    // backend が配列だけ返すケースにも対応
    if (Array.isArray(data)) {
      return {
        items: data as Member[],
        totalCount: (data as Member[]).length,
        page: pageNumber,
        perPage,
        totalPages: 1,
      };
    }

    return data as PageResult<Member>;
  }

  async create(member: Member): Promise<Member> {
    const headers = await getAuthJsonHeaders();
    const url = buildConsoleUrl("/members");

    return await fetchJSON<Member>(url, {
      method: "POST",
      headers,
      body: JSON.stringify(member),
    });
  }

  async update(id: string, patch: MemberPatch, _opts?: SaveOptions): Promise<Member> {
    throw new Error("MemberRepositoryHTTP.update: not supported by current backend API");
  }

  async delete(id: string): Promise<void> {
    throw new Error("MemberRepositoryHTTP.delete: not supported by current backend API");
  }

  async listByCursor(
    filter: MemberFilter,
    _sort: MemberSort,
    cursorPage: CursorPage,
  ): Promise<CursorPageResult<Member>> {
    const limit = cursorPage.limit && cursorPage.limit > 0 ? cursorPage.limit : 50;

    const page: Page = { number: 1, perPage: limit, totalPages: 1 };
    const res = await this.list(page, filter);

    return {
      items: res.items,
      nextCursor: null,
      prevCursor: undefined,
      hasNext: false,
      hasPrev: false,
    };
  }

  async getByEmail(email: string): Promise<Member | null> {
    const res = await this.list(
      { number: 1, perPage: 50, totalPages: 1 },
      { searchQuery: email },
    );

    const hit = res.items.find(
      (m) => (m.email ?? "").toLowerCase() === email.trim().toLowerCase(),
    );
    return hit ?? null;
  }

  async exists(id: string): Promise<boolean> {
    return (await this.getById(id)) != null;
  }

  async count(filter: MemberFilter): Promise<number> {
    const res = await this.list({ number: 1, perPage: 100, totalPages: 1 }, filter);
    return res.totalCount ?? res.items.length;
  }

  async save(member: Member, opts?: SaveOptions): Promise<Member> {
    if (opts?.mode === "update" || opts?.ifExists) {
      throw new Error("MemberRepositoryHTTP.save(update): not supported by current backend API");
    }
    return this.create(member);
  }

  async reset(): Promise<void> {
    throw new Error("MemberRepositoryHTTP.reset: not supported by current backend API");
  }
}

/**
 * ID → 担当者名 解決
 */
export async function fetchMemberDisplayNameById(memberId: string): Promise<string> {
  const trimmed = memberId.trim();
  if (!trimmed) return "-";

  try {
    const headers = await getAuthHeaders();
    const url = buildConsoleUrl(`/members/${encodeURIComponent(trimmed)}`);

    const res = await fetch(url, { headers });
    if (res.status === 404) return trimmed;

    const ct = res.headers.get("content-type") ?? "";
    if (!ct.includes("application/json")) return trimmed;

    const m = (await res.json()) as Member;
    const lastName = (m as any).lastName?.trim?.() ?? "";
    const firstName = (m as any).firstName?.trim?.() ?? "";
    const fullNameField = (m as any).fullName?.trim?.() ?? "";
    const email = (m as any).email?.trim?.() ?? "";

    const nameParts = [lastName, firstName].filter(Boolean);
    const nameFromLF = nameParts.join(" ");

    return nameFromLF || fullNameField || email || trimmed;
  } catch {
    return trimmed;
  }
}
