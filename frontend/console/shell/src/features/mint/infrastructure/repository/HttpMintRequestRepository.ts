// frontend/console/shell/src/features/mintRequest/infrastructure/repository/HttpMintRequestRepository.ts

import type {
  BrandSummary,
  MintQueuedResponse,
  MintRequestRepository,
  TokenBlueprintSummary,
} from "../../application/port/MintRequestRepository";

import type {
  InspectionBatchDTO,
} from "../../../../shared/types/inspections";

import type {
  MintDTO,
} from "../dto/mint.dto";

import type {
  ModelVariationForMintDTO,
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
  fetchMintByProductionIdHTTP,
  postMintRequestHTTP,
} from "./http/mintRequests";

import {
  fetchModelVariationByIdForMintHTTP,
} from "./http/modelVariations";

import {
  fetchProductBlueprintIdByProductionIdHTTP,
} from "./http/productions";

import {
  fetchProductBlueprintPatchHTTP,
} from "./http/productBlueprintPatch";

import {
  fetchTokenBlueprintsByBrandHTTP,
} from "./http/tokenBlueprints";

/**
 * MintRequestRepositoryのHTTP実装。
 *
 * HTTPレスポンスの正規化は各HTTP関数側で行い、
 * このクラスはInfrastructure層のHTTP処理を
 * Application層のRepository契約へ接続する。
 *
 * 参照系の取得失敗は既存画面との互換性を維持するため、
 * nullまたは空配列へ変換する。
 *
 * Model Variationの取得、検品完了、Mint申請では、
 * Application層でエラー処理方針を判断できるよう、
 * HTTPエラーを握りつぶさず呼び出し元へ伝播する。
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

  /**
   * modelIdに紐づくModel Variationを取得する。
   *
   * resolveInspectionModelMeta UseCaseから呼び出される。
   *
   * 個別取得失敗を継続可能として扱うかどうかは
   * Application層で判断するため、
   * HTTPエラーはここでは握りつぶさない。
   */
  fetchModelVariationByIdForMint(
    modelId: string,
  ): Promise<ModelVariationForMintDTO | null> {
    return fetchModelVariationByIdForMintHTTP(
      modelId,
    );
  }

  /**
   * productionIdに紐づく検品を完了する。
   *
   * completeMintInspection UseCaseから呼び出される。
   * エラーはPresentation層で表示できるように伝播させる。
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
   * Backendは202 AcceptedとQUEUEDレスポンスを返し、
   * Mint処理を非同期で順次実行する。
   *
   * エラーはPresentation層で表示できるように伝播させる。
   */
  postMintRequest(
    productionId: string,
    tokenBlueprintId: string,
    scheduledBurnDate?: string,
  ): Promise<MintQueuedResponse | null> {
    return postMintRequestHTTP(
      productionId,
      tokenBlueprintId,
      scheduledBurnDate,
    );
  }
}