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
 * - inspectionResult / inspectedBy / inspectedAt は null もしくは未設定を許容（Go のポインタに対応）
 * - inspectedAt は ISO8601 日時文字列（例: "2025-01-01T00:00:00Z"）を想定
 */
export interface InspectionItem {
  productId: string;
  modelId: string;

  // ★追加: useInspectionResultCard.tsx が参照するため
  // 既存データに無い可能性があるので optional + nullable で受ける
  modelNumber?: string | null;

  inspectionResult?: InspectionResult | null;
  inspectedBy?: string | null;
  inspectedAt?: string | null; // time.Time 相当（ISO8601）
}

/**
 * InspectionBatch
 * backend/internal/domain/inspection/entity.go の InspectionBatch に対応。
 *
 * - requested は boolean（inspections 側は requested だけを持つ）
 * - requestedBy / requestedAt / mintedAt / scheduledBurnDate / tokenBlueprintId は
 *   mints テーブル側が責務を持つため、この型からは削除する
 */
export interface InspectionBatch {
  productionId: string;
  status: InspectionStatus;

  quantity: number;
  totalPassed: number;

  /** ミント申請済みフラグ（mints 側に詳細がある前提） */
  requested: boolean;

  inspections: InspectionItem[];
}

/**
 * MintUsecase が返す modelId → モデルメタ情報のマップ要素。
 * backend/internal/application/usecase/mint_usecase.go の MintModelMeta に対応。
 */
export interface MintModelMeta {
  size: string;
  colorName: string;
  rgb: number;
}

/**
 * MintUsecase が返す MintInspectionView に対応するフロント側 DTO。
 * InspectionBatch に加えて:
 *
 * - productBlueprintId
 * - productName
 * - modelMeta: modelId → { size, colorName, rgb }
 */
export interface MintInspectionView extends InspectionBatch {
  productBlueprintId: string;
  productName: string;
  modelMeta: Record<string, MintModelMeta>;
}

/* =========================================================
 * ユーティリティ
 * =======================================================*/

/** InspectionStatus 妥当性チェック（Go の IsValidInspectionStatus に対応） */
export function isValidInspectionStatus(s: string): s is InspectionStatus {
  return s === "inspecting" || s === "completed";
}

/** InspectionResult 妥当性チェック（Go の IsValidInspectionResult に対応） */
export function isValidInspectionResult(r: string): r is InspectionResult {
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
export function validateInspectionBatch(batch: InspectionBatch): string[] {
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

  // requested
  if (typeof batch.requested !== "boolean") {
    errors.push("requested must be boolean");
  }

  // inspections[i] の整合性チェック（Go の validate() と同じ方針）
  for (const ins of batch.inspections ?? []) {
    if (!ins.productId?.trim()) {
      errors.push("inspection.productId is required");
      continue;
    }

    // InspectionResult が nil の場合は「まだ何も書いていない」扱い
    // inspectedBy/inspectedAt が入っていてもエラーにしない。
    if (ins.inspectionResult == null) {
      continue;
    }

    if (!isValidInspectionResult(ins.inspectionResult)) {
      errors.push(
        `inspectionResult must be one of 'notYet' | 'passed' | 'failed' | 'notManufactured' (productId=${ins.productId})`,
      );
      continue;
    }

    // notYet の場合は互換性のため、by/at が入っていてもエラーにしない
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
  const normalizeOpt = (v: string | null | undefined): string | null => {
    const t = v?.trim() ?? "";
    return t ? t : null;
  };

  const normalized: InspectionBatch = {
    ...input,
    productionId: input.productionId.trim(),
    status: input.status,
    quantity: input.quantity,
    totalPassed: input.totalPassed,
    requested: !!input.requested,
    inspections: (input.inspections ?? []).map((ins) => ({
      ...ins,
      productId: ins.productId.trim(),
      modelId: ins.modelId.trim(),
      // ★追加: 文字列なら trim、空なら null
      modelNumber: normalizeOpt((ins as any).modelNumber),
      inspectionResult: (ins.inspectionResult ?? null) as
        | InspectionResult
        | null,
      inspectedBy: normalizeOpt(ins.inspectedBy),
      inspectedAt: normalizeOpt(ins.inspectedAt),
    })),
  };

  const errors = validateInspectionBatch(normalized);
  if (errors.length > 0) {
    throw new Error(`Invalid InspectionBatch: ${errors.join(", ")}`);
  }

  return normalized;
}
