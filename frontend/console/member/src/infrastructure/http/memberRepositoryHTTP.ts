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

// ★ Hookは使わず、関数APIからIDトークンを取得
import { getAuthHeaders } from "../../../../shell/src/auth/application/authService";

const API_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(/\/+$/, "") ?? "";

function toQuery(params: Record<string, any>) {
  const sp = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v === undefined || v === null || v === "") return;
    if (Array.isArray(v)) v.forEach((x) => sp.append(k, String(x)));
    else sp.set(k, String(v));
  });
  return sp.toString();
}

async function fetchJSON(input: RequestInfo, init?: RequestInit) {
  const res = await fetch(input, init);
  const ct = res.headers.get("content-type") ?? "";
  if (!ct.includes("application/json")) {
    // SPAのindex.htmlなどを拾った時にすぐ原因がわかるように
    const text = await res.text().catch(() => "");
    throw new Error(`Unexpected content-type: ${ct}\n${text.slice(0, 200)}`);
  }
  if (!res.ok) {
    throw new Error(await res.text().catch(() => `HTTP ${res.status}`));
  }
  return res.json();
}

/**
 * HTTP 実装の MemberRepository
 * - サーバ側が用意しているエンドポイントに合わせて呼び分けます
 * - まだサーバに存在しないAPIは「未サポート」として例外を投げます（型は満たす）
 */
export class MemberRepositoryHTTP implements MemberRepository {
  // ===== CRUD / List =====
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
    // companyId は送らない（Usecase が ctx.companyId で強制上書き）
    const qs = toQuery({
      q: filter?.searchQuery,
      brandIds: filter?.brandIds,
      status: filter?.status,
      page: (page.offset ?? 0) / (page.limit ?? 50) + 1,
      perPage: page.limit ?? 50,
      sort: "updatedAt",
      order: "desc",
    });
    const data = await fetchJSON(`${API_BASE}/members?${qs}`, { headers });

    // ハンドラが配列のみ返す実装にも対応
    if (Array.isArray(data)) {
      return {
        items: data as Member[],
        totalCount: (data as Member[]).length,
        page: { limit: page.limit ?? 50, offset: page.offset ?? 0 },
      };
    }
    return data as PageResult<Member>;
  }

  async create(member: Member): Promise<Member> {
    const headers = { ...(await getAuthHeaders()), "Content-Type": "application/json" };
    return (await fetchJSON(`${API_BASE}/members`, {
      method: "POST",
      headers,
      body: JSON.stringify(member),
    })) as Member;
  }

  // ===== 追加メソッド（型を満たすために実装 or 例外） =====

  async update(id: string, patch: MemberPatch, _opts?: SaveOptions): Promise<Member> {
    // サーバのHTTPハンドラに PUT/PATCH が無ければ未サポート
    // 実装を追加したらこちらを有効化:
    // const headers = { ...(await getAuthHeaders()), "Content-Type": "application/json" };
    // return (await fetchJSON(`${API_BASE}/members/${encodeURIComponent(id)}`, {
    //   method: "PATCH",
    //   headers,
    //   body: JSON.stringify(patch),
    // })) as Member;
    throw new Error("MemberRepositoryHTTP.update: not supported by current backend API");
  }

  async delete(id: string): Promise<void> {
    // 同上：エンドポイントが無ければ未サポート
    // const headers = await getAuthHeaders();
    // await fetchJSON(`${API_BASE}/members/${encodeURIComponent(id)}`, { method: "DELETE", headers });
    throw new Error("MemberRepositoryHTTP.delete: not supported by current backend API");
  }

  async listByCursor(
    filter: MemberFilter,
    sort: MemberSort,
    cursorPage: CursorPage
  ): Promise<CursorPageResult<Member>> {
    // サーバ側に Cursor API が無い前提では、page/limit にフォールバック
    const limit = cursorPage.limit > 0 ? cursorPage.limit : 50;
    const page: Page = { limit, offset: 0 };
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
    // サーバの list が email クエリを直接サポートしていないため、
    // 暫定で searchQuery に寄せる（完全一致ではない点に注意）
    const res = await this.list({ limit: 50, offset: 0 }, { searchQuery: email });
    const hit = res.items.find((m) => (m.email ?? "").toLowerCase() === email.trim().toLowerCase());
    return hit ?? null;
  }

  async exists(id: string): Promise<boolean> {
    return (await this.getById(id)) != null;
  }

  async count(filter: MemberFilter): Promise<number> {
    // 専用の /members/count が無い前提では list の件数で代替
    const res = await this.list({ limit: 100, offset: 0 }, filter);
    return res.totalCount ?? res.items.length;
  }

  async save(member: Member, opts?: SaveOptions): Promise<Member> {
    // Backendに upsert API が無い前提では create を優先
    // update API 実装後は、exists → update / create に切り替えてください
    if (opts?.mode === "update" || opts?.ifExists) {
      throw new Error("MemberRepositoryHTTP.save(update): not supported by current backend API");
    }
    if (opts?.mode === "create" || opts?.ifNotExists) {
      return this.create(member);
    }
    // デフォルトは create 相当
    return this.create(member);
  }

  async reset(): Promise<void> {
    // 開発用APIが無い前提
    throw new Error("MemberRepositoryHTTP.reset: not supported by current backend API");
  }
}
