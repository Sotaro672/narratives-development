// frontend\console\production\src\presentation\create\mappers.ts

import type { Brand } from "../../../../brand/src/domain/entity/brand";
import type { Member } from "../../../../member/src/domain/entity/member";
import { getMemberFullName } from "../../../../member/src/domain/entity/member";

import type { ProductBlueprintManagementRow } from "../../../../productBlueprint/src/infrastructure/query/productBlueprintQuery";
import type { ModelVariationResponse } from "../../../../productBlueprint/src/application/productBlueprintDetailService";
import type { ItemType, Fit } from "../../../../productBlueprint/src/domain/entity/catalog";

import type { ProductBlueprintForCard, ProductionQuantityRow } from "./types";

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
): ProductBlueprintManagementRow[] {
  if (!brandName) return [];
  return rows.filter((pb) => pb.brandName === brandName);
}

export function buildProductRows(
  filtered: ProductBlueprintManagementRow[],
): Array<{ id: string; name: string }> {
  return filtered.map((pb) => ({
    id: pb.id,
    name: pb.productName,
  }));
}

// ======================================================================
// buildSelectedForCard（UIカード表示用）
// ======================================================================
// detail は productBlueprintDetailService 等から返る DTO を想定（現状 any を許容）
// 型を強めたい場合は detail DTO 型を定義して差し替えてください。
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
export function buildAssigneeOptions(
  members: Member[],
): Array<{ id: string; name: string }> {
  return members.map((m) => ({
    id: m.id,
    name: getMemberFullName(m) || m.email || m.id,
  }));
}

// ======================================================================
// ModelVariations → ProductionQuantityRow（UI入力用の行に変換）
// ======================================================================
export function mapModelVariationsToRows(
  list: ModelVariationResponse[],
): ProductionQuantityRow[] {
  return list.map((mv) => ({
    modelVariationId: mv.id,
    modelNumber: mv.modelNumber,
    size: mv.size,

    color: mv.color?.name ?? "",
    rgb: mv.color?.rgb ?? null,

    quantity: 0,
  }));
}
