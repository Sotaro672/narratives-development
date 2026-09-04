// frontend/amol/src/features/wallet/types/tokenTypes.ts

export type WalletStatus = "active" | "inactive";

export type WalletDTO = {
  walletAddress: string;
  assetIds: string[];
  lastUpdatedAt: string;
  status: WalletStatus;
};

export type TokenMetadataAttributeDTO = {
  traitType: string;
  value: string;
};

export type TokenMetadataDTO = {
  name: string;
  symbol: string;
  description: string;
  image: string;
  externalUrl: string;
  attributes: TokenMetadataAttributeDTO[];
  createdAt: string;
  tokenBlueprintId: string;
  raw: Record<string, unknown>;
};

export type TokenResolveDTO = {
  productId: string;
  brandId: string;
  brandName: string;
  productBlueprintId: string;
  productName: string;
  metadataUri: string;
  assetId: string;
};

export type WalletTokenItem = {
  assetId: string;
  productId: string;
  brandId: string;
  brandName: string;
  productName: string;
  productBlueprintId: string;
  tokenBlueprintId: string;
  metadataUri: string;
  metadata: TokenMetadataDTO | null;
};

export type WalletTokenListResult = {
  wallet: WalletDTO | null;
  tokens: WalletTokenItem[];
};