// frontend/console/shell/src/features/mintRequest/infrastructure/dto/mintRequestLocal.dto.ts

import type { InspectionBatchDTO } from "../../domain/inspections";
import type { MintDTO } from "./mint.dto";

import type {
  ProductBlueprintCategorySnapshot,
  CategoryFieldValues,
} from "../../../productBlueprint/domain/productBlueprintCategory";

/**
 * ProductBlueprint.modelRefs 取得用DTO。
 *
 * displayOrderはProductBlueprint側にのみ存在する前提のため、
 * UIはこの値を正として扱う。
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
   * ProductBlueprint側にdenormalize保存される
   * カテゴリsnapshotを正とする。
   *
   * itemTypeは廃止し、カテゴリ判定には
   * productBlueprintCategoryを使用する。
   */
  productBlueprintCategory?:
    | ProductBlueprintCategorySnapshot
    | null;

  /**
   * カテゴリ別入力値。
   *
   * alcoholの例:
   * {
   *   vintage,
   *   region,
   *   material,
   *   alcoholContent
   * }
   */
  categoryFields?:
    | CategoryFieldValues
    | null;

  productIdTag?: {
    type?: string | null;
    Type?: string | null;
  } | null;

  assigneeId?: string | null;

  /**
   * displayOrderの唯一のソース。
   *
   * ProductBlueprint.modelRefsを正とする。
   */
  modelRefs?:
    | ProductBlueprintModelRefDTO[]
    | null;
};

export type TokenBlueprintForMintDTO = {
  id: string;

  /**
   * selector表示用。
   *
   * Backend responseの正はtokenNameだが、
   * 既存UIはnameを表示用fieldとして使用する。
   */
  name: string;

  /**
   * TokenBlueprintCard表示用。
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
   * alcohol対応。
   *
   * model variation側で容量と単位を扱う。
   *
   * 表示例:
   * - volume: 720
   * - volumeUnit: "ml"
   */
  volume?: string | number | null;
  volumeUnit?: string | null;
};

/**
 * MintRequest detail DTO。
 *
 * productionIdを正とする。
 * inspectionId fallbackは扱わない。
 */
export type MintRequestDetailDTO = {
  productionId?: string | null;

  inspection?: InspectionBatchDTO | null;

  mint?: MintDTO | null;

  productBlueprintPatch?:
    | ProductBlueprintPatchDTO
    | null;

  modelMeta?:
    | Record<
        string,
        MintModelMetaEntryDTO
      >
    | null;

  tokenBlueprintId?: string | null;
  productName?: string | null;
  tokenName?: string | null;

  productBlueprintId?: string | null;

  [key: string]: any;
};

export type ModelVariationForMintDTO = {
  id: string;
  modelNumber: string | null;
  size: string | null;
  colorName: string | null;
  rgb: number | null;

  /**
   * alcohol対応。
   *
   * model variation側で容量と単位を扱う。
   *
   * Backend・mapper側ではvolumeUnitへ正規化する。
   * 例: "ml", "L"
   */
  volume?: string | number | null;
  volumeUnit?: string | null;
};