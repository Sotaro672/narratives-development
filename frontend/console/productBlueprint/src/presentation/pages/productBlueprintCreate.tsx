// frontend/console/productBlueprint/src/presentation/pages/productBlueprintCreate.tsx

import * as React from "react";

import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import { AdminCard } from "../../../../admin/src/presentation/components/AdminCard";
import ProductBlueprintCard from "../cards/productBlueprintForm";
import {
  ProductBlueprintBrandCard,
  ProductBlueprintCategoryCard,
} from "../cards/classification";
import CategoryFieldsCard from "../cards/categoryFields";
import ColorVariationCard from "../../../../model/src/presentation/components/ColorVariationCard";
import SizeVariationCard from "../../../../model/src/presentation/components/SizeVariationCard";
import ModelNumberCard from "../../../../model/src/presentation/components/ModelNumberCard";

// モデルナンバー用のロジックは model 側の hook を利用
import { useModelCard } from "../../../../model/src/presentation/hook/useModelCard";

import { useProductBlueprintCreate } from "../hooks/create/useProductBlueprintCreate";

import type { CategoryFieldValues } from "../../domain/entity/productBlueprintCategory";

function shouldShowModelVariationCards(categoryCode: string): boolean {
  return (
    categoryCode === "apparel.tops" ||
    categoryCode === "apparel.bottoms" ||
    categoryCode === "apparel.dress" ||
    categoryCode === "apparel.outerwear" ||
    categoryCode === "apparel.shoes"
  );
}

function toSafeNumber(value: unknown, fallback = 0): number {
  return typeof value === "number" && !Number.isNaN(value) ? value : fallback;
}

function toSafeStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return [];

  return value.filter(
    (item): item is string => typeof item === "string" && item.trim() !== "",
  );
}

export default function ProductBlueprintCreate() {
  const {
    // ブランド
    brandId,
    brandName,
    brandOptions,
    brandLoading,
    brandError,
    onChangeBrandId,

    // 商品設計フィールド
    productName,
    productBlueprintCategoryId,
    productBlueprintCategory,
    productBlueprintCategoryLabel,
    productBlueprintCategoryOptions,
    productBlueprintCategoryLoading,
    productBlueprintCategoryError,
    isApparelCategory,
    fit,
    material,
    weight,
    qualityAssurance,
    categoryFields,

    // 商品カテゴリから導出された採寸項目
    measurementOptions,

    // バリエーション
    colorInput,
    colors,
    colorRgbMap,
    sizes,
    modelNumbers,

    onChangeProductName,
    onChangeProductBlueprintCategory,
    onChangeFit,
    onChangeMaterial,
    onChangeWeight,
    onChangeQualityAssurance,
    onChangeCategoryField,

    onChangeColorInput,
    onAddColor,
    onRemoveColor,
    onChangeColorRgb,

    // サイズ操作
    onAddSize,
    onRemoveSize,
    onChangeSize,

    // モデルナンバー操作（アプリケーション層）
    onChangeModelNumber,

    // 管理情報
    assigneeId,
    assigneeName,
    onSelectAssignee,
    onEditAssignee,
    onClickAssignee,

    // 画面アクション
    onCreate,
    onBack,
  } = useProductBlueprintCreate();

  const categoryCode = String(productBlueprintCategory?.code ?? "").trim();

  const mergedCategoryFields = React.useMemo<CategoryFieldValues>(() => {
    return {
      ...(categoryFields ?? {}),
      fit,
      material: String(material ?? ""),
      weight: toSafeNumber(weight, 0),
      washTags: toSafeStringArray(qualityAssurance),
    };
  }, [categoryFields, fit, material, weight, qualityAssurance]);

  const showModelVariationCards = React.useMemo(
    () => isApparelCategory && shouldShowModelVariationCards(categoryCode),
    [isApparelCategory, categoryCode],
  );

  // -----------------------------
  // モデルナンバー表示用の hook（model 側）
  // rgb を hook 経由で渡すため colorRgbMap も渡す
  // -----------------------------
  const { getCode, onChangeModelNumber: uiOnChangeModelNumber } = useModelCard({
    sizes,
    colors,
    modelNumbers,
    colorRgbMap,
  });

  // UI 変更時に「model 側の内部状態」と「productBlueprintCreate の状態」の両方を更新
  const handleChangeModelNumber = React.useCallback(
    (sizeLabel: string, color: string, nextCode: string) => {
      uiOnChangeModelNumber(sizeLabel, color, nextCode);
      onChangeModelNumber(sizeLabel, color, nextCode);
    },
    [uiOnChangeModelNumber, onChangeModelNumber],
  );

  return (
    <PageStyle
      layout="grid-2"
      title="商品設計を作成"
      onBack={onBack}
      onSave={onCreate}
    >
      <div className="space-y-4">
        <ProductBlueprintCategoryCard
          mode="edit"
          productBlueprintCategoryId={productBlueprintCategoryId}
          productBlueprintCategory={productBlueprintCategory}
          productBlueprintCategoryOptions={productBlueprintCategoryOptions}
          productBlueprintCategoryLoading={productBlueprintCategoryLoading}
          productBlueprintCategoryError={productBlueprintCategoryError}
          onChangeProductBlueprintCategory={onChangeProductBlueprintCategory}
        />

        <ProductBlueprintCard
          mode="edit"
          productName={productName}
          productBlueprintCategory={productBlueprintCategory}
          onChangeProductName={onChangeProductName}
        />

        {!productBlueprintCategory && (
          <p className="mt-2 text-xs text-slate-500">
            商品カテゴリを選択すると、カテゴリに応じた入力欄が表示されます。
          </p>
        )}

        {productBlueprintCategory && (
          <CategoryFieldsCard
            categoryCode={categoryCode}
            categoryFields={mergedCategoryFields}
            mode="edit"
            onChangeCategoryField={onChangeCategoryField}
          />
        )}

        {productBlueprintCategory && !showModelVariationCards && (
          <p className="mt-2 text-xs text-slate-500">
            選択中の商品カテゴリ: {productBlueprintCategoryLabel}
          </p>
        )}

        {showModelVariationCards && (
          <>
            <ColorVariationCard
              colors={colors}
              colorInput={colorInput}
              onChangeColorInput={onChangeColorInput}
              onAddColor={onAddColor}
              onRemoveColor={onRemoveColor}
              colorRgbMap={colorRgbMap}
              onChangeColorRgb={onChangeColorRgb}
            />

            <SizeVariationCard
              sizes={sizes}
              onRemove={onRemoveSize}
              onChangeSize={onChangeSize}
              measurementOptions={measurementOptions}
              mode="edit"
              onAddSize={onAddSize}
            />

            <ModelNumberCard
              sizes={sizes}
              colors={colors}
              getCode={getCode}
              onChangeModelNumber={handleChangeModelNumber}
            />
          </>
        )}
      </div>

      <div className="space-y-4">
        <AdminCard
          mode="edit"
          assigneeId={assigneeId}
          assigneeName={assigneeName || "未設定"}
          onSelectAssignee={onSelectAssignee}
          onEditAssignee={onEditAssignee}
          onClickAssignee={onClickAssignee}
        />

        <ProductBlueprintBrandCard
          mode="edit"
          brandId={brandId}
          brandName={brandName}
          brandOptions={brandOptions}
          brandLoading={brandLoading}
          brandError={brandError}
          onChangeBrandId={onChangeBrandId}
        />
      </div>
    </PageStyle>
  );
}