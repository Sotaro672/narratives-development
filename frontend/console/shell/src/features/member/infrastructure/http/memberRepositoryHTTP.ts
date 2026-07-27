// frontend\console\shell\src\features\member\infrastructure\http\memberRepositoryHTTP.ts
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
} from "../../../../shared/types/common/common";
import type {
  Member,
  MemberPatch,
} from "../../../../shared/types/member";

import { buildConsoleUrl } from "../../../../shared/http/apiBase";
import {
  getAuthHeaders,
  getAuthJsonHeaders,
} from "../../../../shared/http/authHeaders";
import { fetchJSON } from "../../../../shared/http/fetchJSON";
import { withQuery } from "../../../../shared/http/queryString";

type UnknownRecord = Record<string, unknown>;

function isRecord(value: unknown): value is UnknownRecord {
  return (
    typeof value === "object" &&
    value !== null &&
    !Array.isArray(value)
  );
}

function normalizeAssignedBrands(
  value: unknown,
): string[] | null {
  if (!Array.isArray(value)) {
    return null;
  }

  const assignedBrandIds = value
    .map((brandId) => String(brandId ?? "").trim())
    .filter((brandId) => brandId.length > 0);

  return assignedBrandIds.length > 0
    ? assignedBrandIds
    : null;
}

function normalizeMemberWire(value: unknown): Member {
  if (!isRecord(value)) {
    throw new Error(
      "メンバーレスポンスの形式が不正です。",
    );
  }

  return {
    ...value,
    assignedBrands: normalizeAssignedBrands(
      value.assignedBrands,
    ),
  } as unknown as Member;
}

function requirePositiveInteger(
  value: unknown,
  fieldName: string,
): number {
  if (
    typeof value !== "number" ||
    !Number.isFinite(value) ||
    !Number.isInteger(value) ||
    value <= 0
  ) {
    throw new Error(
      `メンバー一覧レスポンスの${fieldName}が不正です。`,
    );
  }

  return value;
}

function requireNonNegativeInteger(
  value: unknown,
  fieldName: string,
): number {
  if (
    typeof value !== "number" ||
    !Number.isFinite(value) ||
    !Number.isInteger(value) ||
    value < 0
  ) {
    throw new Error(
      `メンバー一覧レスポンスの${fieldName}が不正です。`,
    );
  }

  return value;
}

function normalizeMemberPageResult(
  value: unknown,
): PageResult<Member> {
  if (!isRecord(value)) {
    throw new Error(
      "メンバー一覧レスポンスの形式が不正です。",
    );
  }

  if (!Array.isArray(value.items)) {
    throw new Error(
      "メンバー一覧レスポンスにitemsがありません。",
    );
  }

  return {
    items: value.items.map(normalizeMemberWire),
    totalCount: requireNonNegativeInteger(
      value.totalCount,
      "totalCount",
    ),
    totalPages: requirePositiveInteger(
      value.totalPages,
      "totalPages",
    ),
    page: requirePositiveInteger(
      value.page,
      "page",
    ),
    perPage: requirePositiveInteger(
      value.perPage,
      "perPage",
    ),
  };
}

export class MemberRepositoryHTTP
  implements MemberRepository
{
  /**
   * Firebase UID で member を取得する。
   *
   * backend:
   * GET /members/{uid}
   */
  async getByUid(uid: string): Promise<Member | null> {
    const trimmed = uid.trim();

    if (!trimmed) {
      return null;
    }

    const headers = await getAuthHeaders();
    const url = buildConsoleUrl(
      `/members/${encodeURIComponent(trimmed)}`,
    );

    const res = await fetch(url, { headers });

    if (res.status === 404) {
      return null;
    }

    const contentType =
      res.headers.get("content-type") ?? "";

    if (!contentType.includes("application/json")) {
      const text = await res.text().catch(() => "");

      throw new Error(
        `Unexpected content-type: ${contentType}\n${text.slice(0, 200)}`,
      );
    }

    if (!res.ok) {
      const message = await res
        .text()
        .catch(() => `HTTP ${res.status}`);

      throw new Error(message);
    }

    const data: unknown = await res.json();

    return normalizeMemberWire(data);
  }

  async list(
    page: Page,
    filter?: MemberFilter,
  ): Promise<PageResult<Member>> {
    const headers = await getAuthHeaders();

    const pageNumber =
      page.number && page.number > 0
        ? page.number
        : 1;

    const perPage =
      page.perPage && page.perPage > 0
        ? page.perPage
        : 50;

    const url = withQuery(
      buildConsoleUrl("/members"),
      {
        q: filter?.searchQuery,
        uid: filter?.uid,
        brandIds: filter?.brandIds,
        status: filter?.status,
        page: pageNumber,
        perPage,
        sort: "updatedAt",
        order: "desc",
      },
    );

    const data = await fetchJSON<unknown>(url, {
      headers,
    });

    return normalizeMemberPageResult(data);
  }

  async create(member: Member): Promise<Member> {
    const headers = await getAuthJsonHeaders();
    const url = buildConsoleUrl("/members");

    const data = await fetchJSON<unknown>(url, {
      method: "POST",
      headers,
      body: JSON.stringify({
        firstName: member.firstName ?? "",
        lastName: member.lastName ?? "",
        firstNameKana:
          member.firstNameKana ?? "",
        lastNameKana:
          member.lastNameKana ?? "",
        email: member.email ?? "",
        permissions:
          member.permissions ?? [],
        assignedBrands:
          member.assignedBrands ?? [],
        status: member.status ?? "",
      }),
    });

    return normalizeMemberWire(data);
  }

  async update(
    docId: string,
    _patch: MemberPatch,
    _opts?: SaveOptions,
  ): Promise<Member> {
    void docId;

    throw new Error(
      "MemberRepositoryHTTP.update: not supported by current backend API",
    );
  }

  async delete(docId: string): Promise<void> {
    void docId;

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
      cursorPage.limit && cursorPage.limit > 0
        ? cursorPage.limit
        : 50;

    const page: Page = {
      number: 1,
      perPage: limit,
      totalPages: 1,
    };

    const res = await this.list(page, filter);

    return {
      items: res.items,
      nextCursor: null,
      prevCursor: undefined,
      hasNext: false,
      hasPrev: false,
    };
  }

  async getByEmail(
    email: string,
  ): Promise<Member | null> {
    const trimmed = email.trim().toLowerCase();

    if (!trimmed) {
      return null;
    }

    const res = await this.list(
      {
        number: 1,
        perPage: 50,
        totalPages: 1,
      },
      {
        searchQuery: trimmed,
      },
    );

    const hit = res.items.find(
      (member) =>
        (member.email ?? "")
          .trim()
          .toLowerCase() === trimmed,
    );

    return hit ?? null;
  }

  async existsByUid(uid: string): Promise<boolean> {
    return (await this.getByUid(uid)) !== null;
  }

  async count(filter: MemberFilter): Promise<number> {
    const res = await this.list(
      {
        number: 1,
        perPage: 100,
        totalPages: 1,
      },
      filter,
    );

    return res.totalCount;
  }

  async save(
    member: Member,
    opts?: SaveOptions,
  ): Promise<Member> {
    if (
      opts?.mode === "update" ||
      opts?.ifExists
    ) {
      throw new Error(
        "MemberRepositoryHTTP.save(update): not supported by current backend API",
      );
    }

    return this.create(member);
  }

  async reset(): Promise<void> {
    throw new Error(
      "MemberRepositoryHTTP.reset: not supported by current backend API",
    );
  }
}