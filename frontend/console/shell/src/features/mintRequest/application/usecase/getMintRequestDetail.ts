// frontend/console/shell/src/features/mintRequest/application/usecase/getMintRequestDetail.ts

import {
  extractMintInfoFromBatch,
  extractMintInfoFromMintDTO,
} from "../mapper/mintInfoMapper";
import type { MintRequestRepository } from "../port/MintRequestRepository";
import { asNonEmptyString } from "../util/primitive";

async function resolveProductBlueprintId(
  repo: MintRequestRepository,
  productionId: string,
): Promise<string> {
  const productBlueprintId =
    await repo.fetchProductBlueprintIdByProductionId(
      productionId,
    );

  return asNonEmptyString(
    productBlueprintId,
  );
}

export async function getMintRequestDetail(
  repo: MintRequestRepository,
  productionId: string,
) {
  const pid = String(
    productionId ?? "",
  ).trim();

  if (!pid) {
    return {
      inspectionBatch: null,
      mint: null,
      mintProgress: null,
      productBlueprintId: "",
    };
  }

  const [
    inspectionBatch,
    mintDTO,
    productBlueprintId,
  ] = await Promise.all([
    repo.fetchInspectionByProductionId(
      pid,
    ),
    repo.fetchMintByProductionId(
      pid,
    ),
    resolveProductBlueprintId(
      repo,
      pid,
    ),
  ]);

  /**
   * Infrastructure DTOからApplicationモデルへの変換は
   * Application層で完了させる。
   *
   * Mint取得APIの結果を優先し、
   * 存在しない場合のみinspection内のMint情報を使用する。
   */
  const mint =
    extractMintInfoFromMintDTO(
      mintDTO,
    ) ??
    extractMintInfoFromBatch(
      inspectionBatch,
    );

  return {
    inspectionBatch,
    mint,
    mintProgress:
      mintDTO?.mintProgress ?? null,
    productBlueprintId:
      productBlueprintId || "",
  };
}