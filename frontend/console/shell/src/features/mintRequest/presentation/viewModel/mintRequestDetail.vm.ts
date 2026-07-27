// frontend/console/shell/src/features/mintRequest/presentation/viewModel/mintRequestDetail.vm.ts

// ============================================================
// ViewModel Types for MintRequestDetail
// ============================================================

import type {
  BrandSummary,
  TokenBlueprintSummary,
} from "../../application/port/MintRequestRepository";

import type {
  CategoryFieldValues,
  ProductBlueprintCategoryKind,
  ProductBlueprintCategorySnapshot,
} from "../../../productBlueprint/domain/productBlueprintCategory";

export type ProductBlueprintCategoryFieldRowVM = {
  label: string;
  value: string;
};

export type ProductBlueprintCardVM = {
  productName?: string;

  /**
   * ブランドの表示名。
   *
   * IDや別名は保持せず、
   * UI表示ではbrandNameのみを使用する。
   */
  brandName?: string;

  /**
   * 商品カテゴリsnapshot。
   *
   * ProductBlueprintCardはcategoryNameではなく、
   * productBlueprintCategoryまたは
   * productBlueprintPatch.productBlueprintCategoryを参照して
   * 商品カテゴリを表示する。
   */
  productBlueprintCategory?:
    | ProductBlueprintCategorySnapshot
    | null;

  /**
   * 旧itemTypeは廃止。
   *
   * 表示本体はproductBlueprintCategoryを正とする。
   *
   * categoryName、categoryCode、categoryKindは、
   * MintRequest側で補助表示や条件分岐が必要な場合の派生値。
   */
  categoryName?: string;
  categoryCode?: string;
  categoryKind?:
    | ProductBlueprintCategoryKind
    | string;

  /**
   * categoryFieldsのraw値。
   *
   * 表示用にはcategoryFieldRowsを優先する。
   */
  categoryFields?:
    | CategoryFieldValues
    | null;

  /**
   * categoryFieldsをUI表示用の
   * label/valueへ変換した値。
   *
   * alcoholの例:
   * - ヴィンテージ: 2020
   * - 地域: 福島
   * - 原材料: 山田錦
   * - アルコール度数: 78%
   */
  categoryFieldRows?:
    ProductBlueprintCategoryFieldRowVM[];

  productIdTag?: string;
};

export type TokenBlueprintCardVM = {
  id: string;

  /**
   * トークン名。
   *
   * MintRequest内ではtokenNameを正とし、
   * nameは使用しない。
   */
  tokenName: string;

  symbol: string;

  /**
   * brandIdはUI表示には使用しない。
   */
  brandId: string;

  /**
   * UI表示ではbrandNameのみを使用する。
   */
  brandName: string;

  description: string;
  iconUrl?: string;

  isEditMode: boolean;

  /**
   * ブランド候補はBrandSummaryへ統一する。
   */
  brandOptions: BrandSummary[];
};

export type TokenBlueprintCardHandlersVM = {
  onPreview: () => void;
};

export type MintInfoVM = {
  id: string;

  brandId: string;
  tokenBlueprintId: string;

  createdBy: string;
  createdByName?: string | null;
  createdAt: string | null;
  requestedByName?: string | null;

  minted: boolean;
  mintedAt?: string | null;
  onChainTxSignature?: string | null;
  scheduledBurnDate?: string | null;
};

export type MintModelMetaEntryVM = {
  modelNumber?: string | null;
  size?: string | null;
  colorName?: string | null;
  rgb?: number | null;

  /**
   * alcohol対応。
   *
   * model variation側で容量も扱う。
   */
  volume?: string | number | null;
};

export type ModelInspectionRowVM = {
  modelId: string;

  modelNumber: string | null;
  size: string | null;
  colorName: string | null;
  rgb: number | null;

  /**
   * alcohol対応。
   *
   * model variation側で容量も扱う。
   */
  volume?: string | number | null;

  passedCount: number;
  totalCount: number;
};

/**
 * 詳細画面ViewModel。
 *
 * batchとmintのraw DTOをUIへ直接公開せず、
 * 画面に必要な情報をViewModelとして束ねる。
 */
export type MintRequestDetailVM = {
  requestId: string;

  /**
   * 画面内のデータ取得と送信用ID。
   */
  productionId: string;
  productBlueprintId: string | null;

  /**
   * カード表示用ViewModel。
   */
  productBlueprintCard:
    | ProductBlueprintCardVM
    | null;

  tokenBlueprintCard:
    | TokenBlueprintCardVM
    | null;

  /**
   * Mint情報。
   */
  mintInfo: MintInfoVM | null;

  /**
   * 選択候補。
   */
  brandOptions: BrandSummary[];

  tokenBlueprintOptions:
    TokenBlueprintSummary[];

  /**
   * モデル単位の検品集計。
   */
  modelRows: ModelInspectionRowVM[];

  /**
   * inventory側などから取得する
   * TokenBlueprint追加表示情報。
   */
  tokenBlueprintPatchRaw?: any | null;
};