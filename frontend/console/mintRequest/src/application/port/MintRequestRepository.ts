// frontend/console/mintRequest/src/application/port/MintRequestRepository.ts

export type BrandSummary = {
  id: string;
  name: string;
};

export type TokenBlueprintSummary = {
  id: string;

  /**
   * 右側の「トークン設計一覧」表示用。
   *
   * backend response の正は tokenName だが、
   * 既存 UI では name を表示用 field として使う。
   */
  name: string;

  /**
   * TokenBlueprintCard 表示用。
   */
  tokenName?: string;

  symbol: string;

  brandId?: string;
  brandName?: string;
  companyId?: string;

  description?: string;
  minted?: boolean;
  metadataUri?: string;

  iconUrl?: string;
};

export type MintTaskProgress = {
  total: number;
  pending: number;
  minting: number;
  minted: number;
  failedRetryable: number;
  failedFatal: number;
  percentage: number;
};

export type MintQueuedResponse = {
  mintRequestId: string;
  productionId: string;
  status: "QUEUED";
  message: string;
};

export interface MintRequestRepository {
  // inspection / mint
  // productions / inspections / mints の docId はすべて同一で、フロントでは productionId を正として扱う
  fetchInspectionByProductionId(productionId: string): Promise<unknown | null>;
  fetchMintByProductionId(productionId: string): Promise<unknown | null>;

  // product blueprint
  fetchProductBlueprintIdByProductionId(
    productionId: string,
  ): Promise<string | null>;

  fetchProductBlueprintPatch(
    productBlueprintId: string,
  ): Promise<unknown | null>;

  // options
  fetchBrandsForMint(): Promise<BrandSummary[]>;

  fetchTokenBlueprintsByBrand(
    brandId: string,
  ): Promise<TokenBlueprintSummary[]>;

  // submit
  postMintRequest(
    productionId: string,
    tokenBlueprintId: string,
    scheduledBurnDate?: string,
  ): Promise<MintQueuedResponse | null>;
}