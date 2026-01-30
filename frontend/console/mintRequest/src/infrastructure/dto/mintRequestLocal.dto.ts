// frontend/console/mintRequest/src/infrastructure/dto/mintRequestLocal.dto.ts

import type { InspectionBatchDTO, MintDTO } from "../api/mintRequestApi";

/**
 * ★ NEW: ProductBlueprint.modelRefs 取得用 DTO
 * displayOrder は ProductBlueprint 側にのみ存在する前提のため、UI はこれを正として扱う。
 */
export type ProductBlueprintModelRefDTO = {
  modelId: string;
  displayOrder: number;
};

// ✅ ここで DTO を定義して循環/参照エラーを避ける
export type ProductBlueprintPatchDTO = {
  productName?: string | null;
  brandId?: string | null;
  brandName?: string | null;

  itemType?: string | null;
  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;

  // ✅ normalize で最終的に { type } に揃える（受け取りは Type / type 両対応）
  productIdTag?: { type?: string | null; Type?: string | null } | null;

  assigneeId?: string | null;

  /**
   * ★ NEW: displayOrder の唯一のソース（ProductBlueprint.modelRefs）
   * - patch レスポンスに modelRefs が含まれる前提で、ここで取得する
   */
  modelRefs?: ProductBlueprintModelRefDTO[] | null;
};

export type BrandForMintDTO = {
  id: string;
  name: string;
};

export type TokenBlueprintForMintDTO = {
  id: string;
  name: string;
  symbol: string;
  iconUrl?: string;
};

// ★ NEW: /mint/inspections/{productionId} の detail DTO（バックエンド返却差異に強くするため緩め）
export type MintModelMetaEntryDTO = {
  modelNumber?: string | null;
  size?: string | null;
  colorName?: string | null;
  rgb?: number | null;
};

/**
 * MintRequest detail DTO
 * - displayOrder は ProductBlueprintPatchDTO.modelRefs を正として利用する
 * - modelMeta には displayOrder を持たせない（model 側に displayOrder が存在しない前提のため）
 */
export type MintRequestDetailDTO = {
  // id / productionId / inspectionId など揺れる可能性があるため任意
  productionId?: string | null;
  inspectionId?: string | null;

  // inspection batch（または同等）
  inspection?: InspectionBatchDTO | null;

  // mint（存在すれば）
  mint?: MintDTO | null;

  // product blueprint patch（存在すれば）
  productBlueprintPatch?: ProductBlueprintPatchDTO | null;

  // model variations -> modelMeta（存在すれば）
  // modelId をキーに、modelNumber/size/color/rgb を保持
  modelMeta?: Record<string, MintModelMetaEntryDTO> | null;

  // 主要フィールド（detail の揺れ吸収用）
  tokenBlueprintId?: string | null;
  productName?: string | null;
  tokenName?: string | null;

  /**
   * 可能なら detail で productBlueprintId も受け取る
   * - patch を叩くために必要
   */
  productBlueprintId?: string | null;

  // その他バックエンド側が返すフィールドを落とさない
  [k: string]: any;
};

export type ModelVariationForMintDTO = {
  id: string;
  modelNumber: string | null;
  size: string | null;
  colorName: string | null;
  rgb: number | null;
};
