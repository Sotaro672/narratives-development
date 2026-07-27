// frontend/console/shell/src/features/admin/application/AdminService.tsx
// Admin用のアプリケーションサービス

import {
  fetchMemberList,
} from "../../member/application/memberListService";
import type {
  MemberFilter,
} from "../../member/domain/repository/memberRepository";
import type {
  Page,
} from "../../../shared/types/common/common";
import {
  DEFAULT_PAGE,
} from "../../../shared/types/common/common";
import type {
  Member,
} from "../../../shared/types/member";

export type AssigneeCandidate = {
  id: string;
  name: string;
};

/**
 * Member配列をAdminCard用の担当者候補とnameMapへ変換する。
 *
 * 表示名は次の優先順位で決定する。
 * 1. displayName
 * 2. 姓名
 * 3. email
 * 4. Member ID
 */
export function buildAssigneeCandidates(
  items: Member[],
): {
  candidates: AssigneeCandidate[];
  nameMap: Record<string, string>;
} {
  const candidates: AssigneeCandidate[] = items.map(
    (member) => {
      const fullName = [
        member.lastName,
        member.firstName,
      ]
        .filter((value) => value.length > 0)
        .join(" ");

      const name =
        member.displayName.trim() ||
        fullName ||
        member.email ||
        member.id;

      return {
        id: member.id,
        name,
      };
    },
  );

  const nameMap: Record<string, string> = {};

  for (const candidate of candidates) {
    nameMap[candidate.id] = candidate.name;
  }

  return {
    candidates,
    nameMap,
  };
}

/**
 * 現在ログイン中MemberのcompanyIdでスコープされた
 * AdminCard用の担当者候補を取得する。
 *
 * companyIdはFrontendから指定せず、
 * Backend側で認証中MemberのcompanyIdにスコープされる。
 */
export async function fetchAssigneeCandidatesForCurrentCompany(): Promise<{
  candidates: AssigneeCandidate[];
  nameMap: Record<string, string>;
}> {
  const page: Page = {
    ...DEFAULT_PAGE,
    number: 1,
    perPage: 200,
  };

  const filter: MemberFilter = {};

  const result = await fetchMemberList(
    page,
    filter,
  );

  return buildAssigneeCandidates(
    result.members,
  );
}