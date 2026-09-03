// frontend/amol/src/features/shared/types/category.ts

export type ProductBlueprintCategoryFields = Record<string, unknown>;

export type ProductBlueprintCategoryRoot =
  | "apparel"
  | "alcohol"
  | "cosmetics"
  | "healthcare"
  | "other";

export type ProductCategoryKind = ProductBlueprintCategoryRoot | "unknown";