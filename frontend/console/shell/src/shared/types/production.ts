// frontend/shell/src/shared/types/production.ts

/**
 * ProductionStatus
 * バックエンド（productiondom.Production.Status）のラッパー。
 * 実際の値は "planning" / "inProgress" / "printing" / "printed" などが入りますが、
 * 型崩れを避けるため現時点では string として扱います。
 */
export type ProductionStatus = string;

/**
 * ProductionModel
 * バックエンドの Production.Models 要素に対応する想定の型です。
 * - modelId: 対象の ModelVariation の ID
 * - quantity: そのモデルで生産する数量
 */
export type ProductionModel = {
  /** Firestore ドキュメント ID 等がある場合に使う（なければ無視されてもよい） */
  id?: string;
  /** model_variations コレクションの ID */
  modelId: string;
  /** 生産数量 */
  quantity: number;
};

/**
 * Production
 * backend/internal/domain/production/entity.go の Production 構造体に対応する
 * フロントエンド用の共通型です。
 *
 * Firestore からの JSON をそのまま受け取れるよう、日時は string（ISO8601）で扱います。
 * 画面ごとの DTO（一覧・詳細・作成フォーム用）は各コンテキストの
 * application 層でこの型を拡張して利用してください。
 */
export type Production = {
  /** productions コレクションのドキュメント ID */
  id: string;

  /** 会社 ID */
  companyId: string;

  /** ブランド ID */
  brandId: string;

  /** 紐づく product_blueprints の ID */
  productBlueprintId: string;

  /** 担当者の memberId（未設定なら null / undefined） */
  assigneeId?: string | null;

  /** 生産ステータス */
  status: ProductionStatus;

  /** モデル別の生産数量一覧 */
  models: ProductionModel[];

  // ─── 印刷関連 ────────────────────────────────

  /** 印刷完了日時（ISO8601）。未印刷なら null / undefined */
  printedAt?: string | null;

  /** 印刷を実行したメンバーの memberId（ない場合は null / undefined） */
  printedBy?: string | null;

  // ─── 監査情報 ────────────────────────────────

  /** 作成者の memberId（履歴用途。無い場合もある） */
  createdBy?: string | null;

  /** 最終更新者の memberId（履歴用途。無い場合もある） */
  updatedBy?: string | null;

  /** 作成日時（ISO8601） */
  createdAt: string;

  /** 更新日時（ISO8601） */
  updatedAt: string;
};
