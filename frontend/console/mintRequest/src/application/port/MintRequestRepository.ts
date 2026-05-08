// frontend/console/mintRequest/src/application/port/MintRequestRepository.ts

export type BrandSummary = { id: string; name: string };
export type TokenBlueprintSummary = {
  id: string;
  name: string;
  symbol: string;
  iconUrl?: string;
};

export interface MintRequestRepository {
  // inspection / mint
  fetchInspectionByProductionId(productionId: string): Promise<unknown | null>;
  fetchMintByInspectionId(inspectionId: string): Promise<unknown | null>;

  // product blueprint
  fetchProductBlueprintIdByProductionId(productionId: string): Promise<string | null>;
  fetchProductBlueprintPatch(productBlueprintId: string): Promise<unknown | null>;

  // options
  fetchBrandsForMint(): Promise<BrandSummary[]>;
  fetchTokenBlueprintsByBrand(brandId: string): Promise<TokenBlueprintSummary[]>;

  // token blueprint patch（inventory等の別コンテキスト呼び出しは実装側に閉じ込める）
  fetchTokenBlueprintPatch(tokenBlueprintId: string): Promise<unknown | null>;

  // submit
  postMintRequest(
    productionId: string,
    tokenBlueprintId: string,
    scheduledBurnDate?: string,
  ): Promise<unknown | null>;
}
