// frontend/amol/src/features/catalog/application/catalogProductInfoMapper.ts

import type { CatalogProductBlueprint } from "../types";

export type ProductCategoryKind = "apparel" | "alcohol" | "unknown";

export type CatalogProductBlueprintDisplayFields = CatalogProductBlueprint & {
  category?: string | null;
  categoryCode?: string | null;
  classification?: string | null;
  region?: string | null;
  vintage?: string | number | null;
  alcoholContent?: string | number | null;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object";
}

export function isNonEmptyText(value: unknown): value is string {
  return typeof value === "string" && value.trim() !== "";
}

export function formatNullableText(value: unknown): string {
  if (typeof value === "string") {
    return value.trim();
  }

  if (typeof value === "number" && Number.isFinite(value)) {
    return String(value);
  }

  return "";
}

export function formatWeight(value: unknown): string {
  if (typeof value !== "number" || !Number.isFinite(value) || value <= 0) {
    return "";
  }

  return `${value}g`;
}

export function formatAlcoholContent(value: unknown): string {
  if (typeof value === "number" && Number.isFinite(value)) {
    return `${value}%`;
  }

  if (typeof value === "string") {
    const text = value.trim();

    if (!text) {
      return "";
    }

    return text.includes("%") ? text : `${text}%`;
  }

  return "";
}

export function resolveCategoryLabel(
  productBlueprint: CatalogProductBlueprintDisplayFields,
): string {
  return (
    formatNullableText(productBlueprint.category) ||
    formatNullableText(productBlueprint.categoryCode) ||
    formatNullableText(productBlueprint.classification)
  );
}

export function resolveQualityAssuranceItems(
  qualityAssurance: unknown,
): string[] {
  if (Array.isArray(qualityAssurance)) {
    return qualityAssurance
      .map((item) => {
        if (typeof item === "string") {
          return item.trim();
        }

        if (isRecord(item)) {
          const label = item.label;
          const title = item.title;
          const value = item.value;

          if (typeof label === "string" && label.trim() !== "") {
            return label.trim();
          }

          if (typeof title === "string" && title.trim() !== "") {
            return title.trim();
          }

          if (typeof value === "string" && value.trim() !== "") {
            return value.trim();
          }
        }

        return "";
      })
      .filter((item): item is string => item !== "");
  }

  if (typeof qualityAssurance === "string") {
    const text = qualityAssurance.trim();
    return text ? [text] : [];
  }

  return [];
}