// frontend/console/shell/src/features/member/application/memberListService.ts

import type { Member } from "../../../shared/types/member";
import type { MemberFilter } from "../domain/repository/memberRepository";
import type {
  PageRequest,
  PageResult,
} from "../../../shared/types/common/common";

// 認証（IDトークン取得用）
import { auth } from "../../../auth/infrastructure/config/firebaseClient";

// Shared API base
import { API_BASE } from "../../../shared/http/apiBase";

// Permission 型
import type {
  Permission,
  PermissionCategory,
} from "../../../shared/types/permission";

// Permission Repository（GET /permissions）
import { PermissionRepositoryHTTP } from "../../permission/infrastructure/http/permissionRepositoryHTTP";

// Brand
import type { Brand } from "../../../shared/types/brand";
import { BrandRepositoryHTTP } from "../../brand/infrastructure/http/brandRepositoryHTTP";

// Member Repository（HTTP 層）
import { MemberRepositoryHTTP } from "../infrastructure/http/memberRepositoryHTTP";

// Singletons
const permissionRepo = new PermissionRepositoryHTTP();
const brandRepo = new BrandRepositoryHTTP();
const memberRepo = new MemberRepositoryHTTP();

// ─────────────────────────────────────────────
// Permission 関連サービス
// ─────────────────────────────────────────────

export async function fetchAllPermissions(): Promise<Permission[]> {
  const pageResult = await permissionRepo.list();

  return pageResult.items;
}

export function groupPermissionsByCategory(
  allPermissions: Permission[],
): Record<PermissionCategory, Permission[]> {
  const map = {} as Record<
    PermissionCategory,
    Permission[]
  >;

  for (const permission of allPermissions) {
    const category = (
      permission.category || "brand"
    ) as PermissionCategory;

    if (!map[category]) {
      map[category] = [];
    }

    map[category].push(permission);
  }

  return map;
}

// ─────────────────────────────────────────────
// CurrentMember
// ─────────────────────────────────────────────

export async function fetchCurrentMember(): Promise<Member | null> {
  const currentUser = auth.currentUser;

  if (!currentUser) {
    return null;
  }

  const uid = currentUser.uid.trim();

  if (!uid) {
    return null;
  }

  try {
    return await memberRepo.getByUid(uid);
  } catch {
    return null;
  }
}

// ─────────────────────────────────────────────
// Brand 関連サービス
// ─────────────────────────────────────────────

export async function fetchBrandsForCurrentMember(): Promise<
  Brand[]
> {
  const currentMember = await fetchCurrentMember();
  const companyId = (
    currentMember?.companyId ?? ""
  ).trim();

  if (!companyId) {
    return [];
  }

  return fetchBrandsByCompany(companyId);
}

export async function fetchBrandsByCompany(
  companyId: string | null,
): Promise<Brand[]> {
  if (!companyId) {
    return [];
  }

  const pageResult = await brandRepo.list({
    page: 1,
    perPage: 200,
  });

  return pageResult.items.filter(
    (brand) =>
      (brand.companyId ?? "") === companyId &&
      brand.isActive &&
      !brand.deletedAt,
  );
}

// ─────────────────────────────────────────────
// Member 一覧
// ─────────────────────────────────────────────

export async function fetchMemberList(
  page: PageRequest,
  filter: MemberFilter,
): Promise<PageResult<Member>> {
  const currentUser = auth.currentUser;

  if (!currentUser) {
    throw new Error(
      "未認証のためメンバー一覧を取得できません。",
    );
  }

  return memberRepo.list(
    page,
    filter,
  );
}

export { API_BASE };