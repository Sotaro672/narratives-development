// frontend/console/production/src/application/productionCreateService.tsx
// ======================================================================
// Application Service for Production Create
// ======================================================================

import type { Brand } from "../../../brand/src/domain/entity/brand";
import type { ProductBlueprintManagementRow } from "../../../productBlueprint/src/infrastructure/query/productBlueprintQuery";
import type { Member } from "../../../member/src/domain/entity/member";
import type { ModelVariationResponse } from "../../../productBlueprint/src/application/productBlueprintDetailService";
import type { ItemType, Fit } from "../../../productBlueprint/src/domain/entity/catalog";

import { getMemberFullName } from "../../../member/src/domain/entity/member";

import { ProductionRepositoryHTTP } from "../infrastructure/http/productionRepositoryHTTP";

export {
  loadBrands,
  loadProductBlueprints,
  loadDetailAndModels,
  loadAssigneeCandidates,
} from "../infrastructure/api/productionCreateApi";

export type {
  Brand,
  ProductBlueprintManagementRow,
  Member,
  ModelVariationResponse,
};

// ======================================================================
// ProductBlueprintCard
// ======================================================================
export type ProductBlueprintForCard = {
  id: string;
  productName: string;
  brand?: string;

  itemType?: ItemType;
  fit?: Fit;
  materials?: string;
  weight?: number;
  washTags?: string[];
  productIdTag?: string;
};

// ======================================================================
// ProductionQuantityRow（UI 専用）
// ======================================================================
export type ProductionQuantityRow = {
  modelVariationId: string;

  /** 型番（例: “GM”） */
  modelNumber: string;

  size: string;

  /** 色名（例: “グリーン”） */
  color: string;

  /** RGB 値（0xRRGGBB int） */
  rgb?: number | string | null;

  quantity: number;
};

// ======================================================================
// Production モデル（バックエンド準拠）
// ======================================================================
export type ProductionStatus = "draft" | "planned" | "in_progress";

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

// ======================================================================
// ブランド（変換）
// ======================================================================
export function buildBrandOptions(brands: Brand[]): string[] {
  return brands.map((b) => b.name).filter(Boolean);
}

// ======================================================================
// 商品設計一覧（変換）
// ======================================================================
export function filterProductBlueprintsByBrand(
  rows: ProductBlueprintManagementRow[],
  brandName: string | null,
) {
  if (!brandName) return [];
  return rows.filter((pb) => pb.brandName === brandName);
}

export function buildProductRows(filtered: ProductBlueprintManagementRow[]) {
  return filtered.map((pb) => ({
    id: pb.id,
    name: pb.productName,
  }));
}

// ======================================================================
// buildSelectedForCard
// ======================================================================
export function buildSelectedForCard(
  detail: any,
  row: ProductBlueprintManagementRow | null,
): ProductBlueprintForCard {
  if (detail) {
    return {
      id: detail.id,
      productName: detail.productName,
      brand: detail.brandName ?? "",
      itemType: detail.itemType as ItemType | undefined,
      fit: detail.fit as Fit | undefined,
      materials: detail.material,
      weight: detail.weight,
      washTags: detail.qualityAssurance ?? [],
      productIdTag: detail.productIdTag?.type ?? "",
    };
  }

  if (row) {
    return {
      id: row.id,
      productName: row.productName,
      brand: row.brandName,
    };
  }

  return { id: "", productName: "", brand: "" };
}

// ======================================================================
// 担当者一覧（変換）
// ======================================================================
export function buildAssigneeOptions(members: Member[]) {
  return members.map((m) => ({
    id: m.id,
    name: getMemberFullName(m) || m.email || m.id,
  }));
}

// ======================================================================
// ModelVariations → ProductionQuantityRow
// ======================================================================
export function mapModelVariationsToRows(
  list: ModelVariationResponse[],
): ProductionQuantityRow[] {
  return list.map((mv) => ({
    modelVariationId: mv.id,
    modelNumber: mv.modelNumber, // ← 修正ポイント
    size: mv.size,

    color: mv.color?.name ?? "",
    rgb: mv.color?.rgb ?? null,

    quantity: 0,
  }));
}

// ======================================================================
// Production 作成リクエスト生成
// ======================================================================
export function buildProductionRequest(params: {
  productBlueprintId: string;
  assigneeId: string;
  creatorId: string;
  quantities: ProductionQuantityRow[];
}): Production {
  const { productBlueprintId, assigneeId, creatorId, quantities } = params;

  return {
    id: "",
    productBlueprintId,
    assigneeId,
    models: quantities.map((q) => ({
      modelId: q.modelVariationId,
      quantity: q.quantity,
    })),
    status: "planned",
    createdBy: creatorId,
    createdAt: new Date().toISOString(),
  };
}

// ======================================================================
// buildProductionPayload
// ======================================================================
export function buildProductionPayload(params: {
  productBlueprintId: string;
  assigneeId: string;
  rows: ProductionQuantityRow[];
  currentMemberId: string | null;
}): Production {
  const { productBlueprintId, assigneeId, rows, currentMemberId } = params;

  const request = buildProductionRequest({
    productBlueprintId,
    assigneeId,
    creatorId: currentMemberId ?? "",
    quantities: rows,
  });

  return request;
}

// ======================================================================
// createProduction
// ======================================================================
export async function createProduction(
  payload: Production,
): Promise<Production> {
  const repo = new ProductionRepositoryHTTP();
  return await repo.create(payload);
}
