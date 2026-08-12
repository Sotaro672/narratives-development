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
 * - inspectionResult / inspectedBy / inspectedAt は null もしくは未設定を許容
 * - inspectedAt は ISO8601 日時文字列を想定
 */
export interface InspectionItem {
  productId: string;
  modelId: string;

  modelNumber?: string | null;

  inspectionResult?: InspectionResult | null;
  inspectedBy?: string | null;
  inspectedAt?: string | null;
}

/**
 * InspectionBatch
 * backend/internal/domain/inspection/entity.go の InspectionBatch に対応。
 *
 * - requested は boolean
 * - requestedBy / requestedAt / mintedAt / scheduledBurnDate / tokenBlueprintId は
 *   mints テーブル側が責務を持つ
 */
export interface InspectionBatch {
  productionId: string;
  status: InspectionStatus;

  quantity: number;
  totalPassed: number;

  /** ミント申請済みフラグ */
  requested: boolean;

  inspections: InspectionItem[];
}

/**
 * modelId → モデル表示情報。
 *
 * GET /mint/inspections/{productionId} の modelMeta に対応する。
 *
 * apparel:
 * - modelNumber
 * - size
 * - colorName
 * - rgb
 *
 * alcohol:
 * - modelNumber
 * - volume
 * - volumeUnit
 */
export interface MintModelMeta {
  modelNumber?: string;
  size?: string;
  colorName?: string;
  rgb?: number;

  volume?: number;
  volumeUnit?: "ml" | "L";
}

/**
 * MintRequest detail の inspection 表示用 DTO。
 *
 * InspectionBatch に加えて:
 * - productBlueprintId
 * - productName
 * - modelMeta
 */
export interface InspectionBatchDTO extends InspectionBatch {
  productBlueprintId: string;
  productName: string;
  modelMeta: Record<string, MintModelMeta>;
}

/* =========================================================
 * ユーティリティ
 * =======================================================*/

/** InspectionStatus 妥当性チェック */
export function isValidInspectionStatus(
  s: string,
): s is InspectionStatus {
  return s === "inspecting" || s === "completed";
}

/** InspectionResult 妥当性チェック */
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

/** ISO8601/日付文字列の簡易チェック */
function isValidDateTimeString(
  value: string | null | undefined,
): boolean {
  if (value == null) return false;

  const v = value.trim();
  if (!v) return false;

  const t = Date.parse(v);
  return !Number.isNaN(t);
}

/**
 * InspectionBatch の簡易バリデーション。
 * backend/internal/domain/inspection/entity.go の validate() ロジックと概ね対応。
 *
 * 問題があればエラーメッセージ配列を返す。
 */
export function validateInspectionBatch(
  batch: InspectionBatch,
): string[] {
  const errors: string[] = [];

  if (!batch.productionId?.trim()) {
    errors.push("productionId is required");
  }

  if (!isValidInspectionStatus(batch.status)) {
    errors.push("status must be 'inspecting' or 'completed'");
  }

  if (!batch.inspections || batch.inspections.length === 0) {
    errors.push("inspections must not be empty");
  }

  if (
    batch.quantity !== batch.inspections.length ||
    batch.quantity <= 0
  ) {
    errors.push(
      "quantity must equal inspections.length and be > 0",
    );
  }

  if (batch.totalPassed < 0) {
    errors.push("totalPassed must be >= 0");
  }

  if (typeof batch.requested !== "boolean") {
    errors.push("requested must be boolean");
  }

  for (const ins of batch.inspections ?? []) {
    if (!ins.productId?.trim()) {
      errors.push("inspection.productId is required");
      continue;
    }

    if (ins.inspectionResult == null) {
      continue;
    }

    if (!isValidInspectionResult(ins.inspectionResult)) {
      errors.push(
        `inspectionResult must be one of 'notYet' | 'passed' | 'failed' | 'notManufactured' (productId=${ins.productId})`,
      );
      continue;
    }

    if (ins.inspectionResult === "notYet") {
      continue;
    }

    const hasBy =
      !!ins.inspectedBy &&
      ins.inspectedBy.trim() !== "";

    const hasAt =
      !!ins.inspectedAt &&
      ins.inspectedAt.trim() !== "";

    if (!hasBy) {
      errors.push(
        `inspectedBy is required when inspectionResult is '${ins.inspectionResult}' (productId=${ins.productId})`,
      );
    }

    if (!hasAt) {
      errors.push(
        `inspectedAt is required when inspectionResult is '${ins.inspectionResult}' (productId=${ins.productId})`,
      );
    } else if (
      !isValidDateTimeString(ins.inspectedAt!)
    ) {
      errors.push(
        `inspectedAt must be a valid datetime string (productId=${ins.productId})`,
      );
    }
  }

  return errors;
}