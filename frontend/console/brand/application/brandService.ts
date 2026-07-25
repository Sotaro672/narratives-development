// frontend/console/brand/src/application/brandService.ts
/// <reference types="vite/client" />

import { brandRepositoryHTTP } from "../infrastructure/http/brandRepositoryHTTP";
import { safeDateLabelJa } from "../../../shell/src/shared/util/dateJa";

export type BrandRow = {
  id: string;
  name: string;
  isActive: boolean;
  managerId?: string | null; // memberId
  memberName?: string; // 「姓 名」
  registeredAt: string; // YYYY/MM/DD
  updatedAt: string; // YYYY/MM/DD
};

// backend から返ってくる Brand の最小形
type Brand = {
  id: string;
  companyId?: string | null;
  name?: string | null;
  description?: string | null;
  websiteUrl?: string | null;
  brandIcon?: string | null;
  brandBackgroundImage?: string | null;
  isActive?: boolean | null;

  // backend DTO は managerId / memberName が来る
  managerId?: string | null;
  memberName?: string | null;

  walletAddress?: string | null;
  createdAt?: string | null;
  createdBy?: string | null;
  updatedAt?: string | null;
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
};

// backend brand 名：返答をそのまま受け渡す（trim しない）
export function formatBrandName(name: string | null | undefined): string {
  return name ?? "";
}

// ===========================
// companyId のブランド一覧取得
// - backend 返答をなるべくそのまま使う（trimしない）
// - 責任者名は backend の memberName をそのまま memberName に入れる
// ===========================
export async function listBrands(companyId: string): Promise<BrandRow[]> {
  if (!companyId) return [];

  const page = await brandRepositoryHTTP.list({
    filter: {
      deleted: false,
      isActive: true,
    },
    sort: { column: "created_at", order: "desc" },
    page: 1,
    perPage: 200,
  });

  const brands = (page.items ?? []) as Brand[];

  return brands.map((b) => ({
    id: b.id,
    name: formatBrandName(b.name),
    isActive: !!b.isActive,
    managerId: b.managerId ?? null,
    memberName: b.memberName ?? "",
    registeredAt: safeDateLabelJa(b.createdAt ?? "", ""),
    updatedAt: safeDateLabelJa(b.updatedAt ?? "", ""),
  }));
}