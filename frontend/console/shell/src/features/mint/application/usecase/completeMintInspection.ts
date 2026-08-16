// frontend/console/shell/src/features/mintRequest/application/usecase/completeMintInspection.ts

import type { InspectionBatch } from "../../../../shared/types/inspections";
import { validateCompleteInspection } from "../validator/validateCompleteInspection";

/**
 * 検品完了処理に必要なRepository契約。
 *
 * /products/inspections/complete のBackend responseである
 * InspectionBatchをそのまま扱う。
 */
export interface CompleteMintInspectionRepository {
  completeInspection(productionId: string): Promise<InspectionBatch | null>;
}

export type CompleteMintInspectionInput = {
  inspectionBatch: InspectionBatch | null | undefined;
};

export type CompleteMintInspectionResult =
  | {
      ok: true;
      productionId: string;
      inspectionBatch: InspectionBatch | null;
    }
  | {
      ok: false;
      message: string;
    };

/**
 * 検品完了条件の検証とRepository呼び出しを行う。
 *
 * productionIdはBackend BFFが返すinspectionBatch.productionIdを正とし、
 * Frontend側ではroute parameterへのfallbackや再構築を行わない。
 *
 * confirm、alert、loading state、画面再取得はPresentation層の責務とする。
 * Repositoryから発生した通信エラーは握りつぶさず、呼び出し元へthrowする。
 *
 * Backendが返すInspectionBatchをFrontend独自DTOへ再構築しない。
 */
export async function completeMintInspection(
  repository: CompleteMintInspectionRepository,
  input: CompleteMintInspectionInput,
): Promise<CompleteMintInspectionResult> {
  const validation = validateCompleteInspection({
    inspectionBatch: input.inspectionBatch,
  });

  if (!validation.ok) {
    return {
      ok: false,
      message: validation.message,
    };
  }

  const inspectionBatch = await repository.completeInspection(validation.productionId);

  return {
    ok: true,
    productionId: validation.productionId,
    inspectionBatch,
  };
}