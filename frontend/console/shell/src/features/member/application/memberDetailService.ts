// frontend/console/shell/src/features/member/application/memberDetailService.ts

import {
  auth,
} from "../../../auth/infrastructure/config/firebaseClient";

import type {
  Member,
} from "../../../shared/types/member";

import {
  PERMISSION_CATEGORIES,
  type PermissionCategory,
} from "../../../shared/types/permission";

import {
  MemberRepositoryHTTP,
} from "../infrastructure/http/memberRepositoryHTTP";

const memberRepo =
  new MemberRepositoryHTTP();

export type PermissionGroups =
  Partial<
    Record<
      PermissionCategory,
      string[]
    >
  >;

export type MemberDetail =
  Member & {
    permissionGroups:
      PermissionGroups;

    permissionCategories:
      PermissionCategory[];
  };

/**
 * 文字列がPermissionCategoryか判定する。
 *
 * カテゴリの定義元は
 * shared/types/permission.ts の
 * PERMISSION_CATEGORIESだけとする。
 */
function isPermissionCategory(
  value: string,
): value is PermissionCategory {
  return PERMISSION_CATEGORIES.some(
    (category) =>
      category === value,
  );
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
  const groups:
    PermissionGroups = {};

  for (
    const permissionName of
    permissionNames
  ) {
    const name =
      permissionName.trim();

    if (!name) {
      continue;
    }

    const separatorIndex =
      name.indexOf(".");

    if (
      separatorIndex <= 0
    ) {
      continue;
    }

    const category =
      name.slice(
        0,
        separatorIndex,
      );

    if (
      !isPermissionCategory(
        category,
      )
    ) {
      continue;
    }

    const categoryPermissions =
      groups[category] ?? [];

    groups[category] = [
      ...categoryPermissions,
      name,
    ];
  }

  return groups;
}

/**
 * メンバー詳細取得
 *
 * IMPORTANT:
 * - BackendのGET /members/{uid}はFirebase UID専用
 * - Firestore MemberのdocIdではなくFirebase Auth UIDを渡す
 */
export async function fetchMemberDetailByUid(
  uid: string,
): Promise<
  MemberDetail | null
> {
  const firebaseUid =
    String(
      uid ?? "",
    ).trim();

  if (!firebaseUid) {
    return null;
  }

  const currentUser =
    auth.currentUser;

  if (!currentUser) {
    throw new Error(
      "未認証のためメンバー情報を取得できません。",
    );
  }

  const member =
    await memberRepo.getByUid(
      firebaseUid,
    );

  if (!member) {
    return null;
  }

  const permissionGroups =
    groupPermissionsByCategory(
      member.permissions,
    );

  const permissionCategories =
    Object.keys(
      permissionGroups,
    ).filter(
      isPermissionCategory,
    );

  return {
    ...member,

    // member.idはBackendが返すFirestore MemberのdocId
    id:
      member.id,

    // 招待前などで空の場合は検索に使用したFirebase UIDを補完
    uid:
      member.uid ||
      firebaseUid,

    permissionGroups,
    permissionCategories,
  };
}