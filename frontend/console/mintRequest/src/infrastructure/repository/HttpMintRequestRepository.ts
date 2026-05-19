// frontend/console/mintRequest/src/infrastructure/repository/HttpMintRequestRepository.ts

import type { MintRequestRepository } from "../../application/port/MintRequestRepository";

import {
  fetchInspectionByProductionIdHTTP,
  fetchMintByProductionIdHTTP,
  fetchProductBlueprintIdByProductionIdHTTP,
  fetchProductBlueprintPatchHTTP,
  fetchBrandsForMintHTTP,
  fetchTokenBlueprintsByBrandHTTP,
  fetchTokenBlueprintPatchHTTP,
  postMintRequestHTTP,
} from "../repository";

const toText = (value: unknown): string => {
  return typeof value === "string" ? value.trim() : "";
};

const toOptionalText = (value: unknown): string | undefined => {
  const text = toText(value);
  return text || undefined;
};

const toBool = (value: unknown): boolean | undefined => {
  if (typeof value === "boolean") return value;

  if (typeof value === "string") {
    const normalized = value.trim().toLowerCase();
    if (normalized === "true") return true;
    if (normalized === "false") return false;
  }

  return undefined;
};

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

  async fetchTokenBlueprintPatch(
    tokenBlueprintId: string,
  ): Promise<unknown | null> {
    return await fetchTokenBlueprintPatchHTTP(tokenBlueprintId).catch(
      () => null,
    );
  }

  async fetchBrandsForMint(): Promise<{ id: string; name: string }[]> {
    const brands = await fetchBrandsForMintHTTP().catch(() => []);

    return (brands ?? [])
      .map((b: any) => ({
        id: toText(b?.id),
        name: toText(b?.name),
      }))
      .filter((b: any) => b.id && b.name);
  }

  async fetchTokenBlueprintsByBrand(
    brandId: string,
  ): Promise<
    {
      id: string;
      name: string;
      tokenName?: string;
      symbol: string;
      brandId?: string;
      brandName?: string;
      companyId?: string;
      description?: string;
      minted?: boolean;
      metadataUri?: string;
      iconUrl?: string;
    }[]
  > {
    const list = await fetchTokenBlueprintsByBrandHTTP(brandId).catch(
      () => [],
    );

    return (list ?? [])
      .map((tb: any) => {
        const tokenName = toText(tb?.tokenName) || toText(tb?.name);

        return {
          id: toText(tb?.id),

          // selector 表示用
          name: tokenName,

          // TokenBlueprintCard 表示用
          tokenName,

          symbol: toText(tb?.symbol),

          brandId: toOptionalText(tb?.brandId),
          brandName: toOptionalText(tb?.brandName),
          companyId: toOptionalText(tb?.companyId),
          description: toOptionalText(tb?.description),
          minted: toBool(tb?.minted),
          metadataUri: toOptionalText(tb?.metadataUri),

          iconUrl: toOptionalText(tb?.iconUrl),
        };
      })
      .filter((tb: any) => tb.id && tb.name && tb.symbol);
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