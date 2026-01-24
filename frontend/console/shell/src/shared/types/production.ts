// frontend/shell/src/shared/types/production.ts

/**
 * This file is aligned to:
 * backend/internal/domain/production/entity.go
 *
 * - ProductionStatus is a strict union of allowed statuses.
 * - Production mirrors the backend Production struct fields (and optionality).
 * - Time fields are represented as ISO8601 strings on the frontend.
 */

/**
 * ProductionStatus
 * backend/internal/domain/production/entity.go の ProductionStatus に準拠
 */
export type ProductionStatus = "printed" | "planning" | "deleted";

/**
 * ProductionModel
 * backend: type ModelQuantity struct { ModelID string; Quantity int }
 * frontend: JSON 受け取りの都合で camelCase に寄せる
 */
export type ProductionModel = {
  /** model_variations の ID（backend: ModelID） */
  modelId: string;

  /** 生産数量（backend: Quantity） */
  quantity: number;
};

/**
 * Production
 * backend/internal/domain/production/entity.go の Production 構造体に準拠
 *
 * - Backend には companyId / brandId は存在しないため削除
 * - printedAt / createdAt / updatedAt / deletedAt は ISO8601 string で表現
 * - createdBy / updatedBy / deletedBy / printedBy は optional + nullable を許容
 */
export type Production = {
  /** productions の ID（backend: ID） */
  id: string;

  /** 紐づく product_blueprints の ID（backend: ProductBlueprintID） */
  productBlueprintId: string;

  /** 担当者の memberId（backend: AssigneeID） */
  assigneeId: string;

  /** 生産ステータス（backend: Status） */
  status: ProductionStatus;

  /** モデル別の生産数量一覧（backend: Models） */
  models: ProductionModel[];

  // ─── 印刷関連 ────────────────────────────────

  /** 印刷完了日時（ISO8601）。未印刷なら null / undefined（backend: *time.Time） */
  printedAt?: string | null;

  /** 印刷担当者の memberId（backend: *string） */
  printedBy?: string | null;

  // ─── 監査情報 ────────────────────────────────

  /** 作成者の memberId（backend: *string） */
  createdBy?: string | null;

  /** 作成日時（ISO8601）。ゼロ許容のため null / undefined を許容（backend: time.Time optional） */
  createdAt?: string | null;

  /** 最終更新者の memberId（backend: *string） */
  updatedBy?: string | null;

  /** 更新日時（ISO8601）。ゼロ許容のため null / undefined を許容（backend: time.Time optional） */
  updatedAt?: string | null;

  /** 削除日時（ISO8601）。未削除なら null / undefined（backend: *time.Time） */
  deletedAt?: string | null;

  /** 削除者の memberId（backend: *string） */
  deletedBy?: string | null;
};
