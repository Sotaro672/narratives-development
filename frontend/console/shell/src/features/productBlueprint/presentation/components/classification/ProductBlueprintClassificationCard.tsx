// frontend\console\shell\src\features\productBlueprint\presentation\cards\classification\ProductBlueprintClassificationCard.tsx

import * as React from "react";
import { Tags } from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../../shared/ui";

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

  productBlueprintCategoryPath: ProductBlueprintCategoryOption | null;
  productBlueprintCategoryOptions?: ProductBlueprintCategoryOption[];
  productBlueprintCategoryLoading?: boolean;
  productBlueprintCategoryError?: Error | null;
  onChangeProductBlueprintCategoryPath?: (
    productBlueprintCategoryPath: ProductBlueprintCategoryOption | null,
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

  productBlueprintCategoryPath,
  productBlueprintCategoryOptions,
  productBlueprintCategoryLoading,
  productBlueprintCategoryError,
  onChangeProductBlueprintCategoryPath,

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
          productBlueprintCategoryPath={productBlueprintCategoryPath}
          productBlueprintCategoryOptions={productBlueprintCategoryOptions}
          productBlueprintCategoryLoading={productBlueprintCategoryLoading}
          productBlueprintCategoryError={productBlueprintCategoryError}
          mode={mode}
          onChangeProductBlueprintCategoryPath={
            onChangeProductBlueprintCategoryPath
          }
        />
      </CardContent>
    </Card>
  );
};

export default ProductBlueprintClassificationCard;