// frontend/console/shell/src/shared/types/production.ts

import type {
  CategoryFieldValues,
  ProductBlueprintCategoryPath,
} from "../../features/productBlueprint/domain/productBlueprintCategory";

// ======================================================================
// Production Quantity
// ======================================================================

export type ProductionQuantityRow = {
  modelId: string;
  quantity: number;
  kind?: "apparel" | "alcohol";
  modelNumber?: string;
  size?: string;
  color?: string;
  rgb?: number;
  volumeValue?: number;
  volumeUnit?: string;
  displayOrder?: number;
};

// ======================================================================
// Internal Shared Fields
// ======================================================================

type ProductionAuditFields = {
  printed: boolean;
  printedAt?: string | null;
  printedBy?: string | null;
  createdBy?: string | null;
  createdAt?: string | null;
  updatedBy?: string | null;
  updatedAt?: string | null;
};

type ProductionResolvedNames = {
  assigneeName: string;
  printedByName: string;
  createdByName: string;
  updatedByName: string;
};

// ======================================================================
// Production
// ======================================================================

export type Production = ProductionAuditFields & {
  id: string;
  productBlueprintId: string;
  assigneeId: string;
  models: ProductionQuantityRow[];
};

// ======================================================================
// Create
// ======================================================================

export type CreateProductionRequest = Pick<
  Production,
  "productBlueprintId" | "assigneeId" | "models"
>;

export type ProductionCreateProductBlueprint = {
  id: string;
  productName: string;
  brandId: string;
  brandName: string;
  productBlueprintCategoryPath: ProductBlueprintCategoryPath;
  categoryFields?: CategoryFieldValues | null;
  assigneeId?: string;
};

export type ProductionCreateContext = {
  productBlueprintPatch: ProductionCreateProductBlueprint;
  rows: ProductionQuantityRow[];
};

// ======================================================================
// Update
// ======================================================================

export type UpdateProductionRequest = Pick<
  Production,
  "assigneeId" | "models"
>;

// ======================================================================
// Detail
// ======================================================================

export type ProductionDetail = Production &
  ProductionResolvedNames & {
    productName: string;
    productBlueprintCategoryPath: ProductBlueprintCategoryPath;
    brandId: string;
    brandName: string;
    totalQuantity: number;
  };

// ======================================================================
// List
// ======================================================================

export type ProductionListItem = Production &
  ProductionResolvedNames & {
    productName: string;
    brandName: string;
    totalQuantity: number;
  };

export type ProductionListRow = ProductionListItem & {
  printedAtLabel: string;
  createdAtLabel: string;
};

export type ProductionListRowView = Pick<
  ProductionListRow,
  | "id"
  | "productBlueprintId"
  | "productName"
  | "assigneeId"
  | "assigneeName"
  | "printed"
  | "totalQuantity"
  | "printedAtLabel"
  | "createdAtLabel"
  | "brandName"
>;

// ======================================================================
// List Sort
// ======================================================================

export type ProductionSortKey =
  | "printedAt"
  | "createdAt"
  | "totalQuantity"
  | null;

export type ProductionSortDirection = "asc" | "desc" | null;