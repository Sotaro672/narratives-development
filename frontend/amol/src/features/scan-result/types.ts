//frontend\amol\src\features\scan-result\types.ts
export type MallOwnerInfo = {
  brandId: string;
  avatarId: string;
  brandName: string;
  avatarName: string;
};

export type MallModelTokenPair = {
  modelId: string;
  tokenBlueprintId: string;
};

export type MallScanVerifyResponse = {
  avatarId: string;
  productId: string;
  scannedModelId: string;
  scannedTokenBlueprintId: string;
  purchasedPairs: MallModelTokenPair[];
  matched: boolean;
  match: MallModelTokenPair | null;
};

export type MallPreviewTransferInfo = {
  transferredAt: string | null;

  fromWalletAddress: string;
  toWalletAddress: string;

  fromAvatarId: string;
  fromAvatarName: string;
  fromAvatarIcon: string;
  fromBrandId: string;
  fromBrandName: string;
  fromBrandIcon: string;

  toAvatarId: string;
  toAvatarName: string;
  toAvatarIcon: string;
  toBrandId: string;
  toBrandName: string;
  toBrandIcon: string;
};

export type MallTokenInfo = {
  productId: string;
  brandId: string;
  brandName: string;
  tokenBlueprintId: string;
  toAddress: string;
  metadataUri: string;
  mintAddress: string;
  onChainTxSignature: string;
  mintedAt: string;
};

export type MallPreviewResponse = {
  productId: string;
  productBlueprintId: string;
  modelId: string;
  modelNumber: string;
  size: string;
  color: string;
  rgb: number;
  measurements: Record<string, number> | null;
  productBlueprintPatch: Record<string, unknown> | null;
  token: MallTokenInfo | null;
  owner: MallOwnerInfo | null;
  transfers: MallPreviewTransferInfo[];
};

export type ProductBlueprintPatchItem = {
  key: string;
  label: string;
  value: string;
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

export type PreviewState = {
  raw: MallPreviewResponse;
  tokenBlueprintPatch: TokenBlueprintPatchVM | null;
  tokenIconUrlEncoded: string | null;
};

export type MallTransferFlowStep = {
  no: number;
  title: string;
  note: string;
};

export type MallScanTransferResponse = {
  avatarId: string;
  productId: string;
  matched: boolean;
  txSignature: string;
  fromWallet: string;
  toWallet: string;
  updatedToAddress: boolean;
  mintAddress: string;
  flow: MallTransferFlowStep[];
};

export type CatalogReview = {
  id: string;
  productBlueprintId: string;
  avatarId: string;
  avatarName: string;
  avatarIcon: string;
  rating: number;
  title: string;
  body: string;
  helpfulVotes: number;
  totalVotes: number;
  reviewedAt: string;
};

export type CatalogReviewPage = {
  items: CatalogReview[];
  page: number;
  perPage: number;
  total: number;
  hasNext: boolean;
};

export type TokenContentFile = {
  id: string;
  name: string;
  viewUri: string;
  contentType: string;
  isPreviewable: boolean;
};

export type TokenResolveDTO = {
  mintAddress: string;
  tokenContentsFiles: TokenContentFile[];
};

export type WalletDTO = {
  tokens: string[];
};

export type ScanResultPageState = {
  productId: string;
  previewState: PreviewState | null;
  meAvatar: MallOwnerInfo | null;
  verifyResult: MallScanVerifyResponse | null;
  transferResult: MallScanTransferResponse | null;
  transferredMintAddress: string;
  transferTxSignature: string;
  transferMatched: boolean;

  reviews: CatalogReviewPage | null;
  reviewsError: string | null;
  reviewPage: number;
  reviewPerPage: number;
  busyReviews: boolean;

  ownedByWallet: boolean | null;
  ownedByWalletError: string | null;
  busyOwnedByWallet: boolean;

  postingReview: boolean;
  postReviewError: string | null;

  resolvingTransferredToken: boolean;
  resolvedTransferredToken: TokenResolveDTO | null;

  loading: boolean;
  error: string | null;
  authAvailable: boolean;
  busyTransfer: boolean;
};