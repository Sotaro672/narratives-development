// frontend/console/admin/src/application/AdminService.tsx
// Admin 用のアプリケーションサービス
// - メンバー一覧取得
// - AdminCard 用の担当者候補リスト整形 など

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

/**
 * 生メンバー配列 → AdminCard 用の候補配列 & nameMap に変換
 */
export function buildAssigneeCandidates(
  items: Member[],
): { candidates: AssigneeCandidate[]; nameMap: Record<string, string> } {
  const candidates: AssigneeCandidate[] = items.map((m: Member) => {
    const full =
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
 * AdminCard 用の担当者候補を取得する。
 *
 * - Firebase Auth から ID トークンを取得
 * - /members を叩く
 * - AdminCard 用フォーマットに変換して返す
 */
export async function fetchAssigneeCandidatesForCurrentCompany(): Promise<{
  candidates: AssigneeCandidate[];
  nameMap: Record<string, string>;
}> {
  const fbUser = auth.currentUser;
  if (!fbUser) {
    // 未ログインの場合は空を返す
    return { candidates: [], nameMap: {} };
  }

  const token = await fbUser.getIdToken();

  const page: Page = { ...DEFAULT_PAGE, number: 1, perPage: 200 };
  const filter: MemberFilter = {};

  const { items } = await fetchMemberListWithToken(token, page, filter);

  const { candidates, nameMap } = buildAssigneeCandidates(items as Member[]);

  return { candidates, nameMap };
}
