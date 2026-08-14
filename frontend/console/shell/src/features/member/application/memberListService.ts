// frontend/console/shell/src/features/member/application/memberListService.ts

import type { Member, MemberFilter } from "../../../shared/types/member";
import type { PageRequest, PageResult } from "../../../shared/types/common/common";
import type { Permission, PermissionCategory } from "../../../shared/types/permission";
import type { Brand } from "../../../shared/types/brand";

import { auth } from "../../../auth/infrastructure/config/firebaseClient";
import { API_BASE } from "../../../shared/http/apiBase";
import { PermissionRepositoryHTTP } from "../../permission/infrastructure/http/permissionRepositoryHTTP";
import { BrandRepositoryHTTP } from "../../brand/infrastructure/http/brandRepositoryHTTP";
import { MemberRepositoryHTTP } from "../infrastructure/memberRepositoryHTTP";

const permissionRepo = new PermissionRepositoryHTTP();
const brandRepo = new BrandRepositoryHTTP();
const memberRepo = new MemberRepositoryHTTP();

// ─────────────────────────────────────────────
// Permission
// ─────────────────────────────────────────────

export async function fetchAllPermissions(): Promise<Permission[]> {
  const pageResult = await permissionRepo.list();
  return pageResult.items;
}

export function groupPermissionsByCategory(
  allPermissions: Permission[],
): Record<PermissionCategory, Permission[]> {
  const map = {} as Record<PermissionCategory, Permission[]>;

  for (const permission of allPermissions) {
    const category = (permission.category || "brand") as PermissionCategory;

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
// Brand
// ─────────────────────────────────────────────

export async function fetchBrandsForCurrentMember(): Promise<Brand[]> {
  const currentMember = await fetchCurrentMember();
  const companyId = (currentMember?.companyId ?? "").trim();

  if (!companyId) {
    return [];
  }

  return fetchBrandsByCompany(companyId);
}

export async function fetchBrandsByCompany(companyId: string | null): Promise<Brand[]> {
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
// Member
// ─────────────────────────────────────────────

export async function fetchMemberList(
  page: PageRequest,
  filter: MemberFilter,
): Promise<PageResult<Member>> {
  if (!auth.currentUser) {
    throw new Error("未認証のためメンバー一覧を取得できません。");
  }

  return memberRepo.list(page, filter);
}

export { API_BASE };