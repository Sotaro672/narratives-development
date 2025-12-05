// frontend/console/admin/src/application/AdminService.tsx
// Admin 用のアプリケーションサービス

import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";
import {
  fetchMemberListWithToken,
} from "../../../member/src/infrastructure/query/memberQuery";
import type {
  MemberFilter,
} from "../../../member/src/domain/repository/memberRepository";
import type {
  Page,
} from "../../../shell/src/shared/types/common/common";
import {
  DEFAULT_PAGE,
} from "../../../shell/src/shared/types/common/common";
import type { Member } from "../../../member/src/domain/entity/member";

export type AssigneeCandidate = {
  id: string;
  name: string;
};

// backend の /members/by-company が返す型：Member + displayName
type MemberWithDisplayName = Member & {
  displayName?: string;
};

/**
 * 生メンバー配列 → AdminCard 用の候補配列 & nameMap に変換
 * displayName を優先的に使う
 */
export function buildAssigneeCandidates(
  items: MemberWithDisplayName[],
): { candidates: AssigneeCandidate[]; nameMap: Record<string, string> } {

  const candidates: AssigneeCandidate[] = items.map((m: MemberWithDisplayName) => {
    const full =
      (m.displayName ?? "").trim() ||
      `${m.lastName ?? ""} ${m.firstName ?? ""}`.trim() ||
      m.email ||
      m.id;

    return { id: m.id, name: full };
  });

  const nameMap: Record<string, string> = {};
  candidates.forEach((c) => {
    nameMap[c.id] = c.name;
  });

  return { candidates, nameMap };
}

/**
 * 現在ログイン中ユーザーの companyId コンテキストで
 * AdminCard 用の担当者候補を取得する
 */
export async function fetchAssigneeCandidatesForCurrentCompany(): Promise<{
  candidates: AssigneeCandidate[];
  nameMap: Record<string, string>;
}> {
  const fbUser = auth.currentUser;
  if (!fbUser) {
    return { candidates: [], nameMap: {} };
  }

  const token = await fbUser.getIdToken();

  const page: Page = { ...DEFAULT_PAGE, number: 1, perPage: 200 };
  const filter: MemberFilter = {};

  // ★ listMembersByCompanyId → displayName 付きレスポンスを取得する想定
  const { items } = await fetchMemberListWithToken(token, page, filter);

  const { candidates, nameMap } = buildAssigneeCandidates(items as MemberWithDisplayName[]);

  return { candidates, nameMap };
}
