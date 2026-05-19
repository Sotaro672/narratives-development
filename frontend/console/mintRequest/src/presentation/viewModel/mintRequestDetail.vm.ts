// frontend/console/mintRequest/src/presentation/viewModel/mintRequestDetail.vm.ts

// ============================================================
// ViewModel Types for MintRequestDetail (Detail Screen)
// ============================================================

import type {
  CategoryFieldValues,
  ProductBlueprintCategoryKind,
  ProductBlueprintCategorySnapshot,
} from "../../../../productBlueprint/src/domain/entity/productBlueprintCategory";

export type BrandOptionVM = {
  id: string;
  name: string;
};

export type TokenBlueprintOptionVM = {
  id: string;

  /**
   * 右側の「トークン設計一覧」表示用。
   *
   * backend response の正は tokenName だが、
   * selector 側では name を表示用 field として使う。
   */
  name: string;

  /**
   * TokenBlueprintCard 表示用。
   *
   * GET /mint/token_blueprints?brandId=... の tokenName を保持する。
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

export type ProductBlueprintCategoryFieldRowVM = {
  label: string;
  value: string;
};

export type ProductBlueprintCardVM = {
  productName?: string;
  brand?: string; // 表示用（brandName のみ）

  /**
   * 商品カテゴリ snapshot。
   *
   * ProductBlueprintCard は categoryName ではなく
   * productBlueprintCategory / productBlueprintPatch.productBlueprintCategory を参照して
   * 商品カテゴリを表示するため、ここで保持して page 側から渡す。
   */
  productBlueprintCategory?: ProductBlueprintCategorySnapshot | null;

  /**
   * 旧 itemType は廃止。
   * 表示本体は productBlueprintCategory を正とする。
   *
   * categoryName / categoryCode / categoryKind は、
   * mintRequest 側で補助表示・条件分岐が必要な場合の派生値。
   */
  categoryName?: string;
  categoryCode?: string;
  categoryKind?: ProductBlueprintCategoryKind | string;

  /**
   * categoryFields の raw 値。
   * 表示用には categoryFieldRows を優先する。
   */
  categoryFields?: CategoryFieldValues | null;

  /**
   * categoryFields を UI 表示しやすい label/value に変換したもの。
   *
   * alcohol の例:
   * - ヴィンテージ: 2020
   * - 地域: 福島
   * - 原材料: 山田錦
   * - アルコール度数: 78%
   */
  categoryFieldRows?: ProductBlueprintCategoryFieldRowVM[];

  productIdTag?: string;
};

export type TokenBlueprintCardVM = {
  id: string;

  /**
   * TokenBlueprintCard の表示名。
   *
   * backend response の tokenName を優先し、
   * fallback として name を使う。
   * id は表示名 fallback に使わない。
   */
  name: string;

  /**
   * backend response の正フィールド。
   */
  tokenName?: string;

  symbol: string;

  // brandId は UI 表示に使わせない（揺れ防止）
  brandId: string;

  // UI 表示は brandName のみに統一
  brandName: string;

  companyId?: string;

  description: string;

  minted?: boolean;
  metadataUri?: string;

  iconUrl?: string;

  isEditMode: boolean;

  brandOptions: BrandOptionVM[];
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
   * alcohol 対応:
   * model variation 側で容量も扱う。
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
   * alcohol 対応:
   * model variation 側で容量も扱う。
   */
  volume?: string | number | null;

  passedCount: number;
  totalCount: number;
};

/**
 * 詳細画面 ViewModel
 * - batch / mint は “raw DTO” を UI に晒さず、必要な情報を VM として束ねる
 * - 必要なら batchRaw / mintRaw を optional で保持してもよいが、まずは最小限
 */
export type MintRequestDetailVM = {
  requestId: string;

  // key refs（画面内の data fetch / submit 用）
  productionId: string;
  productBlueprintId: string | null;

  // cards
  productBlueprintCard: ProductBlueprintCardVM | null;
  tokenBlueprintCard: TokenBlueprintCardVM | null;

  // mint
  mintInfo: MintInfoVM | null;

  // options
  brandOptions: BrandOptionVM[];
  tokenBlueprintOptions: TokenBlueprintOptionVM[];

  // inspections (model aggregate)
  modelRows: ModelInspectionRowVM[];

  // token blueprint patch（inventory 側などからの追加表示用）
  tokenBlueprintPatchRaw?: any | null;
};