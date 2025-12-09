// frontend/console/mintRequest/src/application/mintRequestService.tsx

import type { InspectionBatchDTO } from "../infrastructure/api/mintRequestApi";
import {
  fetchInspectionBatchesHTTP,
  fetchProductBlueprintPatchHTTP,
  // ★ companyId ごとの Brand 一覧取得用
  fetchBrandsForMintHTTP,
  // ★ brandId ごとの TokenBlueprint 一覧取得用（/mint/token_blueprints?brandId=... を想定）
  fetchTokenBlueprintsByBrandHTTP,
  // ★ ミント申請 POST 用
  postMintRequestHTTP,
} from "../infrastructure/repository/mintRequestRepositoryHTTP";

/**
 * backend/internal/domain/productBlueprint.Patch に対応する DTO
 */
export type ProductBlueprintPatchDTO = {
  productName?: string | null;
  brandId?: string | null;

  // MintHandler (/mint/product_blueprints/{id}/patch) が付与するブランド名
  brandName?: string | null;

  itemType?: string | null; // Go 側 ItemType（"tops" / "bottoms" など）に対応
  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;
  productIdTag?: {
    type?: string | null;
  } | null;
  assigneeId?: string | null;
};

/**
 * backend/internal/domain/brand.Brand に対応する簡易 DTO
 * ListBrandByCompanyId（= /mint/brands）用
 */
export type BrandForMintDTO = {
  id: string;
  name: string;
};

/**
 * backend/internal/domain/tokenBlueprint.TokenBlueprint に対応する簡易 DTO
 * ListTokenBlueprintsByBrand（= /mint/token_blueprints?brandId=...）用
 *
 * - name: トークン名
 * - symbol: シンボル（例: LUMI）
 * - iconUrl: アイコン画像 URL（存在しない場合は undefined / 空文字）
 */
export type TokenBlueprintForMintDTO = {
  id: string;
  name: string;
  symbol: string;
  iconUrl?: string;
};

/**
 * MintUsecase 経由の /mint/inspections を叩き、
 * 指定された productionId に対応する InspectionBatchDTO を 1 件返す。
 *
 * ※ /mint/inspections は backend の MintUsecase
 *   （＝ GetModelVariationByID を含む処理）を経由する。
 */
export async function loadInspectionBatchFromMintAPI(
  productionId: string,
): Promise<InspectionBatchDTO | null> {
  const trimmed = productionId.trim();
  if (!trimmed) return null;

  // /mint/inspections を実行（Repository 経由）
  const batches = await fetchInspectionBatchesHTTP();

  // productionId で絞り込み
  const hit =
    batches.find((b) => (b as any).productionId === trimmed) ?? null;

  return hit;
}

/**
 * productBlueprintId から、MintUsecase.GetProductBlueprintPatchByID 経由で
 * ProductBlueprint Patch DTO を取得する。
 */
export async function loadProductBlueprintPatch(
  productBlueprintId: string,
): Promise<ProductBlueprintPatchDTO | null> {
  const trimmed = productBlueprintId.trim();
  if (!trimmed) return null;

  return await fetchProductBlueprintPatchHTTP(trimmed);
}

/**
 * current companyId に紐づく Brand 一覧を取得する。
 * backend/internal/application/usecase.MintUsecase.ListBrandsForCurrentCompany
 * （HTTP: GET /mint/brands）に対応。
 */
export async function loadBrandsForMint(): Promise<BrandForMintDTO[]> {
  const brands = await fetchBrandsForMintHTTP();
  return brands ?? [];
}

/**
 * 指定した brandId に紐づく TokenBlueprint 一覧を取得する。
 * backend/internal/application/usecase.MintUsecase.ListTokenBlueprintsByBrand
 * （HTTP: GET /mint/token_blueprints?brandId=...）に対応。
 *
 * 右カラムの「トークン設計カード」用に、
 * name / symbol / iconUrl を表示する前提。
 */
export async function loadTokenBlueprintsByBrand(
  brandId: string,
): Promise<TokenBlueprintForMintDTO[]> {
  const trimmed = brandId.trim();
  if (!trimmed) {
    return [];
  }

  const tokenBps = await fetchTokenBlueprintsByBrandHTTP(trimmed);
  return tokenBps ?? [];
}

/**
 * ミント申請詳細画面向けの TokenBlueprint を解決する。
 * 現状は個別 TokenBlueprint 詳細 API がないため undefined を返す。
 * 必要になれば backend の tokenBlueprint API を呼ぶ方式に置き換える。
 */
export function resolveBlueprintForMintRequest(requestId?: string) {
  return undefined;
}

/**
 * ★ ミント申請 POST 用サービス関数
 *
 * - productionId: 生産ID（inspectionBatch.productionId）
 * - tokenBlueprintId: 選択されたトークン設計ID
 * - scheduledBurnDate: 焼却予定日（"YYYY-MM-DD" 形式 / 任意）
 *
 * repository の postMintRequestHTTP に委譲する。
 */
export async function postMintRequest(
  productionId: string,
  tokenBlueprintId: string,
  scheduledBurnDate?: string,
) {
  const pid = productionId.trim();
  const tbid = tokenBlueprintId.trim();

  if (!pid || !tbid) {
    throw new Error("productionId / tokenBlueprintId is empty");
  }

  // scheduledBurnDate は undefined 許容でそのまま渡す（API 側で nullable 扱い）
  return await postMintRequestHTTP(pid, tbid, scheduledBurnDate);
}
