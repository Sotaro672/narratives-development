// frontend/console/shell/src/features/brand/application/brandService.ts
/// <reference types="vite/client" />

import type { Brand } from "../../../shared/types/brand";
import { brandRepositoryHTTP } from "../infrastructure/http/brandRepositoryHTTP";
import { safeDateLabelJa } from "../../../shared/util/dateJa";

export type BrandRow = {
  id: string;
  name: string;
  isActive: boolean;
  managerId: string | null;
  memberName: string;
  registeredAt: string;
  updatedAt: string;
};

// Backendのブランド名をそのまま受け渡す
export function formatBrandName(
  name: string | null | undefined,
): string {
  return name ?? "";
}

// ===========================
// ブランド一覧取得
// - BackendにはpageとperPageのみを送る
// - Backend側で認証中ユーザーのcompanyIdに絞り込む
// - 責任者名はmemberNameをそのまま使用する
// ===========================
export async function listBrands(): Promise<BrandRow[]> {
  const page = await brandRepositoryHTTP.list({
    page: 1,
    perPage: 200,
  });

  return page.items.map((brand: Brand) => ({
    id: brand.id,
    name: formatBrandName(brand.name),
    isActive: Boolean(brand.isActive),
    managerId: brand.managerId ?? null,
    memberName: brand.memberName ?? "",
    registeredAt: safeDateLabelJa(
      brand.createdAt ?? "",
      "",
    ),
    updatedAt: safeDateLabelJa(
      brand.updatedAt ?? "",
      "",
    ),
  }));
}