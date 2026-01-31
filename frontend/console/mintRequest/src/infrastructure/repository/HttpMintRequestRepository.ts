// frontend/console/mintRequest/src/infrastructure/repository/HttpMintRequestRepository.ts

import type { MintRequestRepository } from "../../application/port/MintRequestRepository";

import {
  fetchInspectionByProductionIdHTTP,
  fetchMintByInspectionIdHTTP,
  fetchProductBlueprintIdByProductionIdHTTP,
  fetchProductBlueprintPatchHTTP,
  fetchBrandsForMintHTTP,
  fetchTokenBlueprintsByBrandHTTP,
  postMintRequestHTTP,
} from "../repository";

// inventory tokenBlueprintPatch（この “別コンテキストHTTP” は infra に閉じ込める）
import { fetchInventoryTokenBlueprintPatch } from "../adapter/inventoryTokenBlueprintPatch";

export class HttpMintRequestRepository implements MintRequestRepository {
  async fetchInspectionByProductionId(productionId: string): Promise<unknown | null> {
    return await fetchInspectionByProductionIdHTTP(productionId).catch(() => null);
  }

  async fetchMintByInspectionId(inspectionId: string): Promise<unknown | null> {
    return await fetchMintByInspectionIdHTTP(inspectionId).catch(() => null);
  }

  async fetchProductBlueprintIdByProductionId(productionId: string): Promise<string | null> {
    return await fetchProductBlueprintIdByProductionIdHTTP(productionId).catch(() => null);
  }

  async fetchProductBlueprintPatch(productBlueprintId: string): Promise<unknown | null> {
    return await fetchProductBlueprintPatchHTTP(productBlueprintId).catch(() => null);
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
  ): Promise<{ id: string; name: string; symbol: string; iconUrl?: string }[]> {
    const list = await fetchTokenBlueprintsByBrandHTTP(brandId).catch(() => []);
    return (list ?? [])
      .map((tb: any) => ({
        id: String(tb?.id ?? "").trim(),
        name: String(tb?.name ?? "").trim(),
        symbol: String(tb?.symbol ?? "").trim(),
        iconUrl: String(tb?.iconUrl ?? "").trim() || undefined,
      }))
      .filter((tb: any) => tb.id && tb.name && tb.symbol);
  }

  async fetchTokenBlueprintPatch(tokenBlueprintId: string): Promise<unknown | null> {
    // ここで inventory 側 endpoint を吸収（application は知らない）
    return await fetchInventoryTokenBlueprintPatch(tokenBlueprintId).catch(() => null);
  }

  async postMintRequest(
    productionId: string,
    tokenBlueprintId: string,
    scheduledBurnDate?: string,
  ): Promise<unknown | null> {
    return await postMintRequestHTTP(productionId, tokenBlueprintId, scheduledBurnDate).catch(
      () => null,
    );
  }
}
