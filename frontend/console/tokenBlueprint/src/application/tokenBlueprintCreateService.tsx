// frontend/console/tokenBlueprint/src/application/tokenBlueprintCreateService.tsx

/**
 * TokenBlueprint 作成カードのアプリケーションサービス
 * - Brand 一覧取得
 * - brandId → brandName 解析
 * - （必要に応じて）TokenBlueprint 作成・更新処理もここに集約可能
 */

import {
  fetchBrandsForCurrentCompany,
  fetchBrandNameById,
} from "../infrastructure/repository/tokenBlueprintRepositoryHTTP";

// ---------------------------
// Brand 一覧取得
// ---------------------------
export async function loadBrandsForCompany(): Promise<
  { id: string; name: string }[]
> {
  try {
    const brands = await fetchBrandsForCurrentCompany();
    return brands;
  } catch (e) {
    console.error("[tokenBlueprintCreateService] loadBrandsForCompany error:", e);
    return [];
  }
}

// ---------------------------
// brandId → brandName 解決
// ---------------------------
export async function resolveBrandName(
  brandId: string,
): Promise<string> {
  try {
    const name = await fetchBrandNameById(brandId);
    return name ?? "";
  } catch (e) {
    console.error("[tokenBlueprintCreateService] resolveBrandName error:", e);
    return "";
  }
}
