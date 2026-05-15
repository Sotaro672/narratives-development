// frontend/console/mintRequest/src/application/validator/validateMintRequestSubmit.ts

import type { InspectionBatchDTO } from "../../domain/entity/inspections";

export type ValidateMintRequestSubmitInput = {
  inspectionBatch: InspectionBatchDTO | null | undefined;
  isInspectionCompleted: boolean;
  selectedTokenBlueprintId: string | null | undefined;

  /**
   * URL param 由来の productionId。
   * route 名が requestId のままでも、application では productionId として扱う。
   */
  productionId?: string | null;
};

export type ValidateMintRequestSubmitResult =
  | {
      ok: true;
      productionId: string;
      tokenBlueprintId: string;
    }
  | {
      ok: false;
      message: string;
    };

export function validateMintRequestSubmit(
  input: ValidateMintRequestSubmitInput,
): ValidateMintRequestSubmitResult {
  const inspectionBatch = input.inspectionBatch ?? null;

  if (!inspectionBatch) {
    return {
      ok: false,
      message: "検査バッチ情報が取得できていません。",
    };
  }

  if (!input.isInspectionCompleted) {
    return {
      ok: false,
      message: "先に検品を完了してください。",
    };
  }

  const tokenBlueprintId = String(input.selectedTokenBlueprintId ?? "").trim();

  if (!tokenBlueprintId) {
    return {
      ok: false,
      message: "トークン設計を選択してください。",
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
    tokenBlueprintId,
  };
}