// frontend/console/shell/src/features/mintRequest/application/validator/validateMintRequestSubmit.ts

import type { InspectionBatch } from "../../../../shared/types/inspections";

export type ValidateMintRequestSubmitInput = {
  inspectionBatch: InspectionBatch | null | undefined;
  isInspectionCompleted: boolean;
  selectedTokenBlueprintId: string | null | undefined;
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

/**
 * ミント申請を送信できる状態か検証する。
 *
 * Backendはミントを同期実行せず、202 Accepted / QUEUEDを返して順次処理する。
 *
 * このvalidatorでは次の入力条件だけを検証する。
 * - 検品バッチが取得済みである
 * - 検品が完了している
 * - トークン設計が選択されている
 * - inspectionBatch.productionIdが存在する
 *
 * productionIdはBackend BFFが返すinspectionBatch.productionIdのみを正とし、
 * route parameterへのfallbackやFrontend側での再構築は行わない。
 */
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

  if (!inspectionBatch.productionId) {
    return {
      ok: false,
      message: "productionId が特定できません。",
    };
  }

  return {
    ok: true,
    productionId: inspectionBatch.productionId,
    tokenBlueprintId,
  };
}