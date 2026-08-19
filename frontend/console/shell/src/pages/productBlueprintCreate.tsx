// frontend/console/shell/src/pages/productBlueprintCreate.tsx

import * as React from "react";

import PageStyle from "../layout/PageStyle/PageStyle";
import { AdminCard } from "../features/admin/presentation/components/AdminCard";
import ProductBlueprintCard from "../features/productBlueprint/presentation/cards/productBlueprintForm";
import CategoryFieldsCard from "../features/productBlueprint/presentation/cards/categoryFields";
import ColorVariationCard from "../features/model/presentation/components/ColorVariationCard";
import SizeVariationCard from "../features/model/presentation/components/SizeVariationCard";
import ModelNumberCard from "../features/model/presentation/components/ModelNumberCard";
import VolumeCard from "../features/model/presentation/components/VolumeCard";
import AlcoholModelNumberCard from "../features/model/presentation/components/AlcoholModelNumberCard";
import ShippingPackageCard from "../features/model/presentation/components/ShippingPackageCard";
import { useProductBlueprintCreate } from "../features/productBlueprint/presentation/hooks/create/useProductBlueprintCreate";
import {
  toProductBlueprintCategoryPathKey,
} from "../features/productBlueprint/domain/productBlueprintCategory";

function shouldShowApparelVariationCards(categoryCode: string): boolean {
  return (
    categoryCode === "apparel.tops" ||
    categoryCode === "apparel.bottoms" ||
    categoryCode === "apparel.dress" ||
    categoryCode === "apparel.outerwear" ||
    categoryCode === "apparel.shoes"
  );
}

