// frontend/production/src/domain/entity/production.ts

/**
 * ModelQuantity
 * backend/internal/domain/production/entity.go の ModelQuantity に対応
 */
export interface ModelQuantity {
  /** モデルID */
  modelId: string;
  /** 生産数量 (1以上) */
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
  | "planned"
  | "deleted"
  | "suspended";

/** Status の妥当性チェック */
export function isValidProductionStatus(
  status: string
): status is ProductionStatus {
  return (
    status === "manufacturing" ||
    status === "inspected" ||
    status === "printed" ||
    status === "planned" ||
    status === "deleted" ||
    status === "suspended"
  );
}

/**
 * Production
 * backend/internal/domain/production/entity.go の Production 構造体に対応するフロントエンド用モデル
 *
 * - 日付は ISO8601 文字列
 * - ポインタ型は `string | null` として表現
 * - CreatedAt / UpdatedAt は backend 同様「ゼロ許容」だが、
 *   フロントでは空文字/undefined/null を「未設定」とみなす想定
 */
export interface Production {
  id: string;
  productBlueprintId: string;
  assigneeId: string;

  /** [{ modelId, quantity }] */
  models: ModelQuantity[];

  status: ProductionStatus;

  printedAt?: string | null;
  inspectedAt?: string | null;

  createdBy?: string | null;
  createdAt?: string | null; // optional / zero-allowed

  updatedAt?: string | null; // optional / zero-allowed
  updatedBy?: string | null;

  deletedAt?: string | null;
  deletedBy?: string | null;
}

/* =========================================================
 * ユーティリティ
 * =======================================================*/

/** models 配列の正規化: 空ID・数量<=0を除去し、modelId(小文字)で重複排除 */
export function normalizeModelQuantities(
  models: ModelQuantity[]
): ModelQuantity[] {
  const seen = new Set<string>();
  const out: ModelQuantity[] = [];

  for (const m of models || []) {
    const id = (m.modelId ?? "").trim();
    if (!id || m.quantity <= 0) continue;

    const key = id.toLowerCase();
    if (seen.has(key)) continue;

    seen.add(key);
    out.push({ modelId: id, quantity: m.quantity });
  }

  return out;
}

/** Production の簡易バリデーション（backend の validate() に整合） */
export function validateProduction(p: Production): string[] {
  const errors: string[] = [];

  if (!p.id?.trim()) errors.push("id is required");
  if (!p.productBlueprintId?.trim())
    errors.push("productBlueprintId is required");
  if (!p.assigneeId?.trim()) errors.push("assigneeId is required");

  if (!Array.isArray(p.models) || p.models.length === 0) {
    errors.push("models must contain at least one item");
  } else {
    for (const mq of p.models) {
      if (!mq.modelId?.trim()) {
        errors.push("modelId is required");
      }
      if (mq.quantity == null || mq.quantity <= 0) {
        errors.push("quantity must be > 0");
      }
    }
  }

  if (!isValidProductionStatus(p.status)) {
    errors.push("invalid status");
  }

  // printed / inspected / deleted の整合性チェック（backend ロジック準拠で簡略）
  if (p.status === "printed" && !p.printedAt) {
    errors.push("printedAt is required when status is printed");
  }
  if (p.status === "inspected") {
    if (!p.printedAt) errors.push("printedAt is required when inspected");
    if (!p.inspectedAt) errors.push("inspectedAt is required when inspected");
  }
  if (p.status === "deleted" && !p.deletedAt) {
    errors.push("deletedAt is required when status is deleted");
  }

  return errors;
}
