// frontend/console/mintRequest/src/domain/entity/inspections.ts

/**
 * InspectionResult
 * backend/internal/domain/inspection/entity.go の InspectionResult に対応。
 *
 * - "notYet"          : 未検査
 * - "passed"          : 合格
 * - "failed"          : 不合格
 * - "notManufactured" : 生産されていない（欠品など）
 */
export type InspectionResult =
  | "notYet"
  | "passed"
  | "failed"
  | "notManufactured";

/**
 * InspectionStatus
 * backend/internal/domain/inspection/entity.go の InspectionStatus に対応。
 *
 * - "inspecting" : 検査中
 * - "completed"  : 検査完了
 */
export type InspectionStatus = "inspecting" | "completed";

/**
 * InspectionItem
 * backend/internal/domain/inspection/entity.go の InspectionItem に対応。
 *
 * - modelNumber / inspectionResult / inspectedBy / inspectedAt は null もしくは未設定を許容
 * - inspectedAt は ISO8601 日時文字列（例: "2025-01-01T00:00:00Z"）を想定
 */
export interface InspectionItem {
  productId: string;
  modelId: string;
  modelNumber?: string | null;
  inspectionResult?: InspectionResult | null;
  inspectedBy?: string | null;
  inspectedAt?: string | null; // time.Time 相当（ISO8601）
}

/**
 * InspectionBatch
 * backend/internal/domain/inspection/entity.go の InspectionBatch に対応。
 *
 * - requestedBy / requestedAt / mintedAt / scheduledBurnDate / tokenBlueprintId は null or 未設定
 * - 日付/日時系は ISO8601 文字列を想定
 * - productName は MintRequest 画面向けの追加情報（任意）
 */
export interface InspectionBatch {
  productionId: string;
  status: InspectionStatus;

  quantity: number;
  totalPassed: number;

  // MintRequest 向けに、productionId から解決された productName を載せるための任意フィールド
  productName?: string | null;

  requestedBy?: string | null;
  requestedAt?: string | null;       // ISO8601 datetime
  mintedAt?: string | null;          // ISO8601 datetime
  scheduledBurnDate?: string | null; // ISO8601 datetime
  tokenBlueprintId?: string | null;

  inspections: InspectionItem[];
}

/* =========================================================
 * ユーティリティ
 * =======================================================*/

/** InspectionStatus 妥当性チェック（Go の IsValidInspectionStatus に対応） */
export function isValidInspectionStatus(
  s: string,
): s is InspectionStatus {
  return s === "inspecting" || s === "completed";
}

/** InspectionResult 妥当性チェック（Go の IsValidInspectionResult に対応） */
export function isValidInspectionResult(
  r: string,
): r is InspectionResult {
  return (
    r === "notYet" ||
    r === "passed" ||
    r === "failed" ||
    r === "notManufactured"
  );
}

/** ISO8601/日付文字列の簡易チェック（空文字は非許容） */
function isValidDateTimeString(value: string | null | undefined): boolean {
  if (value == null) return false;
  const v = value.trim();
  if (!v) return false;
  const t = Date.parse(v);
  return !Number.isNaN(t);
}

/**
 * InspectionBatch の簡易バリデーション
 * backend/internal/domain/inspection/entity.go の validate() ロジックと概ね対応。
 *
 * 問題があればエラーメッセージ配列を返す。
 */
export function validateInspectionBatch(
  batch: InspectionBatch,
): string[] {
  const errors: string[] = [];

  // productionId
  if (!batch.productionId?.trim()) {
    errors.push("productionId is required");
  }

  // status
  if (!isValidInspectionStatus(batch.status)) {
    errors.push("status must be 'inspecting' or 'completed'");
  }

  // inspections
  if (!batch.inspections || batch.inspections.length === 0) {
    errors.push("inspections must not be empty");
  }

  // quantity / totalPassed
  if (batch.quantity !== batch.inspections.length || batch.quantity <= 0) {
    errors.push("quantity must equal inspections.length and be > 0");
  }
  if (batch.totalPassed < 0) {
    errors.push("totalPassed must be >= 0");
  }

  // inspections[i] の整合性チェック
  for (const ins of batch.inspections) {
    if (!ins.productId?.trim()) {
      errors.push("inspection.productId is required");
      continue;
    }

    // inspectionResult が未設定なら、by/at があっても許容（Go 実装に合わせて緩くする）
    if (ins.inspectionResult == null) {
      continue;
    }

    if (!isValidInspectionResult(ins.inspectionResult)) {
      errors.push(
        `inspectionResult must be one of 'notYet' | 'passed' | 'failed' | 'notManufactured' (productId=${ins.productId})`,
      );
      continue;
    }

    // notYet の場合は by/at が入っていてもエラーにしない
    if (ins.inspectionResult === "notYet") {
      continue;
    }

    // passed / failed / notManufactured のときは by / at 必須
    const hasBy = !!ins.inspectedBy && ins.inspectedBy.trim() !== "";
    const hasAt = !!ins.inspectedAt && ins.inspectedAt.trim() !== "";

    if (!hasBy) {
      errors.push(
        `inspectedBy is required when inspectionResult is '${ins.inspectionResult}' (productId=${ins.productId})`,
      );
    }
    if (!hasAt) {
      errors.push(
        `inspectedAt is required when inspectionResult is '${ins.inspectionResult}' (productId=${ins.productId})`,
      );
    } else if (!isValidDateTimeString(ins.inspectedAt!)) {
      errors.push(
        `inspectedAt must be a valid datetime string (productId=${ins.productId})`,
      );
    }
  }

  return errors;
}

/**
 * InspectionBatch の正規化用ヘルパ
 * - 文字列を trim
 * - 空文字の任意フィールドは null に丸める
 * - バリデーションエラー時は例外を投げる
 */
export function normalizeInspectionBatch(
  input: InspectionBatch,
): InspectionBatch {
  const normalizeOpt = (
    v: string | null | undefined,
  ): string | null => {
    const t = v?.trim() ?? "";
    return t ? t : null;
  };

  const normalized: InspectionBatch = {
    ...input,
    productionId: input.productionId.trim(),
    status: input.status,
    quantity: input.quantity,
    totalPassed: input.totalPassed,
    // 任意の productName は trim + 空文字 → null に正規化
    productName: normalizeOpt(input.productName),
    requestedBy: normalizeOpt(input.requestedBy),
    requestedAt: normalizeOpt(input.requestedAt),
    mintedAt: normalizeOpt(input.mintedAt),
    scheduledBurnDate: normalizeOpt(input.scheduledBurnDate),
    tokenBlueprintId: normalizeOpt(input.tokenBlueprintId),
    inspections: (input.inspections ?? []).map((ins) => ({
      ...ins,
      productId: ins.productId.trim(),
      modelId: ins.modelId.trim(),
      modelNumber: normalizeOpt(ins.modelNumber),
      inspectionResult: ins.inspectionResult ?? null,
      inspectedBy: normalizeOpt(ins.inspectedBy),
      inspectedAt: normalizeOpt(ins.inspectedAt),
    })),
  };

  const errors = validateInspectionBatch(normalized);
  if (errors.length > 0) {
    throw new Error(
      `Invalid InspectionBatch: ${errors.join(", ")}`,
    );
  }

  return normalized;
}
