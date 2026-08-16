// frontend/console/shell/src/features/brand/application/brandService.ts
/// <reference types="vite/client" />

import { brandRepositoryHTTP } from "../infrastructure/http/brandRepositoryHTTP";
import { safeDateTimeLabelJa } from "../../../shared/util/dateJa";

export type BrandRow = {
  id: string;
  name: string;
  isActive: boolean;
  managerId: string | null;
  memberName: string;
  registeredAt: string;
  updatedAt: string;
};

// ブランド一覧取得
// - BackendにはpageとperPageのみを送る
// - Backend側で認証中ユーザーのcompanyIdに絞り込む
// - BackendのBrandレスポンスを正として使用する
// - presentation用の日時表示のみ変換する
export async function listBrands(): Promise<BrandRow[]> {
  const page = await brandRepositoryHTTP.list({
    page: 1,
    perPage: 200,
  });

  return page.items.map((brand) => ({
    id: brand.id,
    name: brand.name,
    isActive: brand.isActive,
    managerId: brand.managerId ?? null,
    memberName: brand.memberName ?? "",
    registeredAt: safeDateTimeLabelJa(brand.createdAt, ""),
    updatedAt: safeDateTimeLabelJa(brand.updatedAt ?? "", ""),
  }));
}