// frontend/console/shell/src/features/mint/application/port/MintRequestRepository.ts

export type BrandSummary = {
  id: string;
  name: string;
};

export type TokenBlueprintSummary = {
  id: string;
  tokenName: string;
  symbol: string;
  brandId?: string;
  description?: string;
  minted: boolean;
  iconUrl?: string;
};

/**
 * ミント申請がBackendに受理されたときのレスポンス。
 * ミント処理の完了結果ではなく、非同期処理の開始受付結果を表す。
 */
export type MintQueuedResponse = {
  mintRequestId: string;
  productionId: string;
  status: "QUEUED";
  message: string;
};

/**
 * SOL見積時のReserve Wallet情報。
 */
export type MintFundingEstimateReserve = {
  address: string;
  balanceLamports: string;
  balanceSol: number;
  minimumLamports: string;
  minimumSol: number;
};

/**
 * Mintに利用するSolanaリソースの現在状態。
 */
export type MintFundingEstimateResources = {
  sharedMerkleTreeExists: boolean;
  sharedMerkleTreeAddress: string | null;
  coreCollectionExists: boolean;
  coreCollectionAddress: string | null;
};

/**
 * Bubblegum V2 Mintに必要なSOL費用見積。
 *
 * metadataUriやGCS上のコンテンツ容量には依存せず、
 * mint transaction feeとShared Merkle Tree / Core Collectionの初回作成費を扱う。
 *
 * initialCreationCost:
 * - Shared Merkle Tree初回作成費
 * - Core Collection初回作成費
 * の合計。
 *
 * totalRequired:
 * - Mint手数料合計
 * - initialCreationCost
 * の合計。
 */
export type MintFundingEstimateCosts = {
  mintTransactionFeePerItemLamports: string;
  mintTransactionFeePerItemSol: number;
  mintTransactionFeeTotalLamports: string;
  mintTransactionFeeTotalSol: number;
  initialCreationCostLamports: string;
  initialCreationCostSol: number;
  totalRequiredLamports: string;
  totalRequiredSol: number;
  sufficient: boolean;
};

/**
 * GET /mint/funding-estimate のBackend response。
 *
 * mintQuantityやFee Payer情報、Reserve補充予定などの
 * funding policy内部値はFrontendへ公開しない。
 */
export type MintFundingEstimate = {
  cluster: string;
  reserve: MintFundingEstimateReserve;
  resources: MintFundingEstimateResources;
  estimate: MintFundingEstimateCosts;
};