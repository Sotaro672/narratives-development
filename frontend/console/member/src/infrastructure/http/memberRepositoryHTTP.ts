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

import { getAuthHeaders } from "../../../../shell/src/auth/application/authService";

// ===========================
// BACKEND BASE URL
// ===========================
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/,
    "",
  ) ?? "";

// ★ Cloud Run の URL をフォールバックに追加
const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

const API_BASE = ENV_BASE || FALLBACK_BASE;

function toQuery(params: Record<string, any>) {
  const sp = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v === undefined || v === null || v === "") return;
    if (Array.isArray(v)) {
      v.forEach((x) => sp.append(k, String(x)));
    } else {
      sp.set(k, String(v));
    }
  });
  return sp.toString();
}

async function fetchJSON(input: RequestInfo, init?: RequestInit) {
  const res = await fetch(input, init);
  const ct = res.headers.get("content-type") ?? "";
  if (!ct.includes("application/json")) {
    const text = await res.text().catch(() => "");
    throw new Error(`Unexpected content-type: ${ct}\n${text.slice(0, 200)}`);
  }
  if (!res.ok) {
    throw new Error(await res.text().catch(() => `HTTP ${res.status}`));
  }
  return res.json();
}

export class MemberRepositoryHTTP implements MemberRepository {
  async getById(id: string): Promise<Member | null> {
    const headers = await getAuthHeaders();
    const url = `${API_BASE}/members/${encodeURIComponent(id)}`;
    const res = await fetch(url, { headers });
    if (res.status === 404) return null;

    const ct = res.headers.get("content-type") ?? "";
    if (!ct.includes("application/json")) {
      throw new Error(`Unexpected content-type: ${ct}`);
    }
    if (!res.ok) throw new Error(await res.text());
    return (await res.json()) as Member;
  }

  async list(page: Page, filter?: MemberFilter): Promise<PageResult<Member>> {
    const headers = await getAuthHeaders();

    const pageNumber = page.number && page.number > 0 ? page.number : 1;
    const perPage = page.perPage && page.perPage > 0 ? page.perPage : 50;

    const qs = toQuery({
      q: filter?.searchQuery,
      brandIds: filter?.brandIds,
      status: filter?.status,
      page: pageNumber,
      perPage,
      sort: "updatedAt",
      order: "desc",
    });

    const data = await fetchJSON(`${API_BASE}/members?${qs}`, { headers });

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
    const headers = {
      ...(await getAuthHeaders()),
      "Content-Type": "application/json",
    };
    return (await fetchJSON(`${API_BASE}/members`, {
      method: "POST",
      headers,
      body: JSON.stringify(member),
    })) as Member;
  }

  async update(id: string, patch: MemberPatch, _opts?: SaveOptions): Promise<Member> {
    throw new Error(
      "MemberRepositoryHTTP.update: not supported by current backend API",
    );
  }

  async delete(id: string): Promise<void> {
    throw new Error(
      "MemberRepositoryHTTP.delete: not supported by current backend API",
    );
  }

  async listByCursor(
    filter: MemberFilter,
    _sort: MemberSort,
    cursorPage: CursorPage,
  ): Promise<CursorPageResult<Member>> {
    const limit =
      cursorPage.limit && cursorPage.limit > 0 ? cursorPage.limit : 50;

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
    const res = await this.list(
      { number: 1, perPage: 100, totalPages: 1 },
      filter,
    );
    return res.totalCount ?? res.items.length;
  }

  async save(member: Member, opts?: SaveOptions): Promise<Member> {
    if (opts?.mode === "update" || opts?.ifExists) {
      throw new Error(
        "MemberRepositoryHTTP.save(update): not supported by current backend API",
      );
    }
    if (opts?.mode === "create" || opts?.ifNotExists) {
      return this.create(member);
    }
    return this.create(member);
  }

  async reset(): Promise<void> {
    throw new Error(
      "MemberRepositoryHTTP.reset: not supported by current backend API",
    );
  }
}

/**
 * assigneeId から画面表示用の担当者名を取得するヘルパー
 * - 姓名があれば「姓 名」
 * - なければ fullName
 * - それもなければ email
 * - それもなければ元の ID
 */
export async function fetchMemberDisplayNameById(
  memberId: string,
): Promise<string> {
  const trimmed = memberId.trim();
  if (!trimmed) return "-";

  try {
    const headers = await getAuthHeaders();
    const url = `${API_BASE}/members/${encodeURIComponent(trimmed)}`;

    console.log(
      "[memberRepositoryHTTP] fetchMemberDisplayNameById request",
      url,
    );

    const res = await fetch(url, { headers });

    if (res.status === 404) {
      console.warn(
        "[memberRepositoryHTTP] fetchMemberDisplayNameById: member not found",
        trimmed,
      );
      return trimmed;
    }

    const ct = res.headers.get("content-type") ?? "";
    if (!ct.includes("application/json")) {
      console.warn(
        "[memberRepositoryHTTP] fetchMemberDisplayNameById: unexpected content-type",
        ct,
      );
      return trimmed;
    }

    const m = (await res.json()) as Member;

    const lastName = (m as any).lastName?.trim?.() ?? "";
    const firstName = (m as any).firstName?.trim?.() ?? "";
    const fullNameField = (m as any).fullName?.trim?.() ?? "";

    const nameParts = [lastName, firstName].filter(Boolean);
    const nameFromLF = nameParts.join(" ");

    const email = (m as any).email?.trim?.() ?? "";

    const display =
      nameFromLF || fullNameField || email || trimmed;

    console.log(
      "[memberRepositoryHTTP] fetchMemberDisplayNameById result",
      { memberId: trimmed, display },
    );

    return display;
  } catch (e) {
    console.error(
      "[memberRepositoryHTTP] fetchMemberDisplayNameById error",
      e,
    );
    return trimmed;
  }
}
