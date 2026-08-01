// frontend/console/shell/src/features/member/infrastructure/http/memberRepositoryHTTP.ts
/// <reference types="vite/client" />

import type {
  MemberRepository,
  MemberFilter,
} from "../../domain/repository/memberRepository";
import type {
  PageRequest,
  PageResult,
} from "../../../../shared/types/common/common";
import type { Member } from "../../../../shared/types/member";

import { buildConsoleUrl } from "../../../../shared/http/apiBase";
import {
  getAuthHeaders,
  getAuthJsonHeaders,
} from "../../../../shared/http/authHeaders";
import { fetchJSON } from "../../../../shared/http/fetchJSON";
import { withQuery } from "../../../../shared/http/queryString";

type UnknownRecord = Record<string, unknown>;

function isRecord(
  value: unknown,
): value is UnknownRecord {
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
    .map((brandId) =>
      String(brandId ?? "").trim(),
    )
    .filter(
      (brandId) =>
        brandId.length > 0,
    );

  return assignedBrandIds.length > 0
    ? assignedBrandIds
    : null;
}

function normalizeMemberWire(
  value: unknown,
): Member {
  if (!isRecord(value)) {
    throw new Error(
      "メンバーレスポンスの形式が不正です。",
    );
  }

  return {
    ...value,
    assignedBrands:
      normalizeAssignedBrands(
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
    items: value.items.map(
      normalizeMemberWire,
    ),
    totalCount:
      requireNonNegativeInteger(
        value.totalCount,
        "totalCount",
      ),
    totalPages:
      requirePositiveInteger(
        value.totalPages,
        "totalPages",
      ),
    page: requirePositiveInteger(
      value.page,
      "page",
    ),
    perPage:
      requirePositiveInteger(
        value.perPage,
        "perPage",
      ),
  };
}

export class MemberRepositoryHTTP
  implements MemberRepository
{
  /**
   * Firebase UIDでMemberを取得する。
   *
   * Backend:
   * GET /members/{uid}
   */
  async getByUid(
    uid: string,
  ): Promise<Member | null> {
    const trimmed = uid.trim();

    if (!trimmed) {
      return null;
    }

    const headers =
      await getAuthHeaders();

    const url = buildConsoleUrl(
      `/members/${encodeURIComponent(
        trimmed,
      )}`,
    );

    const response = await fetch(
      url,
      {
        headers,
      },
    );

    if (response.status === 404) {
      return null;
    }

    const contentType =
      response.headers.get(
        "content-type",
      ) ?? "";

    if (
      !contentType.includes(
        "application/json",
      )
    ) {
      const text = await response
        .text()
        .catch(() => "");

      throw new Error(
        `Unexpected content-type: ${contentType}\n${text.slice(
          0,
          200,
        )}`,
      );
    }

    if (!response.ok) {
      const message = await response
        .text()
        .catch(
          () =>
            `HTTP ${response.status}`,
        );

      throw new Error(message);
    }

    const data: unknown =
      await response.json();

    return normalizeMemberWire(data);
  }

  /**
   * Member一覧を取得する。
   *
   * Backend:
   * GET /members
   */
  async list(
    page: PageRequest,
    filter?: MemberFilter,
  ): Promise<PageResult<Member>> {
    const headers =
      await getAuthHeaders();

    const pageNumber =
      page.number &&
      page.number > 0
        ? page.number
        : 1;

    const perPage =
      page.perPage &&
      page.perPage > 0
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

    const data =
      await fetchJSON<unknown>(
        url,
        {
          headers,
        },
      );

    return normalizeMemberPageResult(
      data,
    );
  }

  /**
   * Memberを作成する。
   *
   * Backend:
   * POST /members
   */
  async create(
    member: Member,
  ): Promise<Member> {
    const headers =
      await getAuthJsonHeaders();

    const url =
      buildConsoleUrl("/members");

    const data =
      await fetchJSON<unknown>(
        url,
        {
          method: "POST",
          headers,
          body: JSON.stringify({
            firstName:
              member.firstName,
            lastName:
              member.lastName,
            firstNameKana:
              member.firstNameKana,
            lastNameKana:
              member.lastNameKana,
            email:
              member.email,
            permissions:
              member.permissions,
            assignedBrands:
              member.assignedBrands ??
              [],
            status:
              member.status,
          }),
        },
      );

    return normalizeMemberWire(
      data,
    );
  }
}