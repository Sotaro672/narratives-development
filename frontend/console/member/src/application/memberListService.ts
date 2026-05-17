// frontend/console/member/src/application/memberListService.ts

import type { Member } from "../domain/entity/member";
import type { MemberFilter } from "../domain/repository/memberRepository";
import type { Page } from "../../../shell/src/shared/types/common/common";

// 認証（IDトークン取得用）
import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";

// Permission 型
import type {
  Permission,
  PermissionCategory,
} from "../../../shell/src/shared/types/permission";

// Permission Repository (GET /permissions)
import { PermissionRepositoryHTTP } from "../../../permission/src/infrastructure/http/permissionRepositoryHTTP";

// Brand Domain
import type { Brand } from "../../../brand/src/domain/entity/brand";
import { BrandRepositoryHTTP } from "../../../brand/src/infrastructure/http/brandRepositoryHTTP";

// Member Repository（HTTP 層）
import { MemberRepositoryHTTP } from "../infrastructure/http/memberRepositoryHTTP";

export type MemberListResult = {
  members: Member[];
  totalPages: number;
};

// Base URL
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)
    ?.replace(/\/+$/g, "") ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

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
  const map: Record<PermissionCategory, Permission[]> = {} as any;

  for (const p of allPermissions) {
    const cat = (p.category || "brand") as PermissionCategory;
    if (!map[cat]) map[cat] = [];
    map[cat].push(p);
  }

  return map;
}

// ─────────────────────────────────────────────
// CurrentMember
// ─────────────────────────────────────────────
export async function fetchCurrentMember(): Promise<Member | null> {
  const currentUser = auth.currentUser;
  if (!currentUser) return null;

  const uid = currentUser.uid.trim();
  if (!uid) return null;

  try {
    const member = await memberRepo.getByUid(uid);
    return member ?? null;
  } catch {
    return null;
  }
}

// ─────────────────────────────────────────────
// Brand 関連サービス
// ─────────────────────────────────────────────
export async function fetchBrandsForCurrentMember(): Promise<Brand[]> {
  const current = await fetchCurrentMember();
  const companyId = (current?.companyId ?? "").trim();
  if (!companyId) return [];

  return fetchBrandsByCompany(companyId);
}

export async function fetchBrandsByCompany(
  companyId: string | null,
): Promise<Brand[]> {
  if (!companyId) return [];

  const pageResult = await brandRepo.list({
    filter: {
      companyId,
      deleted: false,
      isActive: true,
    },
    sort: {
      column: "created_at",
      order: "desc",
    },
    page: 1,
    perPage: 200,
  });

  const items = pageResult.items as Brand[];
  return items.filter((b) => (b.companyId ?? "") === companyId);
}

// ─────────────────────────────────────────────
// Member 一覧
// ─────────────────────────────────────────────
export async function fetchMemberList(
  page: Page,
  filter: MemberFilter,
): Promise<MemberListResult> {
  const currentUser = auth.currentUser;
  if (!currentUser) {
    throw new Error("未認証のためメンバー一覧を取得できません。");
  }

  const pageResult = await memberRepo.list(page, filter);

  const normalized: Member[] = pageResult.items.map((m: Member): Member => {
    let assignedBrandIds: string[] | null = null;

    if (Array.isArray(m.assignedBrands) && m.assignedBrands.length > 0) {
      const ids = m.assignedBrands
        .map((id) => String(id ?? "").trim())
        .filter((id) => id.length > 0);

      assignedBrandIds = ids.length > 0 ? ids : null;
    }

    return {
      ...m,
      assignedBrands: assignedBrandIds,
    };
  });

  return {
    members: normalized,
    totalPages: pageResult.totalPages ?? 1,
  };
}