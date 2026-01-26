// frontend/console/mintRequest/src/presentation/viewModel/mintRequestDetail.vm.ts

// ============================================================
// ViewModel Types for MintRequestDetail (Detail Screen)
// ============================================================

export type BrandOptionVM = {
  id: string;
  name: string;
};

export type TokenBlueprintOptionVM = {
  id: string;
  name: string;
  symbol: string;
  iconUrl?: string;
};

export type ProductBlueprintCardVM = {
  productName?: string;
  brand?: string; // 表示用（brandName のみ）
  itemType?: string;
  fit?: string;
  materials?: string;
  weight?: number;
  washTags?: string[];
  productIdTag?: string;
};

export type TokenBlueprintCardVM = {
  id: string;
  name: string;
  symbol: string;

  // brandId は UI 表示に使わせない（揺れ防止）
  brandId: string;

  // UI 表示は brandName のみに統一
  brandName: string;

  description: string;
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
};

export type ModelInspectionRowVM = {
  modelId: string;

  modelNumber: string | null;
  size: string | null;
  colorName: string | null;
  rgb: number | null;

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
