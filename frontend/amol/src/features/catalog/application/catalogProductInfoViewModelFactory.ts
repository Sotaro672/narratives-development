// frontend/amol/src/features/catalog/application/catalogProductInfoViewModelFactory.ts

import type { CatalogProductBlueprint } from "../../shared/types/catalog";
import type { ProductCategoryKind } from "../../shared/types/category";

export type ProductInfoRowViewModel = {
  key: string;
  label: string;
  value: string;
};

export type ProductInfoCardViewModel = {
  rows: ProductInfoRowViewModel[];
  qualityAssuranceItems: string[];
};

function formatNullableText(value: unknown): string {
  if (typeof value === "string") return value.trim();
  if (typeof value === "number" && Number.isFinite(value)) return String(value);
  return "";
}

function formatAlcoholContent(value: unknown): string {
  if (typeof value === "number" && Number.isFinite(value)) return `${value}%`;

  if (typeof value === "string") {
    const text = value.trim();
    if (!text) return "";
    return text.includes("%") ? text : `${text}%`;
  }

  return "";
}

function resolveCategoryLabel(productBlueprint: CatalogProductBlueprint): string {
  return (
    productBlueprint.productBlueprintCategoryNameJa?.trim() ||
    productBlueprint.productBlueprintCategoryNameEn?.trim() ||
    productBlueprint.productBlueprintCategoryCode?.trim() ||
    ""
  );
}

function resolveQualityAssuranceItems(value: unknown): string[] {
  if (!Array.isArray(value)) return [];

  return value
    .map((item) => {
      if (typeof item === "string") return item.trim();
      if (!item || typeof item !== "object" || Array.isArray(item)) return "";

      const record = item as Record<string, unknown>;
      return formatNullableText(record.label) || formatNullableText(record.title) || formatNullableText(record.value);
    })
    .filter((item): item is string => item !== "");
}

function createRow(key: string, label: string, value: unknown): ProductInfoRowViewModel | null {
  const text = formatNullableText(value);
  if (!text) return null;

  return { key, label, value: text };
}

function createFormattedRow(key: string, label: string, value: string): ProductInfoRowViewModel | null {
  if (!value) return null;
  return { key, label, value };
}

function appendRow(rows: ProductInfoRowViewModel[], row: ProductInfoRowViewModel | null): void {
  if (row) rows.push(row);
}

export function createProductInfoCardViewModel(args: {
  productBlueprint: CatalogProductBlueprint;
  categoryKind?: ProductCategoryKind;
}): ProductInfoCardViewModel {
  const product = args.productBlueprint;
  const categoryKind = args.categoryKind ?? product.productBlueprintCategoryKind ?? "unknown";
  const isAlcohol = categoryKind === "alcohol";
  const isApparel = categoryKind === "apparel" || categoryKind === "unknown";
  const rows: ProductInfoRowViewModel[] = [];
  const categoryFields = product.categoryFields ?? null;

  appendRow(rows, createRow("productName", "商品名", product.productName));
  appendRow(rows, createRow("brandName", "ブランド", product.brandName));
  appendRow(rows, createRow("companyName", "会社名", product.companyName));
  appendRow(rows, createFormattedRow("category", "カテゴリ", resolveCategoryLabel(product)));

  if (isAlcohol) {
    appendRow(rows, createRow("material", "材料", categoryFields?.material));
    appendRow(rows, createRow("region", "生産地", categoryFields?.region));
    appendRow(rows, createRow("vintage", "ビンテージ", categoryFields?.vintage));
    appendRow(rows, createFormattedRow("alcoholContent", "アルコール度数", formatAlcoholContent(categoryFields?.alcoholContent)));
  }

  if (isApparel) {
    appendRow(rows, createRow("fit", "フィット", categoryFields?.fit));
    appendRow(rows, createRow("material", "素材", categoryFields?.material));
    appendRow(rows, createRow("weight", "重量", categoryFields?.weight));
  }

  appendRow(rows, createRow("productIdTagType", "商品IDタグ", product.productIdTagType));

  return {
    rows,
    qualityAssuranceItems: resolveQualityAssuranceItems(categoryFields?.qualityAssurance),
  };
}

export type { ProductCategoryKind };