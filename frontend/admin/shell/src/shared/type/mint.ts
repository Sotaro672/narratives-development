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
  brandId: string;
  tokenBlueprintId: string;
  products: string[];
  status: MintStatus;
  createdAt: string;
  createdBy: string;
  requestedBy?: string;
  mintedAt?: string;
  scheduledBurnDate?: string;
  onChainTxSignature?: string;
};

export type MintListResponse = {
  items: Mint[];
  totalCount: number;
};