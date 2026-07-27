// frontend/console/shell/src/features/mintRequest/infrastructure/repository/HttpMintRequestRepository.ts

import type {
  BrandSummary,
  MintRequestRepository,
  TokenBlueprintSummary,
} from "../../application/port/MintRequestRepository";

import type { InspectionBatchDTO } from "../../domain/inspections";

import type { MintDTO } from "../dto/mint.dto";
import type { ProductBlueprintPatchDTO } from "../dto/mintRequestLocal.dto";

import { fetchBrandsForMintHTTP } from "./http/brands";
import { fetchInspectionByProductionIdHTTP } from "./http/inspections";
import { fetchMintByProductionIdHTTP } from "./http/mintRequests";
import {
  fetchProductBlueprintIdByProductionIdHTTP,
} from "./http/productions";
import { fetchProductBlueprintPatchHTTP } from "./http/productBlueprintPatch";
import { fetchTokenBlueprintsByBrandHTTP } from "./http/tokenBlueprints";

/**
 * MintRequestRepositoryのHTTP実装。
 *
 * HTTPレスポンスの正規化は各HTTP関数側で行い、
 * このクラスは取得処理をApplication層のPortへ接続する。
 *
 * ミント申請の送信はsubmitMintRequestAndRefreshから
 * postMintRequestHTTPを直接呼び出すため、このRepositoryでは扱わない。
 */
export class HttpMintRequestRepository
  implements MintRequestRepository
{
  /**
   * productionIdに紐づく検品バッチを取得する。
   */
  fetchInspectionByProductionId(
    productionId: string,
  ): Promise<InspectionBatchDTO | null> {
    return fetchInspectionByProductionIdHTTP(
      productionId,
    ).catch(() => null);
  }

  /**
   * productionIdに紐づくMint情報を取得する。
   *
   * productions、inspections、mintsの
   * ドキュメントIDは同一であり、
   * フロントエンドではproductionIdを正とする。
   */
  fetchMintByProductionId(
    productionId: string,
  ): Promise<MintDTO | null> {
    return fetchMintByProductionIdHTTP(
      productionId,
    ).catch(() => null);
  }

  /**
   * productionIdに紐づくproductBlueprintIdを取得する。
   */
  fetchProductBlueprintIdByProductionId(
    productionId: string,
  ): Promise<string | null> {
    return fetchProductBlueprintIdByProductionIdHTTP(
      productionId,
    ).catch(() => null);
  }

  /**
   * productBlueprintIdに紐づく
   * プロダクト設計情報を取得する。
   */
  fetchProductBlueprintPatch(
    productBlueprintId: string,
  ): Promise<ProductBlueprintPatchDTO | null> {
    return fetchProductBlueprintPatchHTTP(
      productBlueprintId,
    ).catch(() => null);
  }

  /**
   * ミント申請画面で選択可能なブランドを取得する。
   */
  fetchBrandsForMint(): Promise<BrandSummary[]> {
    return fetchBrandsForMintHTTP().catch(
      () => [],
    );
  }

  /**
   * 指定ブランドに紐づくトークン設計を取得する。
   */
  fetchTokenBlueprintsByBrand(
    brandId: string,
  ): Promise<TokenBlueprintSummary[]> {
    return fetchTokenBlueprintsByBrandHTTP(
      brandId,
    ).catch(() => []);
  }
}