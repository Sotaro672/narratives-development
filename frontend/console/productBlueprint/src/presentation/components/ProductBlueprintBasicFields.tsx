// frontend/console/productBlueprint/src/presentation/components/ProductBlueprintBasicFields.tsx

import * as React from "react";
import { Input } from "../../../../shell/src/shared/ui/input";

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
    <>
      <div className="label">プロダクト名</div>
      {isEdit ? (
        <Input
          value={productName}
          onChange={(e) => onChangeProductName?.(e.target.value)}
          aria-label="プロダクト名"
        />
      ) : (
        <Input
          value={productName}
          variant="readonly"
          readOnly
          aria-label="プロダクト名"
        />
      )}
    </>
  );
};

export default ProductBlueprintBasicFields;