// frontend/production/mockdata.tsx

import {
  type Production,
  type ModelQuantity,
  createProduction,
} from "../../../../shell/src/shared/types/production";

/**
 * モックデータ用に簡略化された生産レコード。
 * backend の Production エンティティ構造に基づく。
 */
export const PRODUCTIONS: Production[] = [
  createProduction({
    id: "production_001",
    productBlueprintId: "pb_001",
    assigneeId: "member_001",
    models: [{ modelId: "model_001", quantity: 10 } as ModelQuantity],
    status: "printed",
    printedAt: "2025-11-03T00:00:00Z",
    inspectedAt: null,
    createdBy: "member_001",
    createdAt: "2025-11-05T00:00:00Z",
    updatedBy: null,
    updatedAt: null,
    deletedBy: null,
    deletedAt: null,
  }),

  createProduction({
    id: "production_002",
    productBlueprintId: "pb_002",
    assigneeId: "member_002",
    models: [{ modelId: "model_002", quantity: 9 } as ModelQuantity],
    status: "printed",
    printedAt: "2025-11-04T00:00:00Z",
    inspectedAt: null,
    createdBy: "member_002",
    createdAt: "2025-11-05T00:00:00Z",
    updatedBy: null,
    updatedAt: null,
    deletedBy: null,
    deletedAt: null,
  }),

  createProduction({
    id: "production_003",
    productBlueprintId: "pb_001",
    assigneeId: "member_001",
    models: [{ modelId: "model_003", quantity: 7 } as ModelQuantity],
    status: "printed",
    printedAt: "2025-11-01T00:00:00Z",
    inspectedAt: null,
    createdBy: "member_001",
    createdAt: "2025-10-31T00:00:00Z",
    updatedBy: null,
    updatedAt: null,
    deletedBy: null,
    deletedAt: null,
  }),

  createProduction({
    id: "production_004",
    productBlueprintId: "pb_002",
    assigneeId: "member_002",
    models: [{ modelId: "model_004", quantity: 4 } as ModelQuantity],
    status: "printed",
    printedAt: "2025-10-30T00:00:00Z",
    inspectedAt: null,
    createdBy: "member_002",
    createdAt: "2025-10-29T00:00:00Z",
    updatedBy: null,
    updatedAt: null,
    deletedBy: null,
    deletedAt: null,
  }),

  createProduction({
    id: "production_005",
    productBlueprintId: "pb_001",
    assigneeId: "member_001",
    models: [{ modelId: "model_005", quantity: 2 } as ModelQuantity],
    status: "manufacturing",
    printedAt: null,
    inspectedAt: null,
    createdBy: "member_001",
    createdAt: "2025-11-04T00:00:00Z",
    updatedBy: null,
    updatedAt: null,
    deletedBy: null,
    deletedAt: null,
  }),
];
