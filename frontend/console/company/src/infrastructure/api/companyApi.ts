import type { Company } from "../../domain/entity/company";
import type {
  CompanyRepository,
  CompanyFilter,
  CompanyPatch,
  Sort,
  Page,
  PageResult,
  CursorPage,
  CursorPageResult,
  SaveOptions,
} from "../../domain/repository/companyRepository";

/**
 * REST API による CompanyRepository の実装
 */
export class CompanyApi implements CompanyRepository {
  private readonly baseUrl: string;

  constructor(baseUrl: string = "/api/companies") {
    this.baseUrl = baseUrl;
  }

  // --------------------------
  // 共通 fetch wrapper
  // --------------------------
  private async request<T>(input: RequestInfo, init?: RequestInit): Promise<T> {
    const res = await fetch(input, {
      headers: {
        "Content-Type": "application/json",
        ...(init?.headers ?? {})
      },
      credentials: "include",
      ...init,
    });

    if (!res.ok) {
      let message = `Request failed: ${res.status}`;
      try {
        const data = await res.json();
        if (data?.message) message = data.message;
      } catch {}
      throw new Error(message);
    }

    if (res.status === 204) {
      return undefined as unknown as T;
    }

    return (await res.json()) as T;
  }

  // --------------------------
  // Query Builder
  // --------------------------
  private buildQuery(
    filter?: CompanyFilter,
    sort?: Sort,
    page?: Page,
    cursor?: CursorPage
  ): string {
    const params = new URLSearchParams();

    // ---- Filter ----
    if (filter) {
      if (filter.searchQuery) params.set("q", filter.searchQuery);

      if (filter.ids) filter.ids.forEach((id) => params.append("ids", id));
      if (filter.name) params.set("name", filter.name);
      if (filter.admin) params.set("admin", filter.admin);
      if (typeof filter.isActive === "boolean") params.set("isActive", String(filter.isActive));

      if (filter.createdBy) params.set("createdBy", filter.createdBy);
      if (filter.updatedBy) params.set("updatedBy", filter.updatedBy);
      if (filter.deletedBy) params.set("deletedBy", filter.deletedBy);

      if (filter.createdFrom) params.set("createdFrom", filter.createdFrom);
      if (filter.createdTo) params.set("createdTo", filter.createdTo);

      if (filter.updatedFrom) params.set("updatedFrom", filter.updatedFrom);
      if (filter.updatedTo) params.set("updatedTo", filter.updatedTo);

      if (filter.deletedFrom) params.set("deletedFrom", filter.deletedFrom);
      if (filter.deletedTo) params.set("deletedTo", filter.deletedTo);

      if (typeof filter.deleted === "boolean") params.set("deleted", String(filter.deleted));
    }

    // ---- Sort ----
    if (sort) {
      params.set("sortField", sort.field);
      params.set("sortOrder", sort.order);
    }

    // ---- Page (page / size) ----
    if (page) {
      params.set("page", String(page.page));
      params.set("size", String(page.size));
    }

    // ---- Cursor Page (cursor / size)----
    if (cursor) {
      params.set("size", String(cursor.size));
      if (cursor.cursor) params.set("cursor", cursor.cursor);
    }

    const qs = params.toString();
    return qs ? `?${qs}` : "";
  }

  // --------------------------
  // Repository implementation
  // --------------------------

  /** 一覧取得（ページ番号ベース）*/
  async list(
    filter: CompanyFilter,
    sort: Sort,
    page: Page
  ): Promise<PageResult<Company>> {
    const qs = this.buildQuery(filter, sort, page);
    return this.request<PageResult<Company>>(`${this.baseUrl}${qs}`);
  }

  /** 一覧取得（カーソルベース）*/
  async listByCursor(
    filter: CompanyFilter,
    sort: Sort,
    cursor: CursorPage
  ): Promise<CursorPageResult<Company>> {
    const qs = this.buildQuery(filter, sort, undefined, cursor);
    return this.request<CursorPageResult<Company>>(`${this.baseUrl}/cursor${qs}`);
  }

  /** 単体取得（getById） */
  async getById(id: string): Promise<Company | null> {
    const url = `${this.baseUrl}/${encodeURIComponent(id)}`;
    try {
      return await this.request<Company>(url);
    } catch (e: any) {
      if (e?.message?.includes("404")) return null;
      throw e;
    }
  }

  /** 存在チェック */
  async exists(id: string): Promise<boolean> {
    const url = `${this.baseUrl}/${encodeURIComponent(id)}/exists`;
    const res = await this.request<{ exists: boolean }>(url);
    return res.exists;
  }

  /** 件数取得 */
  async count(filter: CompanyFilter): Promise<number> {
    const qs = this.buildQuery(filter);
    const res = await this.request<{ count: number }>(`${this.baseUrl}/count${qs}`);
    return res.count;
  }

  /** 作成 */
  async create(c: Company): Promise<Company> {
    return this.request<Company>(this.baseUrl, {
      method: "POST",
      body: JSON.stringify(c),
    });
  }

  /** 更新 (PATCH) */
  async update(id: string, patch: CompanyPatch): Promise<Company> {
    const url = `${this.baseUrl}/${encodeURIComponent(id)}`;
    return this.request<Company>(url, {
      method: "PATCH",
      body: JSON.stringify(patch),
    });
  }

  /** 削除 */
  async delete(id: string): Promise<void> {
    const url = `${this.baseUrl}/${encodeURIComponent(id)}`;
    await this.request<void>(url, { method: "DELETE" });
  }

  /** Save / Upsert */
  async save(c: Company, opts?: SaveOptions): Promise<Company> {
    const id = (c as any).id;

    // id があれば PUT
    if (id) {
      const url = `${this.baseUrl}/${encodeURIComponent(id)}`;
      return this.request<Company>(url, {
        method: "PUT",
        body: JSON.stringify(c),
      });
    }

    // id がなければ Create
    return this.create(c);
  }
}
