// frontend/console/mintRequest/src/presentation/di/mintRequestContainer.ts

import type {
  BrandSummary,
  MintRequestRepository,
  TokenBlueprintSummary,
} from "../../application/port/MintRequestRepository";

import {
  fetchInspectionByProductionIdHTTP,
  fetchMintByProductionIdHTTP,
  fetchProductBlueprintIdByProductionIdHTTP,
  fetchProductBlueprintPatchHTTP,
  fetchBrandsForMintHTTP,
  fetchTokenBlueprintsByBrandHTTP,
  fetchTokenBlueprintPatchHTTP,
  postMintRequestHTTP,
} from "../../infrastructure/repository";

class HttpMintRequestRepository implements MintRequestRepository {
  async fetchInspectionByProductionId(
    productionId: string,
  ): Promise<unknown | null> {
    return fetchInspectionByProductionIdHTTP(productionId);
  }

  /**
   * productions / inspections / mints の docId はすべて同一。
   * フロントでは productionId を正として扱う。
   */
  async fetchMintByProductionId(
    productionId: string,
  ): Promise<unknown | null> {
    return fetchMintByProductionIdHTTP(productionId);
  }

  async fetchProductBlueprintIdByProductionId(
    productionId: string,
  ): Promise<string | null> {
    return fetchProductBlueprintIdByProductionIdHTTP(productionId);
  }

  async fetchProductBlueprintPatch(
    productBlueprintId: string,
  ): Promise<unknown | null> {
    return fetchProductBlueprintPatchHTTP(productBlueprintId);
  }

  async fetchTokenBlueprintPatch(
    tokenBlueprintId: string,
  ): Promise<unknown | null> {
    return fetchTokenBlueprintPatchHTTP(tokenBlueprintId);
  }

  async fetchBrandsForMint(): Promise<BrandSummary[]> {
    return fetchBrandsForMintHTTP();
  }

  async fetchTokenBlueprintsByBrand(
    brandId: string,
  ): Promise<TokenBlueprintSummary[]> {
    return fetchTokenBlueprintsByBrandHTTP(brandId);
  }

  async postMintRequest(
    productionId: string,
    tokenBlueprintId: string,
    scheduledBurnDate?: string,
  ): Promise<unknown | null> {
    return postMintRequestHTTP(
      productionId,
      tokenBlueprintId,
      scheduledBurnDate,
    );
  }
}

export function mintRequestContainer(): {
  mintRequestRepo: MintRequestRepository;
} {
  return {
    mintRequestRepo: new HttpMintRequestRepository(),
  };
}