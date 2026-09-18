// frontend/console/shell/src/features/productBlueprint/presentation/cards/productBlueprintForm/ProductBlueprintBasicFields.tsx

import * as React from "react";

import { CardField } from "../../../../../shared/ui/card";
import { Input } from "../../../../../shared/ui/input";
import { Label } from "../../../../../shared/ui/label";

type ProductBlueprintBasicFieldsProps = {
  productName: string;
  mode?: "edit" | "view";
  onChangeProductName?: (v: string) => void;
};

const ProductBlueprintBasicFields: React.FC<ProductBlueprintBasicFieldsProps> = ({
  productName,
  mode = "edit",
  onChangeProductName,
}) => {
  const isEdit = mode === "edit";

  return (
    <CardField>
      <Label htmlFor="product-blueprint-product-name">
        プロダクト名
      </Label>

      {isEdit ? (
        <Input
          id="product-blueprint-product-name"
          value={productName}
          onChange={(event) => onChangeProductName?.(event.target.value)}
        />
      ) : (
        <Input
          id="product-blueprint-product-name"
          value={productName}
          variant="readonly"
          readOnly
        />
      )}
    </CardField>
  );
};

export default ProductBlueprintBasicFields;