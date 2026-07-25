// frontend/console/productBlueprint/src/presentation/cards/classification/ProductBlueprintCategoryCard.tsx

import * as React from "react";
import { Tags } from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../../shell/src/shared/ui";

import type { ProductBlueprintCategorySnapshot } from "../../../domain/entity/productBlueprintCategory";

import ProductBlueprintCategoryField, {
  type ProductBlueprintCategoryOption,
} from "./ProductBlueprintCategoryField";

type ProductBlueprintCategoryCardProps = {
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

const ProductBlueprintCategoryCard: React.FC<ProductBlueprintCategoryCardProps> = ({
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
        <CardTitle className="box__title">商品カテゴリ</CardTitle>
      </CardHeader>

      <CardContent className="box__body">
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

export default ProductBlueprintCategoryCard;