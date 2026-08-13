// frontend/console/shell/src/features/admin/application/AdminService.tsx
// Admin用のアプリケーションサービス

import {
  fetchMemberList,
} from "../../member/application/memberListService";

import type {
  MemberFilter,
} from "../../member/domain/repository/memberRepository";

import type {
  PageRequest,
} from "../../../shared/types/common/common";

import {
  DEFAULT_PAGE_REQUEST,
} from "../../../shared/types/common/common";

import type {
  Member,
} from "../../../shared/types/member";

export type AssigneeCandidate = {
  /**
   * Firebase Authentication UID。
   *
   * Backend の assigneeId / NameResolver は
   * Firebase Auth UID を正とする。
   */
  id: string;

  /**
   * Backend で解決済みの Member 表示名。
   */
  name: string;
};

/**
 * Member配列をAdminCard用の担当者候補とnameMapへ変換する。
 *
 * assigneeId は Firebase Auth UID を正とする。
 *
 * 表示名は次の優先順位で決定する。
 * 1. displayName
 * 2. 姓名
 * 3. email
 *
 * uid が空の Member は担当者候補に含めない。
 */
export function buildAssigneeCandidates(
  items: Member[],
): {
  candidates: AssigneeCandidate[];
  nameMap: Record<string, string>;
} {
  const candidates =
    items.flatMap(
      (
        member,
      ): AssigneeCandidate[] => {
        if (!member.uid) {
          return [];
        }

        const fullName =
          [
            member.lastName,
            member.firstName,
          ]
            .filter(
              (value) =>
                value.length > 0,
            )
            .join(" ");

        const name =
          member.displayName ||
          fullName ||
          member.email;

        if (!name) {
          return [];
        }

        return [
          {
            id:
              member.uid,

            name,
          },
        ];
      },
    );

  const nameMap:
    Record<string, string> = {};

  for (
    const candidate
    of candidates
  ) {
    nameMap[
      candidate.id
    ] =
      candidate.name;
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
 *
 * candidate.id は Firebase Auth UID。
 */
export async function fetchAssigneeCandidatesForCurrentCompany(): Promise<{
  candidates: AssigneeCandidate[];
  nameMap: Record<string, string>;
}> {
  const page:
    PageRequest = {
      ...DEFAULT_PAGE_REQUEST,

      number:
        1,

      perPage:
        200,
    };

  const filter:
    MemberFilter = {};

  const result =
    await fetchMemberList(
      page,
      filter,
    );

  return buildAssigneeCandidates(
    result.items,
  );
}