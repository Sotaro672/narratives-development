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

export type ValidateMintExecutionResultInput = {
  refreshedMint: unknown;
};

export type ValidateMintExecutionResultResult =
  | {
      ok: true;
    }
  | {
      ok: false;
      message: string;
    };

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function getString(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function getBoolean(value: unknown): boolean {
  return value === true;
}

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

export function validateMintExecutionResult(
  input: ValidateMintExecutionResultInput,
): ValidateMintExecutionResultResult {
  const mint = input.refreshedMint;

  if (!isRecord(mint)) {
    return {
      ok: false,
      message:
        "ミント申請は作成されましたが、ミント結果を取得できませんでした。画面を更新してミント状態を確認してください。",
    };
  }

  const minted = getBoolean(mint.minted);
  const txSignature = getString(mint.onChainTxSignature);

  if (!minted) {
    return {
      ok: false,
      message:
        "ミント申請は作成されましたが、オンチェーンのミント完了を確認できませんでした。minted が false のため、バックエンドログを確認してください。",
    };
  }

  if (!txSignature) {
    return {
      ok: false,
      message:
        "ミント申請は作成されましたが、トランザクション署名を確認できませんでした。onChainTxSignature が空のため、ミント結果を確認してください。",
    };
  }

  return {
    ok: true,
  };
}