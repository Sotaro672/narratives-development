// frontend/amol/src/features/wallet/types/tokenTypes.ts

export type WalletDTO = {
  walletAddress: string;
  tokens: string[];
  lastUpdatedAt: string | null;
  status: string;
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
  metadataUri: string;
  mintAddress: string;
  productBlueprintId: string;
  brandName: string;
  productName: string;
  tokenBlueprintId: string;
};

export type WalletTokenItem = {
  mintAddress: string;
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