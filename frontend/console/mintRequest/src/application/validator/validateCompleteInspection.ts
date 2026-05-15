// frontend/console/mintRequest/src/application/validator/validateCompleteInspection.ts

import type { InspectionBatchDTO } from "../../infrastructure/api/mintRequestApi";

export type ValidateCompleteInspectionInput = {
  inspectionBatch: InspectionBatchDTO | null | undefined;
  requestId?: string | null;
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

  const productionId = String(
    (inspectionBatch as any).productionId ?? input.requestId ?? "",
  ).trim();

  if (!productionId) {
    return {
      ok: false,
      message: "productionId が特定できません。",
    };
  }

  return {
    ok: true,
    productionId,
  };
}