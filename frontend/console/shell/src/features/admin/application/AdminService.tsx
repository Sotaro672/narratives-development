// frontend/console/shell/src/features/admin/application/AdminService.tsx
// Admin用のアプリケーションサービス

import { fetchMemberList } from "../../member/application/memberListService";
import type { PageRequest } from "../../../shared/types/common/common";
import { DEFAULT_PAGE_REQUEST } from "../../../shared/types/common/common";
import type { Member, MemberFilter } from "../../../shared/types/member";

export type AssigneeCandidate = {
  /**
   * Firestore members の document ID。
   *
   * Backend の assigneeId / managerId / NameResolver は
   * Member document ID を正とする。
   */
  id: string;

  /**
   * Backend で解決済みの Member 表示名。
   */
  name: string;
};

/**
 * Member配列をAdminCardやBrand責任者選択で使用する
 * 担当者候補とnameMapへ変換する。
 *
 * candidate.id は Firestore members の document ID を正とする。
 *
 * 表示名はBackendで解決済みのdisplayNameを使用する。
 * id が空のMemberは担当者候補に含めない。
 */
export function buildAssigneeCandidates(
  items: Member[],
): {
  candidates: AssigneeCandidate[];
  nameMap: Record<string, string>;
} {
  const candidates = items.flatMap(
    (member): AssigneeCandidate[] => {
      if (!member.id) {
        return [];
      }

      const name =
        member.displayName ||
        member.email ||
        member.id;

      return [
        {
          id: member.id,
          name,
        },
      ];
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
 * 担当者候補を取得する。
 *
 * companyIdはFrontendから指定せず、
 * Backend側で認証中MemberのcompanyIdにスコープされる。
 *
 * candidate.id は Firestore members の document ID。
 *
 * AdminCard の assigneeId や
 * Brand の managerId など、
 * Member document ID を保存する項目で共通利用する。
 */
export async function fetchAssigneeCandidatesForCurrentCompany(): Promise<{
  candidates: AssigneeCandidate[];
  nameMap: Record<string, string>;
}> {
  const page: PageRequest = {
    ...DEFAULT_PAGE_REQUEST,
    number: 1,
    perPage: 200,
  };

  const filter: MemberFilter = {};

  const result = await fetchMemberList(
    page,
    filter,
  );

  return buildAssigneeCandidates(
    result.items,
  );
}