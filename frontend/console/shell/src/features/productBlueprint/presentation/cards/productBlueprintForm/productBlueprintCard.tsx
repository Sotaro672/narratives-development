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
  ProductBlueprintCategoryPath,
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

  productBlueprintCategoryPath?: ProductBlueprintCategoryPath | null;

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

  productBlueprintCategoryPath?: ProductBlueprintCategoryPath | null;
  productBlueprintCategoryOptions?: ProductBlueprintCategoryOption[];
  productBlueprintCategoryLoading?: boolean;
  productBlueprintCategoryError?: Error | null;
  onChangeProductBlueprintCategoryPath?: (
    productBlueprintCategoryPath: ProductBlueprintCategoryPath | null,
  ) => void;

  onChangeProductName?: (value: string) => void;

  mode?: "edit" | "view";
};

function resolveCardTitle(
  productBlueprintCategoryPath:
    ProductBlueprintCategoryPath | null | undefined,
): string {
  const root =
    productBlueprintCategoryPath?.[0] ?? "";

  if (root === "apparel") {
    return "基本情報（衣類）";
  }

  if (root === "alcohol") {
    return "基本情報（酒類）";
  }

  if (root === "cosmetics") {
    return "基本情報（化粧品）";
  }

  if (root === "healthcare") {
    return "基本情報（ヘルスケア）";
  }

  if (root === "other") {
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

  productBlueprintCategoryPath,
  productBlueprintCategoryOptions,
  productBlueprintCategoryLoading,
  productBlueprintCategoryError,
  onChangeProductBlueprintCategoryPath,

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

  const mergedProductBlueprintCategoryPath =
    productBlueprintCategoryPath ??
    productBlueprintPatch?.productBlueprintCategoryPath ??
    null;

  const cardTitle = resolveCardTitle(
    mergedProductBlueprintCategoryPath,
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
          productBlueprintCategoryPath={
            mergedProductBlueprintCategoryPath
          }
          productBlueprintCategoryOptions={
            productBlueprintCategoryOptions
          }
          productBlueprintCategoryLoading={
            productBlueprintCategoryLoading
          }
          productBlueprintCategoryError={
            productBlueprintCategoryError
          }
          mode={mode}
          onChangeProductBlueprintCategoryPath={
            isEdit
              ? onChangeProductBlueprintCategoryPath
              : undefined
          }
        />
      </CardContent>
    </Card>
  );
};

export default ProductBlueprintCard;