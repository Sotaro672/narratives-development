// frontend/console/mintRequest/src/infrastructure/dto/mintRequestLocal.dto.ts

import type { InspectionBatchDTO } from "../../domain/entity/inspections";
import type { MintDTO } from "../api/mintRequestApi";

/**
 * ProductBlueprint.modelRefs 取得用 DTO
 * displayOrder は ProductBlueprint 側にのみ存在する前提のため、UI はこれを正として扱う。
 */
export type ProductBlueprintModelRefDTO = {
  modelId: string;
  displayOrder: number;
};

export type ProductBlueprintPatchDTO = {
  productName?: string | null;
  brandId?: string | null;
  brandName?: string | null;

  itemType?: string | null;
  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;

  productIdTag?: { type?: string | null; Type?: string | null } | null;

  assigneeId?: string | null;

  /**
   * displayOrder の唯一のソース（ProductBlueprint.modelRefs）
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

export type MintModelMetaEntryDTO = {
  modelNumber?: string | null;
  size?: string | null;
  colorName?: string | null;
  rgb?: number | null;
  volume?: string | number | null;
};

/**
 * MintRequest detail DTO
 * ✅ productionId を正とする。
 * ✅ inspectionId fallback は扱わない。
 */
export type MintRequestDetailDTO = {
  productionId?: string | null;

  inspection?: InspectionBatchDTO | null;

  mint?: MintDTO | null;

  productBlueprintPatch?: ProductBlueprintPatchDTO | null;

  modelMeta?: Record<string, MintModelMetaEntryDTO> | null;

  tokenBlueprintId?: string | null;
  productName?: string | null;
  tokenName?: string | null;

  productBlueprintId?: string | null;

  [k: string]: any;
};

export type ModelVariationForMintDTO = {
  id: string;
  modelNumber: string | null;
  size: string | null;
  colorName: string | null;
  rgb: number | null;

  /**
   * alcohol 対応:
   * model variation 側で容量も扱う。
   */
  volume?: string | number | null;
};