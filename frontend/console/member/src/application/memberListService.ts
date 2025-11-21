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

// Query APIs
import {
  fetchMemberListWithToken,
  fetchMemberByIdWithToken,
  formatLastFirst,
} from "../infrastructure/query/memberQuery";

export type MemberListResult = {
  members: Member[];
  nameMap: Record<string, string>;
  /** backend から受け取った総ページ数（Pagination 用） */
  totalPages: number;
};

// ─────────────────────────────────────────────
// Backend base URL
// ─────────────────────────────────────────────
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)
    ?.replace(/\/+$/g, "") ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

// Singletons
const permissionRepo = new PermissionRepositoryHTTP();
const brandRepo = new BrandRepositoryHTTP();

// ─────────────────────────────────────────────
// Permission 関連サービス
// ─────────────────────────────────────────────
export async function fetchAllPermissions(): Promise<Permission[]> {
  const pageResult = await permissionRepo.list();
  return pageResult.items;
}

// カテゴリ別の Permission グルーピング
export function groupPermissionsByCategory(
  allPermissions: Permission[],
): Record<PermissionCategory, Permission[]> {
  const map: Record<string, Permission[]> = {};
  for (const p of allPermissions) {
    const cat = (p.category || "brand") as PermissionCategory;
    if (!map[cat]) map[cat] = [];
    map[cat].push(p);
  }
  return map as Record<PermissionCategory, Permission[]>;
}

// ─────────────────────────────────────────────
// Member / CurrentMember 関連サービス
// ─────────────────────────────────────────────
export async function fetchCurrentMember(): Promise<Member | null> {
  const currentUser = auth.currentUser;
  if (!currentUser) {
    console.warn("[memberService] fetchCurrentMember: no auth.currentUser");
    return null;
  }

  const uid = currentUser.uid;
  const token = await currentUser.getIdToken();

  console.log(
    "[memberService] fetchCurrentMember uid:",
    uid,
    "GET",
    `${API_BASE}/members/${encodeURIComponent(uid)}`,
  );

  try {
    const member = await fetchMemberByIdWithToken(token, uid);
    if (!member) return null;
    return member;
  } catch (e) {
    console.error("[memberService] fetchCurrentMember error:", e);
    return null;
  }
}

// ─────────────────────────────────────────────
// Brand 関連サービス（他用途で使用、fetchMemberList では使用しない）
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

  console.log("[memberService] fetchBrandsByCompany companyId =", companyId);

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

  const items = (pageResult.items ?? []) as Brand[];
  const filtered = items.filter((b) => (b.companyId ?? "") === companyId);

  console.log(
    "[memberService] fetchBrandsByCompany brands =",
    filtered.map((b) => ({ id: b.id, name: b.name, companyId: b.companyId })),
  );

  return filtered;
}

// ─────────────────────────────────────────────
// Member 一覧（ブランド取得なし版）
// ─────────────────────────────────────────────
export async function fetchMemberList(
  page: Page,
  filter: MemberFilter,
): Promise<MemberListResult> {
  const currentUser = auth.currentUser;
  if (!currentUser) {
    throw new Error("未認証のためメンバー一覧を取得できません。");
  }

  const token = await currentUser.getIdToken();

  console.log(
    "[memberService.fetchMemberList] currentUser.uid:",
    currentUser.uid,
  );

  // backend の PageResult<Member> 相当
  const { items, totalPages } = await fetchMemberListWithToken(
    token,
    page,
    filter,
  );

  // ★★ ブランド取得は完全に削除（無限ループ防止） ★★
  // useMemberList がブランド一覧を管理する

  // 正規化
  const normalized: Member[] = items.map((m) => {
    const noFirst = !String(m.firstName ?? "").trim();
    const noLast = !String(m.lastName ?? "").trim();

    // Firestore → ID 配列のまま保持
    let assignedBrandIds: string[] | null = null;
    if (Array.isArray(m.assignedBrands) && m.assignedBrands.length > 0) {
      const ids = m.assignedBrands
        .map((id) => String(id ?? "").trim())
        .filter((id) => id.length > 0);
      assignedBrandIds = ids.length > 0 ? ids : null;
    }

    const base: Member =
      noFirst && noLast
        ? ({ ...m, firstName: "招待中" } as Member)
        : (m as Member);

    return {
      ...base,
      assignedBrands: assignedBrandIds,
    };
  });

  // 氏名キャッシュ
  const nameMap: Record<string, string> = {};
  for (const m of normalized) {
    const disp = formatLastFirst(m.lastName as any, m.firstName as any);
    if (disp) nameMap[m.id] = disp;
  }

  return {
    members: normalized,
    nameMap,
    totalPages: totalPages ?? 1,
  };
}

// ─────────────────────────────────────────────
// 単一メンバーの表示名取得
// ─────────────────────────────────────────────
export async function fetchMemberNameLastFirstById(
  memberId: string,
): Promise<string> {
  const id = String(memberId ?? "").trim();
  if (!id) return "";

  const currentUser = auth.currentUser;
  if (!currentUser) return "";

  const token = await currentUser.getIdToken();
  const member = await fetchMemberByIdWithToken(token, id);
  if (!member) return "";

  const disp = formatLastFirst(
    member.lastName as any,
    member.firstName as any,
  );
  return disp ?? "";
}
