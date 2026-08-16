// frontend/console/mintRequest/src/application/validator/validateCompleteInspection.ts

import type { InspectionBatch } from "../../../../shared/types/inspections";

export type ValidateCompleteInspectionInput = {
  inspectionBatch: InspectionBatch | null | undefined;
};

export type ValidateCompleteInspectionResult =
  | {
      ok: true;
      productionId: string;
    }
  | {
      ok: false;
      message: string;
    };

/**
 * 検品完了処理に必要な状態を検証する。
 *
 * Backend BFFが返すInspectionBatchを正とし、
 * productionIdはinspectionBatch.productionIdのみを使用する。
 *
 * Frontend側ではroute parameterへのfallback、
 * productionIdの再構築・normalizationを行わない。
 */
export function validateCompleteInspection(
  input: ValidateCompleteInspectionInput,
): ValidateCompleteInspectionResult {
  const inspectionBatch = input.inspectionBatch ?? null;

  if (!inspectionBatch) {
    return {
      ok: false,
      message: "検査バッチ情報が取得できていません。",
    };
  }

  if (!inspectionBatch.productionId) {
    return {
      ok: false,
      message: "productionId が特定できません。",
    };
  }

  return {
    ok: true,
    productionId: inspectionBatch.productionId,
  };
}