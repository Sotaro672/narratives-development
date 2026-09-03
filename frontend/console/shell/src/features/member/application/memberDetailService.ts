// frontend/console/shell/src/features/member/application/memberDetailService.ts

import { auth } from "../../../auth/infrastructure/config/firebaseClient";
import type { Member } from "../../../shared/types/member";
import {
  PERMISSION_CATEGORIES,
  type PermissionCategory,
} from "../../../shared/types/permission";
import { MemberRepositoryHTTP } from "../infrastructure/memberRepositoryHTTP";

const memberRepo = new MemberRepositoryHTTP();

export type PermissionGroups = Partial<
  Record<PermissionCategory, string[]>
>;

export type MemberDetail = Member & {
  permissionGroups: PermissionGroups;
  permissionCategories: PermissionCategory[];
};

/**
 * 文字列がPermissionCategoryか判定する。
 * カテゴリの定義元はshared/types/permission.tsのPERMISSION_CATEGORIESだけとする。
 */
function isPermissionCategory(value: string): value is PermissionCategory {
  return PERMISSION_CATEGORIES.some((category) => category === value);
}

/**
 * Permission名をカテゴリごとに分類する。
 *
 * 例:
 * - brand.view → brand
 * - brand.detail.view → brand
 * - member.roles.view → member
 */
function groupPermissionsByCategory(
  permissionNames: string[],
): PermissionGroups {
  const groups: PermissionGroups = {};

  for (const permissionName of permissionNames) {
    if (!permissionName) {
      continue;
    }

    const separatorIndex = permissionName.indexOf(".");
    if (separatorIndex <= 0) {
      continue;
    }

    const category = permissionName.slice(0, separatorIndex);
    if (!isPermissionCategory(category)) {
      continue;
    }

    const categoryPermissions = groups[category] ?? [];
    groups[category] = [...categoryPermissions, permissionName];
  }

  return groups;
}

/**
 * Firestore Member document IDからメンバー詳細を取得する。
 *
 * Backend:
 * GET /members/by-id/{memberId}
 *
 * Firebase UIDによるfallbackは行わない。
 * 招待中Memberではuidが空文字のまま返される。
 */
export async function fetchMemberDetailById(
  memberId: string,
): Promise<MemberDetail | null> {
  if (!memberId) {
    return null;
  }

  if (!auth.currentUser) {
    throw new Error("未認証のためメンバー情報を取得できません。");
  }

  const member = await memberRepo.getById(memberId);
  if (!member) {
    return null;
  }

  const permissionGroups = groupPermissionsByCategory(member.permissions);
  const permissionCategories = Object.keys(permissionGroups).filter(
    isPermissionCategory,
  );

  return {
    ...member,
    permissionGroups,
    permissionCategories,
  };
}