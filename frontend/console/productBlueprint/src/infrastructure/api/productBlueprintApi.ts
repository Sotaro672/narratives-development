// frontend/console/productBlueprint/src/infrastructure/api/productBlueprintApi.ts

import { PRODUCT_BLUEPRINTS } from "../mockdata/productBlueprint_mockdata";
import {
  MODEL_NUMBERS,
  SIZE_VARIATIONS,
} from "../../../../model/src/infrastructure/mockdata/mockdata";
import type { ProductBlueprint } from "../../../../shell/src/shared/types/productBlueprint";

// BrandID → 表示名（モック用マッピング）
export const brandLabelFromId = (brandId: string): string => {
  switch (brandId) {
    case "brand_lumina":
      return "LUMINA Fashion";
    case "brand_nexus":
      return "NEXUS Street";
    default:
      return brandId || "-";
  }
};

// ISO8601 → "YYYY/MM/DD"（壊れてたらそのまま返す） ※一覧用
const toDisplayDate = (iso?: string | null): string => {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}/${m}/${day}`;
};

// ISO8601 → "YYYY/M/D" 表示 ※詳細画面用（元の挙動を維持）
export const formatProductBlueprintDate = (iso?: string | null): string => {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const y = d.getFullYear();
  const m = d.getMonth() + 1;
  const day = d.getDate();
  return `${y}/${m}/${day}`;
};

// 一覧表示用のUI行モデル（API が返す形）
export type ProductBlueprintListRow = {
  id: string;
  productName: string;
  brandLabel: string;
  assigneeLabel: string;
  tagLabel: string;
  createdAt: string; // YYYY/MM/DD
  lastModifiedAt: string; // YYYY/MM/DD
};

// 詳細画面用：サイズ行モデル
export type SizeRow = {
  id: string;
  sizeLabel: string;
  chest: number;
  waist: number;
  length: number;
  shoulder: number;
};

// 詳細画面用：モデルナンバー行モデル
export type ModelNumberRow = {
  size: string;
  color: string;
  code: string;
};

/**
 * 商品設計一覧用の行データを取得する API（現在はモック）
 * - backend の ProductBlueprint エンティティを UI 用行データへ変換する責務を持つ
 * - ソフトデリート済み（deletedAt が truthy）のものは一覧から除外
 */
export function fetchProductBlueprintListRows(): ProductBlueprintListRow[] {
  return (PRODUCT_BLUEPRINTS as ProductBlueprint[])
    .filter((pb) => !pb.deletedAt) // deletedAt が設定されているものは除外
    .map((pb) => ({
      id: pb.id,
      productName: pb.productName,
      brandLabel: brandLabelFromId(pb.brandId),
      assigneeLabel: pb.assigneeId || "-",
      // entity.go 準拠: Tag は ProductIdTag (struct) を保持し、その type を表示
      tagLabel:
        pb.productIdTag && pb.productIdTag.type
          ? pb.productIdTag.type.toUpperCase()
          : "-",
      createdAt: toDisplayDate(pb.createdAt),
      // entity.go 準拠: 最終更新日時は UpdatedAt
      lastModifiedAt: toDisplayDate(pb.updatedAt),
    }));
}

/**
 * ID から ProductBlueprint を取得（現在はモック配列を探索）
 * - ソフトデリート済み（deletedAt が truthy）のものは取得対象外
 */
export function fetchProductBlueprintById(
  blueprintId?: string,
): ProductBlueprint | undefined {
  if (!blueprintId) return undefined;
  return (PRODUCT_BLUEPRINTS as ProductBlueprint[]).find(
    (pb) => pb.id === blueprintId && !pb.deletedAt,
  );
}

/**
 * 詳細画面用：サイズ行データを取得（現在は SIZE_VARIATIONS から復元）
 */
export function fetchProductBlueprintSizeRows(): SizeRow[] {
  return SIZE_VARIATIONS.map((v, i) => ({
    id: String(i + 1),
    sizeLabel: v.size,
    chest: v.measurements["身幅"] ?? 0,
    waist: v.measurements["ウエスト"] ?? 0,
    length: v.measurements["着丈"] ?? 0,
    shoulder: v.measurements["肩幅"] ?? 0,
  }));
}

/**
 * 詳細画面用：モデルナンバー行データを取得（現在は MODEL_NUMBERS から復元）
 */
export function fetchProductBlueprintModelNumberRows(): ModelNumberRow[] {
  return MODEL_NUMBERS.map((m) => ({
    size: m.size,
    color: m.color,
    code: m.modelNumber,
  }));
}
