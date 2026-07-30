// frontend/console/shell/src/features/mintRequest/application/usecase/resolveInspectionModelMeta.ts

import {
  getInspectionModelIds,
  getMissingModelIds,
  type InspectionBatchForCard,
} from "../mapper/buildInspectionResultCardData";

import {
  toMintModelMetaEntry,
} from "../mapper/modelVariationMapper";

import type {
  MintModelMetaEntryDTO,
  ModelVariationForMintDTO,
} from "../../infrastructure/dto/mintRequestLocal.dto";

/**
 * 不足しているModel Variationを取得するための
 * Application層側の契約。
 *
 * Infrastructure層のHTTP関数をPresentation層から
 * 直接呼ばないために使用する。
 */
export interface InspectionModelMetaRepository {
  fetchModelVariationByIdForMint(
    modelId: string,
  ): Promise<
    ModelVariationForMintDTO | null
  >;
}

export type ResolveInspectionModelMetaInput = {
  /**
   * 検品結果カード構築用のバッチ。
   */
  batch:
    | InspectionBatchForCard
    | null
    | undefined;

  /**
   * これまでにAPIから補完済みのmodelMeta。
   *
   * batch.modelMetaとは分けて保持し、
   * Presentation層のstateとして使用する。
   */
  resolvedMeta?:
    | Record<
        string,
        MintModelMetaEntryDTO
      >
    | null;
};

export type ResolveInspectionModelMetaResult =
  Record<
    string,
    MintModelMetaEntryDTO
  >;

/**
 * Inspection内で参照されているmodelIdのうち、
 * modelMetaが存在しないものだけを取得する。
 *
 * 個別モデルの取得失敗は画面全体の取得失敗にはせず、
 * 取得できたモデルだけを返す。
 */
export async function resolveInspectionModelMeta(
  repository: InspectionModelMetaRepository,
  input: ResolveInspectionModelMetaInput,
): Promise<ResolveInspectionModelMetaResult> {
  const {
    batch,
    resolvedMeta,
  } = input;

  const currentResolvedMeta = {
    ...(resolvedMeta ?? {}),
  };

  if (!batch) {
    return currentResolvedMeta;
  }

  const modelIds =
    getInspectionModelIds(
      batch,
    );

  const mergedModelMeta: Record<
    string,
    MintModelMetaEntryDTO
  > = {
    ...(batch.modelMeta ?? {}),
    ...currentResolvedMeta,
  };

  const missingModelIds =
    getMissingModelIds({
      modelIds,
      modelMeta:
        mergedModelMeta,
    });

  if (
    missingModelIds.length === 0
  ) {
    return currentResolvedMeta;
  }

  const settled =
    await Promise.all(
      missingModelIds.map(
        async (modelId) => {
          try {
            const variation =
              await repository
                .fetchModelVariationByIdForMint(
                  modelId,
                );

            return {
              modelId,
              variation,
            };
          } catch {
            return {
              modelId,
              variation:
                null as
                  | ModelVariationForMintDTO
                  | null,
            };
          }
        },
      ),
    );

  const nextResolvedMeta: Record<
    string,
    MintModelMetaEntryDTO
  > = {
    ...currentResolvedMeta,
  };

  for (const item of settled) {
    const meta =
      toMintModelMetaEntry(
        item.variation,
      );

    if (!meta) {
      continue;
    }

    nextResolvedMeta[
      item.modelId
    ] = {
      ...meta,
      modelId:
        item.modelId,
    };
  }

  return nextResolvedMeta;
}