// frontend/amol/src/features/shared/types/category.ts

export type ProductBlueprintCategoryFields =
  Record<string, unknown>;

export type ProductCategoryKind =
  | "apparel"
  | "alcohol"
  | "cosmetics"
  | "healthcare"
  | "other"
  | "unknown"
  | (string & {});