//frontend\console\production\src\application\create\ProductionCreateTypes.ts
// ======================================================================
// Production モデル（バックエンド準拠 / Usecase Input-Output）
// ======================================================================

export type ProductionStatus =  "planned" | "printed";

export interface ModelQuantity {
  modelId: string;
  quantity: number;
}

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

  updatedAt?: string | null;
  updatedBy?: string | null;

  deletedAt?: string | null;
  deletedBy?: string | null;
}
