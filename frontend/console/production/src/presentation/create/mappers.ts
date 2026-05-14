// frontend/console/production/src/presentation/create/mappers.ts

import type { Brand } from "../../../../brand/src/domain/entity/brand";
import type { Member } from "../../../../member/src/domain/entity/member";
import { getMemberFullName } from "../../../../member/src/domain/entity/member";

import type { ProductBlueprintManagementRow } from "../../../../productBlueprint/src/infrastructure/query/productBlueprintQuery";
import type { ModelVariationResponse } from "../../../../productBlueprint/src/application/productBlueprintDetailService";
import type { ProductBlueprintCategorySnapshot } from "../../../../productBlueprint/src/domain/entity/productBlueprintCategory";

import type { ProductBlueprintForCard, ProductionQuantityRow } from "./types";

function normalizeProductBlueprintCategorySnapshot(
  value: unknown,
): ProductBlueprintCategorySnapshot | null {
  if (typeof value !== "object" || value === null) {
    return null;
  }

  return value as ProductBlueprintCategorySnapshot;
}

function isApparelModelVariation(
  mv: ModelVariationResponse,
): mv is Extract<ModelVariationResponse, { kind: "apparel" }> {
  return (mv as any)?.kind === "apparel";
}

function isAlcoholModelVariation(
  mv: ModelVariationResponse,
): mv is Extract<ModelVariationResponse, { kind: "alcohol" }> {
  return (mv as any)?.kind === "alcohol";
}

function buildApparelVariationLabel(args: {
  size?: string;
  color?: string;
}): string {
  return [args.size, args.color]
    .map((v) => String(v ?? "").trim())
    .filter(Boolean)
    .join(" / ");
}

function buildAlcoholVariationLabel(args: {
  volumeValue?: unknown;
  volumeUnit?: unknown;
}): string {
  const value =
    typeof args.volumeValue === "number" && Number.isFinite(args.volumeValue)
      ? args.volumeValue
      : undefined;

  const unit = String(args.volumeUnit ?? "").trim();

  if (value === undefined || !unit) {
    return "";
  }

  return `${value}${unit}`;
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
//
// 重要:
// - ProductBlueprintCard が期待する productBlueprintCategory は
//   string ではなく ProductBlueprintCategorySnapshot | null
// - ProductBlueprintCard が表示に使うブランド名は brandName
// - backend の正レスポンスではカテゴリ snapshot は detail.productBlueprintCategory
// ======================================================================
export function buildSelectedForCard(
  detail: any,
  row: ProductBlueprintManagementRow | null,
): ProductBlueprintForCard {
  if (detail) {
    const productBlueprintCategory = normalizeProductBlueprintCategorySnapshot(
      detail.productBlueprintCategory,
    );

    return {
      id: String(detail.id ?? "").trim(),
      productName: String(detail.productName ?? "").trim(),
      brandName: String(detail.brandName ?? "").trim(),
      productBlueprintCategory,

      fit: detail.fit ? String(detail.fit).trim() : undefined,
      materials: detail.material ? String(detail.material).trim() : undefined,
      weight:
        typeof detail.weight === "number" && Number.isFinite(detail.weight)
          ? detail.weight
          : undefined,
      washTags: Array.isArray(detail.qualityAssurance)
        ? detail.qualityAssurance.filter(
            (tag: unknown): tag is string =>
              typeof tag === "string" && tag.trim() !== "",
          )
        : undefined,
      productIdTag: String(detail.productIdTag?.type ?? "").trim() || undefined,
    };
  }

  if (row) {
    const productBlueprintCategory = normalizeProductBlueprintCategorySnapshot(
      (row as any).productBlueprintCategory,
    );

    return {
      id: String(row.id ?? "").trim(),
      productName: String(row.productName ?? "").trim(),
      brandName: String(row.brandName ?? "").trim(),
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
// そのため option.id には m.id ではなく m.uid を入れる。
// ======================================================================
export function buildAssigneeOptions(
  members: Member[],
): Array<{ id: string; name: string }> {
  return members
    .map((m) => {
      const uid = String((m as any).uid ?? "").trim();

      return {
        id: uid,
        name: getMemberFullName(m) || m.email || uid || m.id,
      };
    })
    .filter((option) => option.id);
}

// ======================================================================
// ModelVariations → ProductionQuantityRow（UI入力用の行に変換）
// - modelId のみを正キーとして採用
// - displayOrder は detail.modelRefs 側が唯一のソースなので、ここでは注入しない
// - apparel / alcohol の discriminated union に対応
// ======================================================================
export function mapModelVariationsToRows(
  list: ModelVariationResponse[],
): ProductionQuantityRow[] {
  const safe = Array.isArray(list) ? list : [];

  return safe.map((mv, index) => {
    const modelId = String((mv as any)?.id ?? "").trim() || String(index);
    const modelNumber = String((mv as any)?.modelNumber ?? "").trim();

    if (isApparelModelVariation(mv)) {
      const size = String(mv.size ?? "").trim();
      const color = String(mv.color?.name ?? "").trim();
      const rgb = (mv.color?.rgb ?? null) as number | string | null;
      const variationLabel = buildApparelVariationLabel({ size, color });

      return {
        modelId,
        kind: "apparel",
        modelNumber,
        variationLabel,
        size,
        color,
        rgb,
        displayOrder: undefined,
        quantity: 0,
      };
    }

    if (isAlcoholModelVariation(mv)) {
      const volumeValueRaw = (mv as any).volume?.value;
      const volumeUnit = String((mv as any).volume?.unit ?? "").trim();

      const volumeValue =
        typeof volumeValueRaw === "number" && Number.isFinite(volumeValueRaw)
          ? volumeValueRaw
          : undefined;

      const variationLabel = buildAlcoholVariationLabel({
        volumeValue,
        volumeUnit,
      });

      return {
        modelId,
        kind: "alcohol",
        modelNumber,
        variationLabel,
        size: undefined,
        color: undefined,
        rgb: null,
        volumeValue,
        volumeUnit,
        displayOrder: undefined,
        quantity: 0,
      };
    }

    return {
      modelId,
      kind: String((mv as any)?.kind ?? "").trim() || undefined,
      modelNumber,
      variationLabel: "",
      size: undefined,
      color: undefined,
      rgb: null,
      displayOrder: undefined,
      quantity: 0,
    };
  });
}