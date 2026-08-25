// frontend/amol/src/features/shared/types/scanResult.ts

import type { ProductBlueprintCategoryFields, ProductCategoryKind } from "./category";
import type { ProductBlueprintReviewPage } from "./review";

export type MallOwnerInfo = {
  walletAddress?: string;
  ownerType?: string;
  brandId?: string;
  avatarId?: string;
  brandName?: string;
  avatarName?: string;
};

export type MallPreviewTransferInfo = {
  transferredAt?: string;
  fromWalletAddress?: string;
  toWalletAddress?: string;
  fromAvatarId?: string;
  fromAvatarName?: string;
  fromAvatarIcon?: string;
  fromBrandId?: string;
  fromBrandName?: string;
  fromBrandIcon?: string;
  toAvatarId?: string;
  toAvatarName?: string;
  toAvatarIcon?: string;
  toBrandId?: string;
  toBrandName?: string;
  toBrandIcon?: string;
};

export type MallTokenInfo = {
  productId: string;
  brandId?: string;
  brandName?: string;
  tokenBlueprintId?: string;
  toAddress?: string;
  metadataUri?: string;
  assetId?: string;
  onChainTxSignature?: string;
  mintedAt?: string;
};

export type ProductBlueprintPatch = {
  productName?: string;
  description?: string;
  brandId?: string;
  brandName?: string;
  companyId?: string;
  companyName?: string;
  productBlueprintCategoryPath?: string[];
  categoryFields?: ProductBlueprintCategoryFields;
  productIdTag?: {
    Type?: string;
  };
  assigneeId?: string;
  modelRefs?: Array<{
    ModelID?: string;
    DisplayOrder?: number;
  }>;
};

export type CategoryInputFieldScope = "productBlueprint" | "model" | string;

export type CategoryInputFieldType =
  | "text"
  | "textarea"
  | "number"
  | "select"
  | "multiSelect"
  | "boolean"
  | "date"
  | string;

export type CategoryInputFieldDefinition = {
  scope: CategoryInputFieldScope;
  key: string;
  label: string;
  type: CategoryInputFieldType;
  required: boolean;
  unit?: string;
};

export type CategoryInputSchema = {
  categoryCode: string;
  categoryKind: ProductCategoryKind;
  categoryNameJa: string;
  productBlueprintFields: CategoryInputFieldDefinition[];
  modelFields: CategoryInputFieldDefinition[];
};

export type TokenBlueprintPatchVM = {
  id: string;
  tokenName: string;
  symbol: string;
  brandName: string;
  companyName: string;
  description: string;
  tokenIcon: string;
};

export type MallPreviewResponse = {
  productId: string;
  productBlueprintId: string;
  modelId: string;
  modelKind?: ProductCategoryKind;
  modelNumber: string;
  modelLabel?: string;
  size: string;
  color: string;
  rgb: number;
  measurements: Record<string, number> | null;
  volumeValue?: number | null;
  volumeUnit?: string;
  productBlueprintCategoryPath?: string[];
  categoryInputSchema?: CategoryInputSchema | null;
  productBlueprintPatch: ProductBlueprintPatch | null;
  brandName?: string;
  companyName?: string;
  token: MallTokenInfo | null;
  owner: MallOwnerInfo | null;
  transfers: MallPreviewTransferInfo[];
  tokenBlueprintPatch: TokenBlueprintPatchVM | null;
};

export type PreviewState = {
  raw: MallPreviewResponse;
  tokenIconUrlEncoded: string | null;
};

export type MallScanTransferResponse = {
  avatarId: string;
  productId: string;
  matched: boolean;
  matchedOrderId?: string;
  matchedItemIndex?: number;
  txSignature: string;
  fromDisplayName: string;
  toDisplayName: string;
  updatedToAddress: boolean;
  assetId: string;
};

export type ScanResultPageState = {
  productId: string;
  previewState: PreviewState | null;
  transferResult: MallScanTransferResponse | null;
  transferredAssetId: string;
  transferTxSignature: string;
  transferMatched: boolean;
  reviews: ProductBlueprintReviewPage | null;
  reviewsError: string | null;
  reviewPage: number;
  reviewPerPage: number;
  busyReviews: boolean;
  ownedByWallet: boolean | null;
  ownedByWalletError: string | null;
  busyOwnedByWallet: boolean;
  postingReview: boolean;
  postReviewError: string | null;
  loading: boolean;
  error: string | null;
  authAvailable: boolean;
  busyTransfer: boolean;
  transferError: string | null;
};