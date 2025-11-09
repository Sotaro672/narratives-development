// frontend/model/src/infrastructure/mockdata/mockdata.ts

import type {
  ModelData,
  ModelVariation,
  ModelNumber,
  SizeVariation,
  ItemSpec,
} from "../../../../shell/src/shared/types/model";

/**
 * モデル関連のモックデータ
 * backend/internal/domain/model/entity.go に対応
 */

/* =========================================================
 * ModelVariation モック
 * =======================================================*/

export const MODEL_VARIATIONS: ModelVariation[] = [
  {
    id: "mv-001",
    productBlueprintId: "pb-001",
    modelNumber: "LM-SB-S-WHT",
    size: "S",
    color: "ホワイト",
    measurements: { 着丈: 60.5, 身幅: 47.5, 肩幅: 38.0, 袖丈: 56.0 },
    createdAt: "2024-04-10T00:00:00Z",
    createdBy: "member-001",
    updatedAt: "2024-04-12T00:00:00Z",
    updatedBy: "member-002",
  },
  {
    id: "mv-002",
    productBlueprintId: "pb-001",
    modelNumber: "LM-SB-M-WHT",
    size: "M",
    color: "ホワイト",
    measurements: { 着丈: 62.0, 身幅: 49.0, 肩幅: 39.0, 袖丈: 57.0 },
    createdAt: "2024-04-10T00:00:00Z",
    createdBy: "member-001",
    updatedAt: "2024-04-12T00:00:00Z",
    updatedBy: "member-002",
  },
  {
    id: "mv-003",
    productBlueprintId: "pb-001",
    modelNumber: "LM-SB-L-BLK",
    size: "L",
    color: "ブラック",
    measurements: { 着丈: 64.0, 身幅: 52.0, 肩幅: 40.0, 袖丈: 58.5 },
    createdAt: "2024-04-10T00:00:00Z",
    createdBy: "member-001",
    updatedAt: "2024-04-12T00:00:00Z",
    updatedBy: "member-002",
  },
];

/* =========================================================
 * ModelData モック
 * =======================================================*/

export const MODEL_DATA: ModelData = {
  productId: "prod-001",
  productBlueprintId: "pb-001",
  variations: MODEL_VARIATIONS,
  updatedAt: "2024-04-12T00:00:00Z",
};

/* =========================================================
 * SizeVariation モック
 * =======================================================*/

export const SIZE_VARIATIONS: SizeVariation[] = [
  {
    id: "size-s",
    size: "S",
    measurements: { 着丈: 60.5, 身幅: 47.5, 肩幅: 38.0, 袖丈: 56.0 },
  },
  {
    id: "size-m",
    size: "M",
    measurements: { 着丈: 62.0, 身幅: 49.0, 肩幅: 39.0, 袖丈: 57.0 },
  },
  {
    id: "size-l",
    size: "L",
    measurements: { 着丈: 64.0, 身幅: 52.0, 肩幅: 40.0, 袖丈: 58.5 },
  },
];

/* =========================================================
 * ModelNumber モック
 * =======================================================*/

export const MODEL_NUMBERS: ModelNumber[] = [
  { size: "S", color: "ホワイト", modelNumber: "LM-SB-S-WHT" },
  { size: "M", color: "ホワイト", modelNumber: "LM-SB-M-WHT" },
  { size: "L", color: "ブラック", modelNumber: "LM-SB-L-BLK" },
];

/* =========================================================
 * ItemSpec モック
 * =======================================================*/

export const ITEM_SPECS: ItemSpec[] = MODEL_VARIATIONS.map((v) => ({
  modelNumber: v.modelNumber,
  size: v.size,
  color: v.color,
  measurements: v.measurements,
}));
