//frontend\console\production\src\application\create\ProductionCreateTypes.ts
// ======================================================================
// Production モデル（バックエンド準拠 / Usecase Input-Output）
// ======================================================================

export interface ModelQuantity {
  ModelID: string;
  Quantity: number;
}

export interface Production {
  id: string;
  productBlueprintId: string;
  assigneeId: string;

  models: ModelQuantity[];

  printed?: boolean | null;
  printedAt?: string | null;
  printedBy?: string | null;

  createdBy?: string | null;
  createdAt?: string | null;

  updatedAt?: string | null;
  updatedBy?: string | null;

  deletedAt?: string | null;
  deletedBy?: string | null;
}