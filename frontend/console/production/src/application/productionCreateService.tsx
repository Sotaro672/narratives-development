//frontend\console\production\src\application\productionCreateService.tsx
// ======================================================================
// Application Service for Production Create
// ======================================================================

// =========================
// 外部型のインポート（独自定義しない）
// =========================
import type { Brand } from "../../../brand/src/domain/entity/brand";
import type { ProductBlueprintManagementRow } from "../../../productBlueprint/src/infrastructure/query/productBlueprintQuery";
import type { Member } from "../../../member/src/domain/entity/member";
import type { ModelVariationResponse } from "../../../productBlueprint/src/application/productBlueprintDetailService";
import type {
  ItemType,
  Fit,
} from "../../../productBlueprint/src/domain/entity/catalog";

import { fetchAllBrandsForCompany } from "../../../brand/src/infrastructure/query/brandQuery";
import { fetchProductBlueprintManagementRows } from "../../../productBlueprint/src/infrastructure/query/productBlueprintQuery";
import {
  getProductBlueprintDetail,
  listModelVariationsByProductBlueprintId,
} from "../../../productBlueprint/src/application/productBlueprintDetailService";
import { scopedFilterByCompanyId } from "../../../member/src/domain/repository/memberRepository";
import { MemberRepositoryHTTP } from "../../../member/src/infrastructure/http/memberRepositoryHTTP";
import { getMemberFullName } from "../../../member/src/domain/entity/member";

// =========================
// 型の再エクスポート（エラー回避）
// =========================
export type {
  Brand,
  ProductBlueprintManagementRow,
  Member,
  ModelVariationResponse,
};

// ======================================================================
// ProductBlueprintCard 用データ型
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
  modelCode: string;
  size: string;
  colorName: string;
  colorCode?: string;
  stock: number;
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
// ブランド
// ======================================================================
export async function loadBrands(): Promise<Brand[]> {
  try {
    return await fetchAllBrandsForCompany("", true);
  } catch {
    return [];
  }
}

export function buildBrandOptions(brands: Brand[]): string[] {
  return brands.map((b) => b.name).filter(Boolean);
}

// ======================================================================
// 商品設計一覧
// ======================================================================
export async function loadProductBlueprints(): Promise<
  ProductBlueprintManagementRow[]
> {
  try {
    return await fetchProductBlueprintManagementRows();
  } catch {
    return [];
  }
}

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
// 詳細 + ModelVariations
// ======================================================================
export async function loadDetailAndModels(pbId: string): Promise<{
  detail: any;
  models: ModelVariationResponse[];
}> {
  const [detail, models] = await Promise.all([
    getProductBlueprintDetail(pbId),
    listModelVariationsByProductBlueprintId(pbId),
  ]);
  return { detail, models };
}

// ProductBlueprintCard 用の整形
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

  return {
    id: "",
    productName: "",
    brand: "",
  };
}

// ======================================================================
// 担当者一覧
// ======================================================================
export async function loadAssigneeCandidates(
  companyId: string,
): Promise<Member[]> {
  try {
    const filter = scopedFilterByCompanyId(companyId, { status: "active" });
    const repo = new MemberRepositoryHTTP();
    const page = { number: 1, perPage: 200, totalPages: 1 };
    const result = await repo.list(page, filter);
    return result.items ?? [];
  } catch {
    return [];
  }
}

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
  return list.map((mv) => {
    let colorCode = "#FFFFFF";

    if (typeof mv.color?.rgb === "number") {
      const rgb = mv.color.rgb;
      // ★ 要件: rgb:0 は黒として扱う
      if (rgb === 0) {
        colorCode = "#000000";
      } else {
        colorCode = `#${rgb.toString(16).padStart(6, "0")}`;
      }
    }

    return {
      modelVariationId: mv.id,
      modelCode: mv.modelNumber,
      size: mv.size,
      colorName: mv.color?.name ?? "",
      colorCode,
      stock: 0,
    };
  });
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
      quantity: q.stock,
    })),
    status: "planned",
    createdBy: creatorId,
    createdAt: new Date().toISOString(),
  };
}

// useProductionCreate から呼び出すためのラッパー
export function buildProductionPayload(params: {
  productBlueprintId: string;
  assigneeId: string;
  rows: ProductionQuantityRow[];
  currentMemberId: string | null;
}): Production {
  const { productBlueprintId, assigneeId, rows, currentMemberId } = params;

  return buildProductionRequest({
    productBlueprintId,
    assigneeId,
    creatorId: currentMemberId ?? "",
    quantities: rows,
  });
}
