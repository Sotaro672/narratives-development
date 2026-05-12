//frontend\amol\src\features\contents\types.ts
import type { useTokenCommentCard } from "../token-commnet/hooks/useTokenCommentCard";

export type ContentsMetadataFile = {
  name: string;
  type: string;
  uri: string;
};

export type ContentsMetadata = {
  name: string;
  symbol: string;
  description: string;
  image: string;
  createdAt: string;
  files: ContentsMetadataFile[];
};

export type ContentsSearchParams = {
  mintAddress: string;
  productId: string;
  brandId: string;
  brandName: string;
  productName: string;
  productBlueprintId: string;
  tokenBlueprintId: string;
  metadataUri: string;
  tokenName: string;
  tokenIconUrl: string;
};

export type TokenCommentCardController = ReturnType<typeof useTokenCommentCard>;