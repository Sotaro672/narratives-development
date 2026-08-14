// frontend/console/shell/src/features/mint/application/port/MintRequestRepository.ts

import type { MintRequestManagementRowDTO } from "../../infrastructure/dto/mintRequestManagementRow";
import type {
  MintProductBlueprintDTO,
  MintRequestDetailDTO,
} from "../../infrastructure/dto/mintRequestLocal.dto";

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
 * SOL見積時のFee Payer情報。
 */
export type MintFundingEstimateFeePayer = {
  address: string;
  balanceLamports: string;
  balanceSol: number;
  targetLamports: string;
  targetSol: number;
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
 * 初回のMerkle Tree / Core Collection作成費とmintQuantity件分のMint transaction feeを扱う。
 */
export type MintFundingEstimateCosts = {
  mintTransactionFeePerItemLamports: string;
  mintTransactionFeePerItemSol: number;
  mintTransactionFeeTotalLamports: string;
  mintTransactionFeeTotalSol: number;
  merkleTreeCreationTransactionFeeLamports: string;
  merkleTreeCreationTransactionFeeSol: number;
  merkleTreeCreationRentLamports: string;
  merkleTreeCreationRentSol: number;
  merkleTreeCreationCostLamports: string;
  merkleTreeCreationCostSol: number;
  coreCollectionCreationTransactionFeeLamports: string;
  coreCollectionCreationTransactionFeeSol: number;
  coreCollectionCreationRentLamports: string;
  coreCollectionCreationRentSol: number;
  coreCollectionCreationCostLamports: string;
  coreCollectionCreationCostSol: number;
  provisioningCostLamports: string;
  provisioningCostSol: number;
  estimatedNetworkCostLamports: string;
  estimatedNetworkCostSol: number;
  requiredFeePayerBalanceLamports: string;
  requiredFeePayerBalanceSol: number;
  estimatedReserveTopUpLamports: string;
  estimatedReserveTopUpSol: number;
  reserveTransferFeeBufferLamports: string;
  reserveTransferFeeBufferSol: number;
  requiredReserveForTopUpLamports: string;
  requiredReserveForTopUpSol: number;
  sufficient: boolean;
};

/**
 * GET /mint/funding-estimate のレスポンス。
 */
export type MintFundingEstimate = {
  cluster: string;
  mintQuantity: number;
  reserve: MintFundingEstimateReserve;
  feePayer: MintFundingEstimateFeePayer;
  resources: MintFundingEstimateResources;
  estimate: MintFundingEstimateCosts;
};

export interface MintRequestRepository {
  /**
   * productionIdに紐づくMint詳細情報を取得する。
   *
   * GET /mint/inspections/{productionId} のBackend BFF responseをそのまま返す。
   * productBlueprintId / productName / modelMeta / inspectionをFrontend側で再構築しない。
   */
  fetchMintRequestDetail(
    productionId: string,
  ): Promise<MintRequestDetailDTO | null>;

  /**
   * productionIdに紐づくMint管理情報を取得する。
   * GET /mint/requests?productionIds={productionId} の対象rowを正とする。
   */
  fetchMintRequestRowByProductionId(
    productionId: string,
  ): Promise<MintRequestManagementRowDTO | null>;

  /**
   * productBlueprintIdに紐づくミント画面用ProductBlueprint情報を取得する。
   * GET /mint/product_blueprints/{productBlueprintId} のBackend BFF responseを正とする。
   */
  fetchMintProductBlueprint(
    productBlueprintId: string,
  ): Promise<MintProductBlueprintDTO | null>;

  /**
   * ミント申請画面で選択可能なブランド一覧を取得する。
   */
  fetchBrandsForMint(): Promise<BrandSummary[]>;

  /**
   * 指定したブランドに紐づくToken Blueprint一覧を取得する。
   */
  fetchTokenBlueprintsByBrand(
    brandId: string,
  ): Promise<TokenBlueprintSummary[]>;

  /**
   * productionIdとtokenBlueprintIdからBubblegum V2 MintのSOL見積を取得する。
   *
   * metadataUriはFrontendから渡さない。
   * mintQuantity、Brand Wallet、TokenBlueprint情報はBackend側で解決する。
   */
  fetchMintFundingEstimate(
    productionId: string,
    tokenBlueprintId: string,
  ): Promise<MintFundingEstimate>;
}