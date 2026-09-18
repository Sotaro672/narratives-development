// frontend/console/shell/src/features/productBlueprint/presentation/cards/classification/ProductBlueprintCategoryCard.tsx

import * as React from "react";
import { Tags } from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../../shared/ui";

import ProductBlueprintCategoryField, {
  type ProductBlueprintCategoryOption,
} from "./ProductBlueprintCategoryField";

type ProductBlueprintCategoryCardProps = {
  productBlueprintCategoryPath: ProductBlueprintCategoryOption | null;
  productBlueprintCategoryOptions?: ProductBlueprintCategoryOption[];
  productBlueprintCategoryLoading?: boolean;
  productBlueprintCategoryError?: Error | null;
  onChangeProductBlueprintCategoryPath?: (
    productBlueprintCategoryPath: ProductBlueprintCategoryOption | null,
  ) => void;

  mode?: "edit" | "view";
};

const ProductBlueprintCategoryCard: React.FC<ProductBlueprintCategoryCardProps> = ({
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
        <CardTitle className="box__title">商品カテゴリ</CardTitle>
      </CardHeader>

      <CardContent className="box__body">
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

export default ProductBlueprintCategoryCard;