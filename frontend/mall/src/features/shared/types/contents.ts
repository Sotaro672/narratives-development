// frontend/amol/src/features/shared/types/contents.ts

import type { useTokenCommentCard } from "../../token-commnet/hooks/useTokenCommentCard";

export type ContentsMetadataFile = {
  type: string;
  uri: string;
};

export type ContentsMetadata = {
  name: string;
  image: string;
  description: string;
  files: ContentsMetadataFile[];
};

export type ContentsSearchParams = {
  assetId: string;
  productId: string;
  brandId: string;
  brandName: string;
  productName: string;
  productBlueprintId: string;
  tokenBlueprintId: string;
  metadataUri: string;
};

export type TokenCommentCardController = ReturnType<typeof useTokenCommentCard>;