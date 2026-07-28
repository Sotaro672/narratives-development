// frontend/console/shell/src/shared/types/production.ts

/**
 * backend/internal/domain/production/entity.go の
 * ProductionおよびModelQuantityに対応する共有型。
 *
 * - フロントエンドではプロパティ名をcamelCaseで表現する
 * - 日時はISO 8601形式の文字列として扱う
 * - ポインタ型はnullまたはundefinedを許容する
 */

/**
 * モデル別の生産数量。
 *
 * backend:
 * type ModelQuantity struct {
 *   ModelID  string
 *   Quantity int
 * }
 */
export type ModelQuantity = {
  /** model_variationsのID（backend: ModelID） */
  modelId: string;

  /** 生産数量（backend: Quantity） */
  quantity: number;
};

/**
 * 旧名称との互換性を維持するための型エイリアス。
 *
 * 新規コードではModelQuantityを使用する。
 */
export type ProductionModel = ModelQuantity;

/**
 * Production
 *
 * backend/internal/domain/production/entity.go の
 * Production構造体に対応する。
 */
export type Production = {
  /** productionsのID（backend: ID） */
  id: string;

  /** 紐づくproduct_blueprintsのID（backend: ProductBlueprintID） */
  productBlueprintId: string;

  /** 担当者のmemberId（backend: AssigneeID） */
  assigneeId: string;

  /** モデル別の生産数量一覧（backend: Models） */
  models: ModelQuantity[];

  // ─── 印刷関連 ────────────────────────────────

  /** 印刷済みかどうか（backend: Printed） */
  printed: boolean;

  /** 印刷完了日時（backend: PrintedAt） */
  printedAt?: string | null;

  /** 印刷担当者のmemberId（backend: PrintedBy） */
  printedBy?: string | null;

  // ─── 監査情報 ────────────────────────────────

  /** 作成者のFirebase Auth UIDまたはmemberId（backend: CreatedBy） */
  createdBy?: string | null;

  /** 作成日時（backend: CreatedAt） */
  createdAt?: string | null;

  /** 最終更新者のFirebase Auth UIDまたはmemberId（backend: UpdatedBy） */
  updatedBy?: string | null;

  /** 更新日時（backend: UpdatedAt） */
  updatedAt?: string | null;
};