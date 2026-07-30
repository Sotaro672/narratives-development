// frontend/console/shell/src/features/mintRequest/application/usecase/completeMintInspection.ts

import type {
  InspectionBatchDTO,
} from "../../domain/inspections";

import {
  validateCompleteInspection,
} from "../validator/validateCompleteInspection";

/**
 * 検品完了処理に必要なRepository契約。
 */
export interface CompleteMintInspectionRepository {
  completeInspection(
    productionId: string,
  ): Promise<
    InspectionBatchDTO | null
  >;
}

export type CompleteMintInspectionInput = {
  inspectionBatch:
    | InspectionBatchDTO
    | null
    | undefined;

  /**
   * URLパラメータ由来のproductionId。
   *
   * inspectionBatch.productionIdが存在する場合は、
   * validator側でそちらを優先する。
   */
  productionId?:
    | string
    | null;
};

export type CompleteMintInspectionResult =
  | {
      ok: true;
      productionId: string;
      inspectionBatch:
        | InspectionBatchDTO
        | null;
    }
  | {
      ok: false;
      message: string;
    };

/**
 * 検品完了条件の検証とRepository呼び出しを行う。
 *
 * confirm、alert、loading state、画面再取得は
 * Presentation層の責務とする。
 *
 * Repositoryから発生した通信エラーは握りつぶさず、
 * 呼び出し元へthrowする。
 */
export async function completeMintInspection(
  repository: CompleteMintInspectionRepository,
  input: CompleteMintInspectionInput,
): Promise<CompleteMintInspectionResult> {
  const validation =
    validateCompleteInspection({
      inspectionBatch:
        input.inspectionBatch,
      productionId:
        input.productionId,
    });

  if (!validation.ok) {
    return {
      ok: false,
      message:
        validation.message,
    };
  }

  const inspectionBatch =
    await repository.completeInspection(
      validation.productionId,
    );

  return {
    ok: true,
    productionId:
      validation.productionId,
    inspectionBatch,
  };
}