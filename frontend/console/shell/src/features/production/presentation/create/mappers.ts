// frontend/console/shell/src/features/production/presentation/create/mappers.ts

import type { Brand } from "../../../../shared/types/brand";
import type { Member } from "../../../../shared/types/member";

import type { ProductBlueprintManagementRow } from "../../../productBlueprint/infrastructure/query/productBlueprintQuery";
import type { ProductBlueprintCategorySnapshot } from "../../../productBlueprint/domain/productBlueprintCategory";

import type { ProductBlueprintForCard } from "./types";

function normalizeProductBlueprintCategorySnapshot(
  value: unknown,
): ProductBlueprintCategorySnapshot | null {
  if (typeof value !== "object" || value === null) {
    return null;
  }

  return value as ProductBlueprintCategorySnapshot;
}

function getMemberFullName(
  member: Member,
): string {
  const displayName = member.displayName.trim();

  if (displayName) {
    return displayName;
  }

  return [
    member.lastName,
    member.firstName,
  ]
    .filter((value) => value.length > 0)
    .join(" ");
}

// ======================================================================
// ブランド（変換）
// ======================================================================
export function buildBrandOptions(
  brands: Brand[],
): string[] {
  return brands
    .map((brand) => brand.name)
    .filter(Boolean);
}

// ======================================================================
// 商品設計一覧（変換）
// ======================================================================
export function filterProductBlueprintsByBrand(
  rows: ProductBlueprintManagementRow[],
  brandName: string | null,
): ProductBlueprintManagementRow[] {
  if (!brandName) {
    return [];
  }

  return rows.filter(
    (productBlueprint) =>
      productBlueprint.brandName === brandName,
  );
}

export function buildProductRows(
  filtered: ProductBlueprintManagementRow[],
): Array<{ id: string; name: string }> {
  return filtered.map((productBlueprint) => ({
    id: productBlueprint.id,
    name: productBlueprint.productName,
  }));
}

// ======================================================================
// buildSelectedForCard（UIカード表示用）
// ======================================================================
// detail は productBlueprintDetailService 等から返る DTO を想定（現状 any を許容）
//
// 重要:
// - ProductBlueprintCard が期待する productBlueprintCategory は
//   string ではなく ProductBlueprintCategorySnapshot | null
// - ProductBlueprintCard が表示に使うブランド名は brandName
// - backend の正レスポンスではカテゴリ snapshot は
//   detail.productBlueprintCategory
// ======================================================================
export function buildSelectedForCard(
  detail: any,
  row: ProductBlueprintManagementRow | null,
): ProductBlueprintForCard {
  if (detail) {
    const productBlueprintCategory =
      normalizeProductBlueprintCategorySnapshot(
        detail.productBlueprintCategory,
      );

    return {
      id: String(
        detail.id ?? "",
      ).trim(),
      productName: String(
        detail.productName ?? "",
      ).trim(),
      brandName: String(
        detail.brandName ?? "",
      ).trim(),
      productBlueprintCategory,

      fit: detail.fit
        ? String(detail.fit).trim()
        : undefined,
      materials: detail.material
        ? String(detail.material).trim()
        : undefined,
      weight:
        typeof detail.weight === "number" &&
        Number.isFinite(detail.weight)
          ? detail.weight
          : undefined,
      washTags: Array.isArray(
        detail.qualityAssurance,
      )
        ? detail.qualityAssurance.filter(
            (
              tag: unknown,
            ): tag is string =>
              typeof tag === "string" &&
              tag.trim() !== "",
          )
        : undefined,
      productIdTag:
        String(
          detail.productIdTag?.type ?? "",
        ).trim() || undefined,
    };
  }

  if (row) {
    const productBlueprintCategory =
      normalizeProductBlueprintCategorySnapshot(
        (row as any).productBlueprintCategory,
      );

    return {
      id: String(
        row.id ?? "",
      ).trim(),
      productName: String(
        row.productName ?? "",
      ).trim(),
      brandName: String(
        row.brandName ?? "",
      ).trim(),
      productBlueprintCategory,
    };
  }

  return {
    id: "",
    productName: "",
    brandName: "",
    productBlueprintCategory: null,
  };
}

// ======================================================================
// 担当者一覧（変換）
// ======================================================================
// production 作成時に assigneeId として保存される値は、
// Firestore members の docId ではなく Firebase Auth UID を正とする。
// そのため option.id には member.id ではなく member.uid を入れる。
// ======================================================================
export function buildAssigneeOptions(
  members: Member[],
): Array<{ id: string; name: string }> {
  return members
    .map((member) => {
      const uid = member.uid.trim();

      return {
        id: uid,
        name:
          getMemberFullName(member) ||
          member.email ||
          uid ||
          member.id,
      };
    })
    .filter((option) => option.id);
}