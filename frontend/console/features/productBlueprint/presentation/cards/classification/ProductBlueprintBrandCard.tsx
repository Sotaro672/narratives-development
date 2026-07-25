// frontend/console/productBlueprint/src/presentation/cards/classification/ProductBlueprintBrandCard.tsx

import * as React from "react";
import { BadgeCheck } from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../../shell/src/shared/ui";

import ProductBlueprintBrandField, {
  type BrandOption,
} from "./ProductBlueprintBrandField";

type ProductBlueprintBrandCardProps = {
  brandId: string;
  brandName?: string;
  brandOptions?: BrandOption[];
  brandLoading?: boolean;
  brandError?: Error | null;
  onChangeBrandId?: (id: string) => void;

  mode?: "edit" | "view";
};

const ProductBlueprintBrandCard: React.FC<ProductBlueprintBrandCardProps> = ({
  brandId,
  brandName,
  brandOptions,
  brandLoading,
  brandError,
  onChangeBrandId,

  mode = "edit",
}) => {
  return (
    <Card className="pbc">
      <CardHeader className="box__header">
        <BadgeCheck size={16} />
        <CardTitle className="box__title">ブランド選択</CardTitle>
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
      </CardContent>
    </Card>
  );
};

export default ProductBlueprintBrandCard;