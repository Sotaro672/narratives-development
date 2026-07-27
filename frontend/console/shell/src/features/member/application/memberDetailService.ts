// frontend/console/shell/src/features/member/application/memberDetailService.ts

import type { Member } from "../../../shared/types/member";
import { auth } from "../../../auth/infrastructure/config/firebaseClient";
import { MemberRepositoryHTTP } from "../infrastructure/http/memberRepositoryHTTP";

import {
  groupPermissionsByCategory,
  type PermissionCategory,
} from "../../permission/application/permissionCatalog";

const memberRepo = new MemberRepositoryHTTP();

export type MemberDetail = Member & {
  permissionGroups: ReturnType<
    typeof groupPermissionsByCategory
  >;
  permissionCategories: PermissionCategory[];
};

/**
 * メンバー詳細取得
 *
 * IMPORTANT:
 * - BackendのGET /members/{uid}はFirebase UID専用
 * - Firestore MemberのdocIdではなくFirebase Auth UIDを渡す
 */
export async function fetchMemberDetailByUid(
  uid: string,
): Promise<MemberDetail | null> {
  const firebaseUid = String(uid ?? "").trim();

  if (!firebaseUid) {
    return null;
  }

  const currentUser = auth.currentUser;

  if (!currentUser) {
    throw new Error(
      "未認証のためメンバー情報を取得できません。",
    );
  }

  const member = await memberRepo.getByUid(
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
    ) as PermissionCategory[];

  return {
    ...member,

    // member.idはBackendが返すFirestore MemberのdocId
    id: member.id,

    // 招待前などで空の場合は検索に使用したFirebase UIDを補完
    uid: member.uid || firebaseUid,

    permissionGroups,
    permissionCategories,
  };
}