function shouldShowAlcoholVariationCards(categoryCode: string): boolean {
  return (
    categoryCode === "alcohol.beer" ||
    categoryCode === "alcohol.sake" ||
    categoryCode === "alcohol.shochu" ||
    categoryCode === "alcohol.spirits" ||
    categoryCode === "alcohol.whisky" ||
    categoryCode === "alcohol.wine"
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
    productBlueprintCategoryPath,
    productBlueprintCategoryLabel,
    productBlueprintCategoryOptions,
    productBlueprintCategoryLoading,
    productBlueprintCategoryError,
    isApparelCategory,
    isAlcoholCategory,
    categoryFields,

    // 商品カテゴリから導出された採寸項目
    measurementOptions,

    // apparelバリエーション
    colorInput,
    colors,
    colorRgbMap,
    sizes,
    modelNumbers,

    // alcoholバリエーション
    volumes,
    alcoholModelNumbers,

    onChangeProductName,
    onChangeProductBlueprintCategoryPath,
    onChangeCategoryField,
    onChangeColorInput,
    onAddColor,
    onRemoveColor,
    onChangeColorRgb,

    // サイズ操作
    onAddSize,
    onRemoveSize,
    onChangeSize,

    // apparelモデルナンバー操作
    getCode,
    onChangeModelNumber,
    onChangeApparelShippingPackage,

    // alcohol容量操作
    onAddVolume,
    onRemoveVolume,
    onChangeVolume,

    // alcoholモデルナンバー操作
    onChangeAlcoholModelNumber,
    onChangeAlcoholShippingPackage,

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

  const hasProductBlueprintCategory = Boolean(
    productBlueprintCategoryPath &&
      productBlueprintCategoryPath.length > 0,
  );

  const categoryCode = productBlueprintCategoryPath
    ? toProductBlueprintCategoryPathKey(
        productBlueprintCategoryPath,
      )
    : "";

  const showApparelVariationCards = React.useMemo(
    () =>
      isApparelCategory &&
      shouldShowApparelVariationCards(categoryCode),
    [isApparelCategory, categoryCode],
  );

  const showAlcoholVariationCards = React.useMemo(
    () =>
      isAlcoholCategory &&
      shouldShowAlcoholVariationCards(categoryCode),
    [isAlcoholCategory, categoryCode],
  );

  const showCategoryOnlyMessage =
    hasProductBlueprintCategory &&
    !showApparelVariationCards &&
    !showAlcoholVariationCards;

  return (
    <PageStyle
      layout="grid-2"
      title="商品設計を作成"
      onBack={onBack}
      onSave={onCreate}
    >
      <div className="space-y-4">
        <ProductBlueprintCard
          mode="edit"
          productName={productName}
          brandId={brandId}
          brandName={brandName}
          brandOptions={brandOptions}
          brandLoading={brandLoading}
          brandError={brandError}
          onChangeBrandId={onChangeBrandId}
          productBlueprintCategoryPath={
            productBlueprintCategoryPath
          }
          productBlueprintCategoryOptions={
            productBlueprintCategoryOptions
          }
          productBlueprintCategoryLoading={
            productBlueprintCategoryLoading
          }
          productBlueprintCategoryError={
            productBlueprintCategoryError
          }
          onChangeProductBlueprintCategoryPath={
            onChangeProductBlueprintCategoryPath
          }
          onChangeProductName={onChangeProductName}
        />

        {!hasProductBlueprintCategory && (
          <p className="mt-2 text-xs text-slate-500">
            商品カテゴリを選択すると、カテゴリに応じた入力欄が表示されます。
          </p>
        )}

        {productBlueprintCategoryPath && (
          <CategoryFieldsCard
            productBlueprintCategoryPath={
              productBlueprintCategoryPath
            }
            categoryFields={categoryFields}
            mode="edit"
            onChangeCategoryField={
              onChangeCategoryField
            }
          />
        )}

        {showCategoryOnlyMessage && (
          <p className="mt-2 text-xs text-slate-500">
            選択中の商品カテゴリ:{" "}
            {productBlueprintCategoryLabel}
          </p>
        )}

        {showApparelVariationCards && (
          <>
            <ColorVariationCard
              colors={colors}
              colorInput={colorInput}
              onChangeColorInput={
                onChangeColorInput
              }
              onAddColor={onAddColor}
              onRemoveColor={onRemoveColor}
              colorRgbMap={colorRgbMap}
              onChangeColorRgb={
                onChangeColorRgb
              }
            />

            <SizeVariationCard
              sizes={sizes}
              onRemove={onRemoveSize}
              onChangeSize={onChangeSize}
              measurementOptions={
                measurementOptions
              }
              mode="edit"
              onAddSize={onAddSize}
            />

            <ModelNumberCard
              sizes={sizes}
              colors={colors}
              getCode={getCode}
              onChangeModelNumber={
                onChangeModelNumber
              }
            />

            <ShippingPackageCard
              kind="apparel"
              modelNumbers={modelNumbers}
              mode="edit"
              onChangeShippingPackage={
                onChangeApparelShippingPackage
              }
            />
          </>
        )}

        {showAlcoholVariationCards && (
          <>
            <VolumeCard
              volumes={volumes}
              mode="edit"
              onAddVolume={onAddVolume}
              onRemoveVolume={onRemoveVolume}
              onChangeVolume={onChangeVolume}
            />

            <AlcoholModelNumberCard
              volumes={volumes}
              modelNumbers={
                alcoholModelNumbers
              }
              mode="edit"
              onChangeModelNumber={
                onChangeAlcoholModelNumber
              }
            />

            <ShippingPackageCard
              kind="alcohol"
              modelNumbers={
                alcoholModelNumbers
              }
              mode="edit"
              onChangeShippingPackage={
                onChangeAlcoholShippingPackage
              }
            />
          </>
        )}
      </div>

      <div className="space-y-4">
        <AdminCard
          mode="edit"
          assigneeId={assigneeId}
          assigneeName={
            assigneeName || "未設定"
          }
          onSelectAssignee={
            onSelectAssignee
          }
          onEditAssignee={
            onEditAssignee
          }
          onClickAssignee={
            onClickAssignee
          }
        />
      </div>
    </PageStyle>
  );
}