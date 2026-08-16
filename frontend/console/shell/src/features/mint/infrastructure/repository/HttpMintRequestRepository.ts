// frontend/console/shell/src/features/mint/infrastructure/repository/HttpMintRequestRepository.ts

import type {
  BrandSummary,
  MintFundingEstimate,
  MintQueuedResponse,
  MintRequestRepository,
  TokenBlueprintSummary,
} from "../../application/port/MintRequestRepository";

import type { InspectionBatch } from "../../../../shared/types/inspections";
import type { MintRequestManagementRowDTO } from "../dto/mintRequestManagementRow";
import type {
  MintProductBlueprintDTO,
  MintRequestDetailDTO,
} from "../dto/mintRequestLocal.dto";

import { fetchBrandsForMintHTTP } from "./http/brands";
import {
  completeInspectionHTTP,
  fetchMintRequestDetailHTTP,
} from "./http/inspections";
import {
  fetchMintFundingEstimateHTTP,
  fetchMintRequestRowByProductionIdHTTP,
  postMintRequestHTTP,
} from "./http/mintRequests";
import { fetchMintProductBlueprintHTTP } from "./http/mintProductBlueprint";
import { fetchTokenBlueprintsByBrandHTTP } from "./http/tokenBlueprints";

/**
 * MintRequestRepositoryのHTTP実装。
 *
 * Backend BFF responseを正とし、Infrastructure層では不要なfallbackやresponse fieldの補完を行わない。
 * HTTPエラーは握りつぶさず、呼び出し元へそのまま伝播する。
 *
 * Mint詳細情報はGET /mint/inspections/{productionId}、
 * ProductBlueprint情報はGET /mint/product_blueprints/{productBlueprintId}のresponseを正とする。
 *
 * productionIdからproductBlueprintIdを取得するためのproductions APIへの追加アクセスは行わない。
 */
export class HttpMintRequestRepository implements MintRequestRepository {
  /**
   * productionIdに紐づくMint詳細情報を取得する。
   *
   * GET /mint/inspections/{productionId} のBackend BFF responseをそのまま返す。
   * productBlueprintId / productName / modelMeta / inspectionをFrontend側で再構築しない。
   */
  fetchMintRequestDetail(productionId: string): Promise<MintRequestDetailDTO | null> {
    return fetchMintRequestDetailHTTP(productionId);
  }

  /**
   * productionIdに紐づくMint管理情報を取得する。
   * GET /mint/requests?productionIds={productionId} の対象rowを正とする。
   */
  fetchMintRequestRowByProductionId(
    productionId: string,
  ): Promise<MintRequestManagementRowDTO | null> {
    return fetchMintRequestRowByProductionIdHTTP(productionId);
  }

  /**
   * productBlueprintIdに紐づくミント画面用ProductBlueprint情報を取得する。
   * GET /mint/product_blueprints/{productBlueprintId} のBackend BFF responseをそのまま返す。
   */
  fetchMintProductBlueprint(
    productBlueprintId: string,
  ): Promise<MintProductBlueprintDTO | null> {
    return fetchMintProductBlueprintHTTP(productBlueprintId);
  }

  /** ミント申請画面で選択可能なブランド一覧を取得する。 */
  fetchBrandsForMint(): Promise<BrandSummary[]> {
    return fetchBrandsForMintHTTP();
  }

  /** 指定ブランドに紐づくToken Blueprint一覧を取得する。 */
  fetchTokenBlueprintsByBrand(brandId: string): Promise<TokenBlueprintSummary[]> {
    return fetchTokenBlueprintsByBrandHTTP(brandId);
  }

  /**
   * productionIdとtokenBlueprintIdからBubblegum V2 Mintに必要なSOL見積を取得する。
   * metadataUriはFrontendから送信しない。
   * 見積取得エラーは呼び出し元へ伝播する。
   */
  fetchMintFundingEstimate(
    productionId: string,
    tokenBlueprintId: string,
  ): Promise<MintFundingEstimate> {
    return fetchMintFundingEstimateHTTP(productionId, tokenBlueprintId);
  }

  /**
   * productionIdに紐づく検品を完了する。
   *
   * /products/inspections/complete はMint detail BFFではなくCommand APIのため、
   * Backendが返すInspectionBatchをそのまま返す。
   * Frontend独自のInspectionBatchDTOへの再構築は行わない。
   */
  completeInspection(productionId: string): Promise<InspectionBatch | null> {
    return completeInspectionHTTP(productionId);
  }

  /**
   * Mint申請をBackendへ送信する。
   * Backendは202 AcceptedとQUEUED responseを返し、Mint処理を非同期で順次実行する。
   * scheduledBurnDateはFrontendから送信しない。
   * HTTPエラーは呼び出し元へ伝播する。
   */
  postMintRequest(
    productionId: string,
    tokenBlueprintId: string,
  ): Promise<MintQueuedResponse | null> {
    return postMintRequestHTTP(productionId, tokenBlueprintId);
  }
}