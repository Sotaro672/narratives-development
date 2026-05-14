// frontend/console/productBlueprint/src/presentation/cards/productBlueprintForm/productBlueprintCard.tsx

import * as React from "react";
import { Package2 } from "lucide-react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../../shell/src/shared/ui";
import { Input } from "../../../../../shell/src/shared/ui/input";

import type {
  CategoryFieldValues,
  ProductBlueprintCategorySnapshot,
} from "../../../domain/entity/productBlueprintCategory";

import ProductBlueprintBasicFields from "./ProductBlueprintBasicFields";
import { resolveProductBlueprintCategoryLabel } from "../classification/ProductBlueprintCategoryField";

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

  /**
   * 閲覧モードでは左カラムの基本情報カードに表示する。
   * 編集モードでは右カラムの ProductBlueprintClassificationCard 側で表示する。
   */
  brandName?: string;

  /**
   * 閲覧モードでは左カラムの基本情報カードに表示する。
   * 編集モードでは右カラムの ProductBlueprintClassificationCard 側で表示する。
   */
  productBlueprintCategory?: ProductBlueprintCategorySnapshot | null;

  onChangeProductName?: (v: string) => void;

  mode?: "edit" | "view";
};

const ProductBlueprintCard: React.FC<ProductBlueprintCardProps> = ({
  productBlueprintPatch,

  productName,
  brandName,

  productBlueprintCategory,

  onChangeProductName,

  mode = "edit",
}) => {
  const isEdit = mode === "edit";

  const mergedProductName =
    productName ?? productBlueprintPatch?.productName ?? "";

  const mergedBrandName = brandName ?? productBlueprintPatch?.brandName ?? "";

  const mergedCategory =
    productBlueprintCategory ??
    productBlueprintPatch?.productBlueprintCategory ??
    null;

  const mergedCategoryLabel =
    resolveProductBlueprintCategoryLabel(mergedCategory);

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

        {!isEdit && (
          <>
            <div className="label">ブランド</div>
            <Input
              value={mergedBrandName}
              variant="readonly"
              readOnly
              aria-label="ブランド"
            />

            <div className="label">商品カテゴリ</div>
            <Input
              value={mergedCategoryLabel}
              variant="readonly"
              readOnly
              aria-label="商品カテゴリ"
            />
          </>
        )}
      </CardContent>
    </Card>
  );
};

export default ProductBlueprintCard;