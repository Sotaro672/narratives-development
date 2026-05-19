// frontend/console/mintRequest/src/application/port/MintRequestRepository.ts

export type BrandSummary = {
  id: string;
  name: string;
};

export type TokenBlueprintSummary = {
  id: string;

  /**
   * selector 表示用。
   *
   * backend response の正は tokenName だが、
   * 既存 UI は name を表示用 field として使うため保持する。
   */
  name: string;

  /**
   * TokenBlueprintCard 表示用。
   *
   * GET /mint/token_blueprints?brandId=... の tokenName を保持する。
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

  // token blueprint
  fetchTokenBlueprintPatch(
    tokenBlueprintId: string,
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
  ): Promise<unknown | null>;
}