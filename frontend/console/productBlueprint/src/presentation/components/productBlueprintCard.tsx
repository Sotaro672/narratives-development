// frontend/console/productBlueprint/src/presentation/components/productBlueprintCard.tsx

import * as React from "react";
import { Package2 } from "lucide-react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shell/src/shared/ui";

import type {
  CategoryFieldValue,
  CategoryFieldValues,
  ProductBlueprintCategorySnapshot,
} from "../../domain/entity/productBlueprintCategory";

import type { Fit } from "../../domain/entity/apparel";

import {
  getCategoryCardVisibility,
} from "../../domain/entity/categoryCardVisibility";

import ProductBlueprintBasicFields from "./ProductBlueprintBasicFields";
import ProductBlueprintBrandField, {
  type BrandOption,
} from "./ProductBlueprintBrandField";
import ProductBlueprintCategoryField, {
  type ProductBlueprintCategoryOption,
} from "./ProductBlueprintCategoryField";
import CategoryFieldsCard from "./CategoryFieldsCard";
import WashTagField from "./WashTagField";

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

type ProductBlueprintCardProps = {
  productBlueprintPatch?: ProductBlueprintPatchInput;

  productName?: string;
  brand?: string;

  brandId?: string;
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

  fit?: Fit;
  materials?: string;
  weight?: number;
  washTags?: string[];

  categoryFields?: CategoryFieldValues | null;
  onChangeCategoryField?: (key: string, value: CategoryFieldValue) => void;

  onChangeProductName?: (v: string) => void;
  onChangeFit?: (v: Fit) => void;
  onChangeMaterials?: (v: string) => void;
  onChangeWeight?: (v: number) => void;
  onChangeWashTags?: (nextTags: string[]) => void;

  mode?: "edit" | "view";
};

function getCategoryCode(
  category: ProductBlueprintCategorySnapshot | null | undefined,
): string {
  return String(category?.code ?? "").trim();
}

const ProductBlueprintCard: React.FC<ProductBlueprintCardProps> = ({
  productBlueprintPatch,

  productName,
  brand,
  brandId,
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

  fit,
  materials,
  weight,
  washTags,

  categoryFields,
  onChangeCategoryField,

  onChangeProductName,
  onChangeFit,
  onChangeMaterials,
  onChangeWeight,
  onChangeWashTags,

  mode = "edit",
}) => {
  const isEdit = mode === "edit";

  const mergedProductName =
    productName ?? productBlueprintPatch?.productName ?? "";

  const mergedBrandId = brandId ?? productBlueprintPatch?.brandId ?? "";
  const mergedBrandName = brand ?? productBlueprintPatch?.brandName ?? "";

  const mergedCategory =
    productBlueprintCategory ??
    productBlueprintPatch?.productBlueprintCategory ??
    null;

  const mergedCategoryId =
    productBlueprintCategoryId ??
    productBlueprintPatch?.productBlueprintCategoryId ??
    mergedCategory?.id ??
    "";

  const mergedCategoryFields =
    categoryFields ?? productBlueprintPatch?.categoryFields ?? null;

  const categoryCode = getCategoryCode(mergedCategory);

  const visibility = React.useMemo(
    () => getCategoryCardVisibility(categoryCode),
    [categoryCode],
  );

  const mergedFit =
    fit ??
    ((typeof productBlueprintPatch?.fit === "string"
      ? (productBlueprintPatch.fit as Fit)
      : undefined) as Fit | undefined) ??
    ("" as Fit);

  const mergedMaterials =
    materials ??
    (typeof mergedCategoryFields?.material === "string"
      ? mergedCategoryFields.material
      : undefined) ??
    productBlueprintPatch?.material ??
    "";

  const mergedWeight =
    typeof weight === "number"
      ? weight
      : typeof mergedCategoryFields?.weight === "number"
        ? mergedCategoryFields.weight
        : typeof productBlueprintPatch?.weight === "number"
          ? productBlueprintPatch.weight
          : 0;

  const mergedWashTags = Array.isArray(washTags)
    ? washTags
    : Array.isArray(productBlueprintPatch?.qualityAssurance)
      ? productBlueprintPatch.qualityAssurance.filter(
          (x): x is string => typeof x === "string" && x.trim() !== "",
        )
      : [];

  return (
    <Card className={`pbc ${!isEdit ? "view-mode" : ""}`}>
      <CardHeader className="box__header">
        <Package2 size={16} />
        <CardTitle className="box__title">基本情報</CardTitle>
      </CardHeader>

      <CardContent className="box__body">
        <ProductBlueprintBasicFields
          productName={mergedProductName}
          mode={mode}
          onChangeProductName={onChangeProductName}
        />

        <ProductBlueprintBrandField
          brandId={mergedBrandId}
          brandName={mergedBrandName}
          brandOptions={brandOptions}
          brandLoading={brandLoading}
          brandError={brandError}
          mode={mode}
          onChangeBrandId={onChangeBrandId}
        />

        <ProductBlueprintCategoryField
          categoryId={mergedCategoryId}
          category={mergedCategory}
          categoryOptions={productBlueprintCategoryOptions}
          categoryLoading={productBlueprintCategoryLoading}
          categoryError={productBlueprintCategoryError}
          mode={mode}
          onChangeCategory={onChangeProductBlueprintCategory}
        />

        <CategoryFieldsCard
          categoryCode={categoryCode}
          fit={mergedFit}
          material={String(mergedMaterials ?? "")}
          weight={
            typeof mergedWeight === "number" && !Number.isNaN(mergedWeight)
              ? mergedWeight
              : 0
          }
          categoryFields={mergedCategoryFields}
          mode={mode}
          onChangeFit={onChangeFit}
          onChangeMaterials={onChangeMaterials}
          onChangeWeight={onChangeWeight}
          onChangeCategoryField={onChangeCategoryField}
        />

        {visibility.showWashTags && (
          <WashTagField
            value={mergedWashTags}
            mode={mode}
            onChange={onChangeWashTags}
          />
        )}
      </CardContent>
    </Card>
  );
};

export default ProductBlueprintCard;