// frontend/console/shell/src/features/brand/application/BrandSelectionService.ts

import { brandRepositoryHTTP } from "../infrastructure/http/brandRepositoryHTTP";

export type BrandOption = {
  id: string;
  name: string;
};

/**
 * 現在ログイン中MemberのcompanyIdに属する有効ブランドを取得する。
 *
 * companyIdはFrontendから送信せず、
 * Backendの認証コンテキストを正とする。
 *
 * ブランド選択ではisActive=trueのみを候補とする。
 */
export async function fetchActiveBrandOptionsForCurrentCompany(): Promise<BrandOption[]> {
  const result = await brandRepositoryHTTP.list({
    page: 1,
    perPage: 200,
  });

  return result.items
    .filter((brand) => brand.isActive)
    .map((brand) => ({
      id: brand.id,
      name: brand.name,
    }));
}