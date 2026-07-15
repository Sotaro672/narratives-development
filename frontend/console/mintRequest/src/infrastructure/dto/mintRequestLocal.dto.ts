// frontend/console/mintRequest/src/infrastructure/dto/mintRequestLocal.dto.ts

import type { InspectionBatchDTO } from "../../domain/inspections";
import type { MintDTO } from "../api/mintRequestApi";

import type {
  ProductBlueprintCategorySnapshot,
  CategoryFieldValues,
} from "../../../../productBlueprint/src/domain/productBlueprintCategory";

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
  description?: string | null;

  brandId?: string | null;
  brandName?: string | null;
  companyId?: string | null;

  /**
   * 商品カテゴリ。
   *
   * ProductBlueprint 側に denormalize 保存されるカテゴリ snapshot を正とする。
   * itemType は廃止し、カテゴリ判定はこの productBlueprintCategory を使う。
   */
  productBlueprintCategory?: ProductBlueprintCategorySnapshot | null;

  /**
   * カテゴリ別入力値。
   *
   * alcohol の例:
   * {
   *   vintage,
   *   region,
   *   material,
   *   alcoholContent
   * }
   */
  categoryFields?: CategoryFieldValues | null;

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

  /**
   * selector 表示用。
   *
   * backend response の正は tokenName だが、
   * 既存 UI は name を表示用 field として使う。
   */
  name: string;

  /**
   * TokenBlueprintCard 表示用。
   */
  tokenName?: string;

  symbol: string;

  brandId?: string;
  brandName?: string;
  companyId?: string;

  description?: string;
  minted?: boolean;
  metadataUri?: string;

  iconUrl?: string;
};

export type MintModelMetaEntryDTO = {
  modelNumber?: string | null;
  size?: string | null;
  colorName?: string | null;
  rgb?: number | null;

  /**
   * alcohol 対応:
   * model variation 側で容量と単位を扱う。
   *
   * 表示例:
   * - volume: 720
   * - volumeUnit: "ml"
   */
  volume?: string | number | null;
  volumeUnit?: string | null;
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
   * model variation 側で容量と単位を扱う。
   *
   * backend / mapper 側では volumeUnit に正規化する。
   * 例: "ml", "L"
   */
  volume?: string | number | null;
  volumeUnit?: string | null;
};