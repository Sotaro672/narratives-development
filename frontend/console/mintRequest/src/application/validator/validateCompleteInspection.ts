// frontend/console/mintRequest/src/application/validator/validateCompleteInspection.ts

import type { InspectionBatchDTO } from "../../domain/inspections";

export type ValidateCompleteInspectionInput = {
  inspectionBatch: InspectionBatchDTO | null | undefined;

  /**
   * URL param 由来の productionId。
   * route 名が requestId のままでも、application では productionId として扱う。
   */
  productionId?: string | null;
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
    (inspectionBatch as any).productionId ?? input.productionId ?? "",
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