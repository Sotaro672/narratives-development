// frontend/console/shell/src/features/mintRequest/application/port/MintRequestRepository.ts

import type { InspectionBatchDTO } from "../../domain/inspections";
import type { MintDTO } from "../../infrastructure/dto/mint.dto";
import type { ProductBlueprintPatchDTO } from "../../infrastructure/dto/mintRequestLocal.dto";

export type BrandSummary = {
  id: string;
  name: string;
};

export type TokenBlueprintSummary = {
  id: string;

  /**
   * 右側の「トークン設計一覧」表示用。
   *
   * Backend responseではtokenNameを正とするが、
   * 既存UIではnameを表示用フィールドとして使用する。
   */
  name: string;

  /**
   * TokenBlueprintCard表示用。
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

export type MintTaskProgress = {
  total: number;
  pending: number;
  minting: number;
  minted: number;
  failedRetryable: number;
  failedFatal: number;
  percentage: number;
};

/**
 * ミント申請がBackendに受理されたときのレスポンス。
 *
 * ミント処理の完了結果ではなく、
 * 非同期処理の開始受付結果を表す。
 */
export type MintQueuedResponse = {
  mintRequestId: string;
  productionId: string;
  status: "QUEUED";
  message: string;
};

export interface MintRequestRepository {
  /**
   * productionIdに紐づく検品バッチを取得する。
   *
   * productions、inspections、mintsのドキュメントIDは
   * すべて同一であり、フロントエンドではproductionIdを正とする。
   */
  fetchInspectionByProductionId(
    productionId: string,
  ): Promise<InspectionBatchDTO | null>;

  /**
   * productionIdに紐づくMint情報を取得する。
   */
  fetchMintByProductionId(
    productionId: string,
  ): Promise<MintDTO | null>;

  /**
   * productionIdに紐づくproductBlueprintIdを取得する。
   */
  fetchProductBlueprintIdByProductionId(
    productionId: string,
  ): Promise<string | null>;

  /**
   * productBlueprintIdに紐づく
   * プロダクト設計情報を取得する。
   */
  fetchProductBlueprintPatch(
    productBlueprintId: string,
  ): Promise<ProductBlueprintPatchDTO | null>;

  /**
   * ミント申請画面で選択可能なブランド一覧を取得する。
   */
  fetchBrandsForMint(): Promise<BrandSummary[]>;

  /**
   * 指定したブランドに紐づく
   * トークン設計一覧を取得する。
   */
  fetchTokenBlueprintsByBrand(
    brandId: string,
  ): Promise<TokenBlueprintSummary[]>;
}