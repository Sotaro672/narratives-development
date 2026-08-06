// frontend\console\shell\src\features\productBlueprint\presentation\cards\productBlueprintForm\productBlueprintCard.tsx

import * as React from "react";
import { Package2 } from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../../shared/ui";

import type {
  CategoryFieldValues,
  ProductBlueprintCategorySnapshot,
} from "../../../domain/productBlueprintCategory";

import ProductBlueprintBrandField, {
  type BrandOption,
} from "../classification/ProductBlueprintBrandField";

import ProductBlueprintCategoryField, {
  type ProductBlueprintCategoryOption,
} from "../classification/ProductBlueprintCategoryField";

import ProductBlueprintBasicFields from "./ProductBlueprintBasicFields";

export type ProductBlueprintPatchInput = {
  productName?: string | null;
  brandId?: string | null;
  brandName?: string | null;

  productBlueprintCategoryId?: string | null;
  productBlueprintCategory?: ProductBlueprintCategorySnapshot | null;

  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;
  categoryFields?: CategoryFieldValues | null;

  assigneeId?: string | null;
};

export type ProductBlueprintCardProps = {
  productBlueprintPatch?: ProductBlueprintPatchInput;

  productName?: string;

  brandId?: string;
  brandName?: string;
  brandOptions?: BrandOption[];
  brandLoading?: boolean;
  brandError?: Error | null;
  onChangeBrandId?: (id: string) => void;

  productBlueprintCategoryId?: string;
  productBlueprintCategory?: ProductBlueprintCategorySnapshot | null;
  productBlueprintCategoryOptions?: ProductBlueprintCategoryOption[];
  productBlueprintCategoryLoading?: boolean;
  productBlueprintCategoryError?: Error | null;
  onChangeProductBlueprintCategory?: (
    category: ProductBlueprintCategorySnapshot | null,
  ) => void;

  onChangeProductName?: (value: string) => void;

  mode?: "edit" | "view";
};

function resolveCardTitle(
  category: ProductBlueprintCategorySnapshot | null | undefined,
): string {
  const kind = category?.kind ?? "";

  if (kind === "apparel") {
    return "基本情報（衣類）";
  }

  if (kind === "alcohol") {
    return "基本情報（酒類）";
  }

  if (kind === "cosmetics") {
    return "基本情報（化粧品）";
  }

  if (kind === "healthcare") {
    return "基本情報（ヘルスケア）";
  }

  if (kind === "other") {
    return "基本情報（その他）";
  }

  return "基本情報";
}

const ProductBlueprintCard: React.FC<ProductBlueprintCardProps> = ({
  productBlueprintPatch,

  productName,

  brandId,
  brandName,
  brandOptions,
  brandLoading,
  brandError,
  onChangeBrandId,

  productBlueprintCategoryId,
  productBlueprintCategory,
  productBlueprintCategoryOptions,
  productBlueprintCategoryLoading,
  productBlueprintCategoryError,
  onChangeProductBlueprintCategory,

  onChangeProductName,

  mode = "edit",
}) => {
  const isEdit = mode === "edit";

  const mergedProductName =
    productName ??
    productBlueprintPatch?.productName ??
    "";

  const mergedBrandId =
    brandId ??
    productBlueprintPatch?.brandId ??
    "";

  const mergedBrandName =
    brandName ??
    productBlueprintPatch?.brandName ??
    "";

  const mergedProductBlueprintCategoryId =
    productBlueprintCategoryId ??
    productBlueprintPatch?.productBlueprintCategoryId ??
    "";

  const mergedProductBlueprintCategory =
    productBlueprintCategory ??
    productBlueprintPatch?.productBlueprintCategory ??
    null;

  const cardTitle = resolveCardTitle(
    mergedProductBlueprintCategory,
  );

  return (
    <Card
      className={`pbc ${
        !isEdit
          ? "view-mode"
          : ""
      }`}
    >
      <CardHeader className="box__header">
        <Package2 size={16} />

        <CardTitle className="box__title">
          {cardTitle}
        </CardTitle>
      </CardHeader>

      <CardContent className="box__body">
        <ProductBlueprintBasicFields
          productName={mergedProductName}
          mode={mode}
          onChangeProductName={
            isEdit
              ? onChangeProductName
              : undefined
          }
        />

        <ProductBlueprintBrandField
          brandId={mergedBrandId}
          brandName={mergedBrandName}
          brandOptions={brandOptions}
          brandLoading={brandLoading}
          brandError={brandError}
          mode={mode}
          onChangeBrandId={
            isEdit
              ? onChangeBrandId
              : undefined
          }
        />

        <ProductBlueprintCategoryField
          categoryId={
            mergedProductBlueprintCategoryId
          }
          category={
            mergedProductBlueprintCategory
          }
          categoryOptions={
            productBlueprintCategoryOptions
          }
          categoryLoading={
            productBlueprintCategoryLoading
          }
          categoryError={
            productBlueprintCategoryError
          }
          mode={mode}
          onChangeCategory={
            isEdit
              ? onChangeProductBlueprintCategory
              : undefined
          }
        />
      </CardContent>
    </Card>
  );
};

export default ProductBlueprintCard;