// frontend/console/member/src/application/memberDetailService.ts

import type { Member } from "../domain/entity/member";
import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";
import { MemberRepositoryHTTP } from "../infrastructure/http/memberRepositoryHTTP";

import {
  groupPermissionsByCategory,
  type PermissionCategory,
} from "../../../permission/src/application/permissionCatalog";

const memberRepo = new MemberRepositoryHTTP();

/**
 * メンバー詳細取得
 *
 * IMPORTANT:
 * - backend の GET /members/{uid} は Firebase UID 専用
 * - Firestore member docId ではなく Firebase Auth UID を渡すこと
 */
export async function fetchMemberDetailByUid(
  uid: string,
): Promise<Member | null> {
  const firebaseUid = String(uid ?? "").trim();
  if (!firebaseUid) return null;

  const currentUser = auth.currentUser;
  if (!currentUser) {
    throw new Error("未認証のためメンバー情報を取得できません。");
  }

  const raw = await memberRepo.getByUid(firebaseUid);
  if (!raw) return null;

  const noFirst =
    raw.firstName === null ||
    raw.firstName === undefined ||
    raw.firstName === "";

  const noLast =
    raw.lastName === null ||
    raw.lastName === undefined ||
    raw.lastName === "";

  const permissions: string[] = raw.permissions ?? [];

  const permissionGroups = groupPermissionsByCategory(permissions);

  const permissionCategories: PermissionCategory[] = Object.keys(
    permissionGroups,
  ) as PermissionCategory[];

  return {
    ...raw,

    // raw.id は backend が返す Firestore member docId
    id: raw.id,

    // raw.uid は Firebase Auth UID
    uid: raw.uid ?? firebaseUid,

    firstName: noFirst ? null : raw.firstName ?? null,
    lastName: noLast ? null : raw.lastName ?? null,

    permissionGroups,
    permissionCategories,
  } as Member;
}