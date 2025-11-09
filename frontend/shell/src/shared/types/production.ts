// frontend/shell/src/shared/types/production.ts

/**
 * ModelQuantity
 * backend/internal/domain/production/entity.go の ModelQuantity に対応
 */
export interface ModelQuantity {
  /** モデルID */
  modelId: string;
  /** 数量 (1以上) */
  quantity: number;
}

/**
 * ProductionStatus
 * backend/internal/domain/production/entity.go の ProductionStatus に対応
 */
export type ProductionStatus =
  | "manufacturing"
  | "inspected"
  | "printed"
  | "planning"
  | "deleted"
  | "suspended";

/** ProductionStatus の妥当性チェック */
export function isValidProductionStatus(
  status: string
): status is ProductionStatus {
  return (
    status === "manufacturing" ||
    status === "inspected" ||
    status === "printed" ||
    status === "planning" ||
    status === "deleted" ||
    status === "suspended"
  );
}

/**
 * Production
 * backend/internal/domain/production/entity.go の Production に対応する共通型
 *
 * - 日付は ISO8601 文字列
 * - ポインタ型は string | null で表現
 */
export interface Production {
  id: string;
  productBlueprintId: string;
  assigneeId: string;
  models: ModelQuantity[];
  status: ProductionStatus;
  printedAt?: string | null;
  inspectedAt?: string | null;
  createdBy?: string | null;
  createdAt?: string | null;
  updatedBy?: string | null;
  updatedAt?: string | null;
  deletedBy?: string | null;
  deletedAt?: string | null;
}

/* =========================================================
 * ユーティリティ
 * =======================================================*/

/** models 配列の正規化（空ID・数量<=0を除外、modelId小文字で重複排除） */
export function normalizeModelQuantities(
  models: ModelQuantity[]
): ModelQuantity[] {
  const seen = new Set<string>();
  const result: ModelQuantity[] = [];

  for (const m of models || []) {
    const id = (m.modelId ?? "").trim();
    if (!id || m.quantity <= 0) continue;
    const key = id.toLowerCase();
    if (seen.has(key)) continue;
    seen.add(key);
    result.push({ modelId: id, quantity: m.quantity });
  }
  return result;
}

/** Production の簡易バリデーション（backend の validate() に整合） */
export function validateProduction(p: Production): string[] {
  const errors: string[] = [];

  if (!p.id?.trim()) errors.push("id is required");
  if (!p.productBlueprintId?.trim())
    errors.push("productBlueprintId is required");
  if (!p.assigneeId?.trim()) errors.push("assigneeId is required");

  if (!Array.isArray(p.models) || p.models.length === 0) {
    errors.push("models must contain at least one element");
  } else {
    for (const mq of p.models) {
      if (!mq.modelId?.trim()) errors.push("modelId is required");
      if (mq.quantity <= 0) errors.push("quantity must be > 0");
    }
  }

  if (!isValidProductionStatus(p.status)) {
    errors.push("invalid status");
  }

  // 状態と時刻の整合性チェック
  switch (p.status) {
    case "printed":
      if (!p.printedAt) errors.push("printedAt is required when printed");
      break;
    case "inspected":
      if (!p.printedAt) errors.push("printedAt is required when inspected");
      if (!p.inspectedAt)
        errors.push("inspectedAt is required when inspected");
      break;
    case "deleted":
      if (!p.deletedAt) errors.push("deletedAt is required when deleted");
      break;
  }

  return errors;
}

/**
 * ファクトリ関数：入力値を正規化して Production を生成
 */
export function createProduction(
  input: Omit<Production, "models"> & { models?: ModelQuantity[] }
): Production {
  const models = normalizeModelQuantities(input.models ?? []);
  return { ...input, models };
}
