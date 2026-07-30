// frontend/console/shell/src/features/mint/infrastructure/dto/mintRequestLocal.dto.ts

import type {
  InspectionBatchDTO,
} from "../../../../shared/types/inspections";

import type {
  ProductIDTag,
} from "../../../../shared/types/productBlueprint";

import type {
  MintDTO,
} from "./mint.dto";

import type {
  ProductBlueprintCategorySnapshot,
  CategoryFieldValues,
} from "../../../productBlueprint/domain/productBlueprintCategory";

/**
 * ProductBlueprint.modelRefs取得用DTO。
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
   */
  categoryFields?:
    | CategoryFieldValues
    | null;

  /**
   * 商品へ付与する識別タグ。
   *
   * shared/types/productBlueprint.tsの
   * ProductIDTagを正規型として使用する。
   *
   * フィールド名はtypeのみを使用し、
   * 旧形式のTypeは保持しない。
   */
  productIdTag?:
    | ProductIDTag
    | null;

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

/**
 * BackendのModelVariationKind。
 */
export type ModelVariationKindForMint =
  | "apparel"
  | "alcohol";

/**
 * Backendのmodel.Volumeで許可されている単位。
 */
export type ModelVolumeUnitForMint =
  | "ml"
  | "L";

/**
 * apparel variationのカラー情報。
 */
export type ModelColorForMintDTO = {
  name: string;
  rgb: number;
};

/**
 * alcohol variationの容量情報。
 */
export type ModelVolumeForMintDTO = {
  value: number;
  unit: ModelVolumeUnitForMint;
};

/**
 * GET /models/{id}で返される共通フィールド。
 */
type ModelVariationBaseForMintDTO = {
  id: string;
  productBlueprintId: string;
  modelNumber: string;

  createdAt?: string;
  createdBy?: string;
  updatedAt?: string;
  updatedBy?: string;
};

/**
 * apparel用のModel variation。
 */
export type ApparelModelVariationForMintDTO =
  ModelVariationBaseForMintDTO & {
    kind: "apparel";

    size: string;
    color: ModelColorForMintDTO;
    measurements?: Record<string, number>;

    volume?: never;
  };

/**
 * alcohol用のModel variation。
 */
export type AlcoholModelVariationForMintDTO =
  ModelVariationBaseForMintDTO & {
    kind: "alcohol";

    volume: ModelVolumeForMintDTO;

    size?: never;
    color?: never;
    measurements?: never;
  };

/**
 * GET /models/{id}の正規レスポンス。
 *
 * kindを判別キーとしてapparelとalcoholを区別する。
 */
export type ModelVariationForMintDTO =
  | ApparelModelVariationForMintDTO
  | AlcoholModelVariationForMintDTO;

/**
 * ミント申請詳細画面で使用するモデル表示情報。
 *
 * modelIdをキーとしたRecord内で使用するが、
 * BackendのMintModelMetaEntryにもmodelIdが含まれるため保持する。
 *
 * volumeとvolumeUnitは、
 * GET /models/{id}によるalcoholモデル補完用フィールド。
 */
export type MintModelMetaEntryDTO = {
  modelId: string;

  modelNumber?: string;
  size?: string;
  colorName?: string;
  rgb?: number;

  volume?: number;
  volumeUnit?: ModelVolumeUnitForMint;
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