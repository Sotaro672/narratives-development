//frontend\console\brand\src\infrastructure\api\brandApi.ts
import type { Brand, BrandPatch } from "../../../../brand/src/domain/entity/brand";
import { brandRepositoryHTTP } from "../http/brandRepositoryHTTP";

/* ------------------------------------------------------------
 * Brand API compatibility wrapper
 * ------------------------------------------------------------ */

/**
 * GET /brands — 全ブランド一覧取得
 * 互換維持のための薄いラッパ
 */
export async function fetchAllBrands(): Promise<Brand[]> {
  const res = await brandRepositoryHTTP.list({
    page: 1,
    perPage: 1000,
  });
  return res.items;
}

/**
 * GET /brands/:id — ブランド取得
 */
export async function fetchBrandById(id: string): Promise<Brand> {
  return brandRepositoryHTTP.getById(id);
}

/**
 * POST /brands — ブランド新規作成
 * Payload: BrandPatch（id や timestamp は backend 側で採番）
 */
export async function createBrand(data: BrandPatch): Promise<Brand> {
  return brandRepositoryHTTP.create(
    data as Omit<Brand, "createdAt" | "updatedAt">
  );
}

/**
 * PATCH /brands/:id — 部分更新
 */
export async function updateBrand(
  id: string,
  patch: BrandPatch
): Promise<Brand> {
  return brandRepositoryHTTP.update(id, patch);
}

/**
 * DELETE /brands/:id — 論理削除
 */
export async function deleteBrand(id: string): Promise<void> {
  await brandRepositoryHTTP.delete(id);
}

/**
 * POST /brands/:id/activate — 有効化
 */
export async function activateBrandApi(id: string): Promise<Brand> {
  return brandRepositoryHTTP.activate(id);
}

/**
 * POST /brands/:id/deactivate — 無効化
 */
export async function deactivateBrandApi(id: string): Promise<Brand> {
  return brandRepositoryHTTP.deactivate(id);
}

/* ------------------------------------------------------------
 * Patch Utility
 * ------------------------------------------------------------ */

/**
 * BrandPatch を生成する小ユーティリティ
 */
export function toBrandPatch(p: Partial<Brand>): BrandPatch {
  return { ...p };
}