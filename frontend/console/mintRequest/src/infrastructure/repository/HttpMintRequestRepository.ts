// frontend/console/mintRequest/src/infrastructure/repository/HttpMintRequestRepository.ts

import type {
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
  postMintRequestHTTP,
} from "../repository";

export class HttpMintRequestRepository implements MintRequestRepository {
  async fetchInspectionByProductionId(
    productionId: string,
  ): Promise<unknown | null> {
    return await fetchInspectionByProductionIdHTTP(productionId).catch(
      () => null,
    );
  }

  async fetchMintByProductionId(
    productionId: string,
  ): Promise<unknown | null> {
    return await fetchMintByProductionIdHTTP(productionId).catch(() => null);
  }

  async fetchProductBlueprintIdByProductionId(
    productionId: string,
  ): Promise<string | null> {
    return await fetchProductBlueprintIdByProductionIdHTTP(productionId).catch(
      () => null,
    );
  }

  async fetchProductBlueprintPatch(
    productBlueprintId: string,
  ): Promise<unknown | null> {
    return await fetchProductBlueprintPatchHTTP(productBlueprintId).catch(
      () => null,
    );
  }

  async fetchBrandsForMint(): Promise<{ id: string; name: string }[]> {
    const brands = await fetchBrandsForMintHTTP().catch(() => []);

    return (brands ?? [])
      .map((b: any) => ({
        id: String(b?.id ?? "").trim(),
        name: String(b?.name ?? "").trim(),
      }))
      .filter((b: any) => b.id && b.name);
  }

  async fetchTokenBlueprintsByBrand(
    brandId: string,
  ): Promise<TokenBlueprintSummary[]> {
    const list = await fetchTokenBlueprintsByBrandHTTP(brandId).catch(
      () => [],
    );

    return (list ?? [])
      .map((tb: any) => {
        const tokenName = String(tb?.tokenName ?? tb?.name ?? "").trim();

        return {
          id: String(tb?.id ?? "").trim(),

          // selector 表示用
          name: tokenName,

          // TokenBlueprintCard 表示用
          tokenName,

          symbol: String(tb?.symbol ?? "").trim(),

          brandId: String(tb?.brandId ?? "").trim() || undefined,
          brandName: String(tb?.brandName ?? "").trim() || undefined,
          companyId: String(tb?.companyId ?? "").trim() || undefined,

          description: String(tb?.description ?? "").trim() || undefined,
          minted:
            typeof tb?.minted === "boolean"
              ? tb.minted
              : String(tb?.minted ?? "").trim().toLowerCase() === "true",
          metadataUri: String(tb?.metadataUri ?? "").trim() || undefined,

          iconUrl: String(tb?.iconUrl ?? "").trim() || undefined,
        };
      })
      .filter((tb: TokenBlueprintSummary) =>
        Boolean(tb.id && tb.name && tb.symbol),
      );
  }

  async postMintRequest(
    productionId: string,
    tokenBlueprintId: string,
    scheduledBurnDate?: string,
  ): Promise<unknown | null> {
    return await postMintRequestHTTP(
      productionId,
      tokenBlueprintId,
      scheduledBurnDate,
    ).catch(() => null);
  }
}