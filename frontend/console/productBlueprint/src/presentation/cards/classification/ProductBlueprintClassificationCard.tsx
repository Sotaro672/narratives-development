// frontend\console\productBlueprint\src\presentation\cards\classification\ProductBlueprintClassificationCard.tsx

import * as React from "react";
import { Tags } from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../../shell/src/shared/ui";

import type { ProductBlueprintCategorySnapshot } from "../../../domain/entity/productBlueprintCategory";

import ProductBlueprintBrandField, {
  type BrandOption,
} from "./ProductBlueprintBrandField";

import ProductBlueprintCategoryField, {
  type ProductBlueprintCategoryOption,
} from "./ProductBlueprintCategoryField";

type ProductBlueprintClassificationCardProps = {
  brandId: string;
  brandName?: string;
  brandOptions?: BrandOption[];
  brandLoading?: boolean;
  brandError?: Error | null;
  onChangeBrandId?: (id: string) => void;

  productBlueprintCategoryId: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot | null;
  productBlueprintCategoryOptions?: ProductBlueprintCategoryOption[];
  productBlueprintCategoryLoading?: boolean;
  productBlueprintCategoryError?: Error | null;
  onChangeProductBlueprintCategory?: (
    category: ProductBlueprintCategorySnapshot | null,
  ) => void;

  mode?: "edit" | "view";
};

const ProductBlueprintClassificationCard: React.FC<
  ProductBlueprintClassificationCardProps
> = ({
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

  mode = "edit",
}) => {
  return (
    <Card className="pbc">
      <CardHeader className="box__header">
        <Tags size={16} />
        <CardTitle className="box__title">商品分類</CardTitle>
      </CardHeader>

      <CardContent className="box__body">
        <ProductBlueprintBrandField
          brandId={brandId}
          brandName={brandName}
          brandOptions={brandOptions}
          brandLoading={brandLoading}
          brandError={brandError}
          mode={mode}
          onChangeBrandId={onChangeBrandId}
        />

        <ProductBlueprintCategoryField
          categoryId={productBlueprintCategoryId}
          category={productBlueprintCategory}
          categoryOptions={productBlueprintCategoryOptions}
          categoryLoading={productBlueprintCategoryLoading}
          categoryError={productBlueprintCategoryError}
          mode={mode}
          onChangeCategory={onChangeProductBlueprintCategory}
        />
      </CardContent>
    </Card>
  );
};

export default ProductBlueprintClassificationCard;