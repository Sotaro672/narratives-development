// frontend/console/shell/src/features/mintRequest/infrastructure/repository/HttpMintRequestRepository.ts

import type {
  BrandSummary,
  MintFundingEstimate,
  MintQueuedResponse,
  MintRequestRepository,
  TokenBlueprintSummary,
} from "../../application/port/MintRequestRepository";

import type {
  InspectionBatchDTO,
} from "../../../../shared/types/inspections";

import type {
  MintRequestManagementRowDTO,
} from "../dto/mintRequestManagementRow";

import type {
  ProductBlueprintPatchDTO,
} from "../dto/mintRequestLocal.dto";

import {
  fetchBrandsForMintHTTP,
} from "./http/brands";

import {
  completeInspectionHTTP,
  fetchInspectionByProductionIdHTTP,
} from "./http/inspections";

import {
  fetchMintFundingEstimateHTTP,
  fetchMintRequestRowByProductionIdHTTP,
  postMintRequestHTTP,
} from "./http/mintRequests";

import {
  fetchProductBlueprintPatchHTTP,
} from "./http/productBlueprintPatch";

import {
  fetchTokenBlueprintsByBrandHTTP,
} from "./http/tokenBlueprints";

/**
 * MintRequestRepositoryのHTTP実装。
 *
 * Backend BFF responseを正とし、
 * Infrastructure層では不要なfallbackや
 * response fieldの補完を行わない。
 *
 * HTTPエラーは握りつぶさず、
 * 呼び出し元へそのまま伝播する。
 *
 * ProductBlueprint情報は
 * GET /mint/product_blueprints/{productBlueprintId}、
 * modelMetaは
 * GET /mint/inspections/{productionId}
 * のresponseを正とする。
 *
 * productionIdからproductBlueprintIdを取得するための
 * productions APIへの追加アクセスは行わない。
 */
export class HttpMintRequestRepository
  implements MintRequestRepository
{
  /**
   * productionIdに紐づく検品情報を取得する。
   *
   * productBlueprintId / productName / modelMetaも
   * 同じBFF responseから取得する。
   */
  fetchInspectionByProductionId(
    productionId: string,
  ): Promise<InspectionBatchDTO | null> {
    return fetchInspectionByProductionIdHTTP(
      productionId,
    );
  }

  /**
   * productionIdに紐づくMint管理情報を取得する。
   *
   * GET /mint/requests?productionIds={productionId}&view=management
   * の対象rowを正とする。
   */
  fetchMintRequestRowByProductionId(
    productionId: string,
  ): Promise<MintRequestManagementRowDTO | null> {
    return fetchMintRequestRowByProductionIdHTTP(
      productionId,
    );
  }

  /**
   * productBlueprintIdに紐づく
   * ミント画面用ProductBlueprint情報を取得する。
   */
  fetchProductBlueprintPatch(
    productBlueprintId: string,
  ): Promise<ProductBlueprintPatchDTO | null> {
    return fetchProductBlueprintPatchHTTP(
      productBlueprintId,
    );
  }

  /**
   * ミント申請画面で選択可能なブランド一覧を取得する。
   */
  fetchBrandsForMint(): Promise<BrandSummary[]> {
    return fetchBrandsForMintHTTP();
  }

  /**
   * 指定ブランドに紐づく
   * Token Blueprint一覧を取得する。
   */
  fetchTokenBlueprintsByBrand(
    brandId: string,
  ): Promise<TokenBlueprintSummary[]> {
    return fetchTokenBlueprintsByBrandHTTP(
      brandId,
    );
  }

  /**
   * productionIdとtokenBlueprintIdから
   * Bubblegum V2 Mintに必要なSOL見積を取得する。
   *
   * metadataUriはFrontendから送信しない。
   * 見積取得エラーは呼び出し元へ伝播する。
   */
  fetchMintFundingEstimate(
    productionId: string,
    tokenBlueprintId: string,
  ): Promise<MintFundingEstimate> {
    return fetchMintFundingEstimateHTTP(
      productionId,
      tokenBlueprintId,
    );
  }

  /**
   * productionIdに紐づく検品を完了する。
   *
   * completeMintInspection UseCaseから呼び出される。
   * HTTPエラーは呼び出し元へ伝播する。
   */
  completeInspection(
    productionId: string,
  ): Promise<InspectionBatchDTO | null> {
    return completeInspectionHTTP(
      productionId,
    );
  }

  /**
   * Mint申請をBackendへ送信する。
   *
   * Backendは202 AcceptedとQUEUED responseを返し、
   * Mint処理を非同期で順次実行する。
   *
   * scheduledBurnDateはFrontendから送信しない。
   * HTTPエラーは呼び出し元へ伝播する。
   */
  postMintRequest(
    productionId: string,
    tokenBlueprintId: string,
  ): Promise<MintQueuedResponse | null> {
    return postMintRequestHTTP(
      productionId,
      tokenBlueprintId,
    );
  }
}