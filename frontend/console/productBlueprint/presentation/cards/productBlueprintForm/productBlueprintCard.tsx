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
   * 編集モードでは右カラム/ブランドカード側で表示する。
   */
  brandName?: string;

  /**
   * 編集モードでは、子カテゴリまで選択された場合のみこのカードを表示する。
   * 閲覧モードでは既存画面への影響を避けるため、未設定でもカード表示する。
   */
  productBlueprintCategory?: ProductBlueprintCategorySnapshot | null;

  onChangeProductName?: (v: string) => void;

  mode?: "edit" | "view";
};

function hasText(value: unknown): boolean {
  return String(value ?? "").trim() !== "";
}

function getCategoryPath(
  category: ProductBlueprintCategorySnapshot | null | undefined,
): string[] {
  return Array.isArray(category?.path)
    ? category.path.map((item) => String(item ?? "").trim()).filter(Boolean)
    : [];
}

function isChildCategory(
  category: ProductBlueprintCategorySnapshot | null | undefined,
): boolean {
  if (!category) {
    return false;
  }

  if (hasText(category.parentId)) {
    return true;
  }

  return getCategoryPath(category).length > 1;
}

function resolveCardTitle(
  category: ProductBlueprintCategorySnapshot | null | undefined,
): string {
  const kind = String(category?.kind ?? "").trim();

  if (kind === "apparel") {
    return "基本情報（衣類）";
  }

  if (kind === "alcohol") {
    return "基本情報（酒類）";
  }

  if (kind === "cosmetics") {
    return "基本情報（化粧品）";
  }

  if (kind === "healthcare") {
    return "基本情報（ヘルスケア）";
  }

  if (kind === "other") {
    return "基本情報（その他）";
  }

  return "基本情報";
}

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

  /**
   * 編集モードでは、親カテゴリだけ選択された状態では基本情報カードを出さない。
   * 子カテゴリまで確定したら表示する。
   *
   * 閲覧モードは既存の detail / inventory / production / mintRequest 画面への影響を避けるため、
   * category が無くても表示する。
   */
  if (isEdit && !isChildCategory(mergedCategory)) {
    return null;
  }

  const mergedCategoryLabel =
    resolveProductBlueprintCategoryLabel(mergedCategory);

  const cardTitle = resolveCardTitle(mergedCategory);

  return (
    <Card className={`pbc ${!isEdit ? "view-mode" : ""}`}>
      <CardHeader className="box__header">
        <Package2 size={16} />
        <CardTitle className="box__title">{cardTitle}</CardTitle>
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