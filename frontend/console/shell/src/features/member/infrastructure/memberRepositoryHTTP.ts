// frontend/console/shell/src/features/member/infrastructure/memberRepositoryHTTP.ts

import type { PageRequest, PageResult } from "../../../shared/types/common/common";
import type {
  CreateMemberInput,
  Member,
  MemberFilter,
} from "../../../shared/types/member";

import { buildConsoleUrl } from "../../../shared/http/apiBase";
import { getAuthHeaders, getAuthJsonHeaders } from "../../../shared/http/authHeaders";
import { fetchJSON } from "../../../shared/http/fetchJSON";
import { withQuery } from "../../../shared/http/queryString";

export class MemberRepositoryHTTP {
  /**
   * Firebase UIDでMemberを取得する。
   *
   * Backend:
   * GET /members/{uid}
   */
  async getByUid(uid: string): Promise<Member | null> {
    const uidValue = uid.trim();
    if (!uidValue) {
      return null;
    }

    const headers = await getAuthHeaders();
    const url = buildConsoleUrl(`/members/${encodeURIComponent(uidValue)}`);
    const response = await fetch(url, { headers });

    if (response.status === 404) {
      return null;
    }

    const contentType = response.headers.get("content-type") ?? "";
    if (!contentType.includes("application/json")) {
      const text = await response.text().catch(() => "");
      throw new Error(`Unexpected content-type: ${contentType}\n${text.slice(0, 200)}`);
    }

    if (!response.ok) {
      const message = await response.text().catch(() => `HTTP ${response.status}`);
      throw new Error(message);
    }

    return (await response.json()) as Member;
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
    const headers = await getAuthHeaders();
    const pageNumber = page.number && page.number > 0 ? page.number : 1;
    const perPage = page.perPage && page.perPage > 0 ? page.perPage : 50;

    const url = withQuery(buildConsoleUrl("/members"), {
      q: filter?.searchQuery,
      uid: filter?.uid,
      brandIds: filter?.brandIds,
      status: filter?.status,
      page: pageNumber,
      perPage,
    });

    return fetchJSON<PageResult<Member>>(url, { headers });
  }

  /**
   * Memberを作成する。
   *
   * Backend:
   * POST /members
   */
  async create(input: CreateMemberInput): Promise<Member> {
    const headers = await getAuthJsonHeaders();
    const url = buildConsoleUrl("/members");

    return fetchJSON<Member>(url, {
      method: "POST",
      headers,
      body: JSON.stringify(input),
    });
  }
}