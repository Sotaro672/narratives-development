// frontend/admin/shell/src/shared/type/mint.ts

export type MintStatus =
  | "CREATED"
  | "QUEUED"
  | "MINTING"
  | "PARTIALLY_MINTED"
  | "MINTED"
  | "FAILED_RETRYABLE"
  | "FAILED_FATAL";

export type Mint = {
  id: string;
  brandName: string;
  tokenBlueprintName: string;
  productCount: number;
  status: MintStatus;
  requestedByName?: string;
  mintedAt?: string;
  scheduledBurnDate?: string;
};

export type MintListResponse = {
  items: Mint[];
  totalCount: number;
};

export type MintDetailModel = {
  modelId: string;
  kind: string;
  modelNumber: string;
  size?: string;
  colorName?: string;
  rgb?: number;
  measurements?: Record<string, number>;
  volume?: number;
  volumeUnit?: string;
  productCount: number;
};

export type MintDetail = {
  id: string;
  companyId: string;
  companyName: string;
  tokenBlueprintId: string;
  tokenBrandName: string;
  tokenName: string;
  productBlueprintId: string;
  productBrandName: string;
  productName: string;
  models: MintDetailModel[];
};