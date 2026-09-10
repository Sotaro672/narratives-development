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