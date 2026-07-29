// frontend/console/shell/src/pages/productBlueprintCreate.tsx

import * as React from "react";

import PageStyle from "../layout/PageStyle/PageStyle";

import { AdminCard } from "../features/admin/presentation/components/AdminCard";

import ProductBlueprintCard from "../features/productBlueprint/presentation/cards/productBlueprintForm";

import {
  ProductBlueprintBrandCard,
  ProductBlueprintCategoryCard,
} from "../features/productBlueprint/presentation/cards/classification";

import CategoryFieldsCard from "../features/productBlueprint/presentation/cards/categoryFields";

import ColorVariationCard from "../features/model/presentation/components/ColorVariationCard";
import SizeVariationCard from "../features/model/presentation/components/SizeVariationCard";
import ModelNumberCard from "../features/model/presentation/components/ModelNumberCard";
import VolumeCard from "../features/model/presentation/components/VolumeCard";
import AlcoholModelNumberCard from "../features/model/presentation/components/AlcoholModelNumberCard";

import { useProductBlueprintCreate } from "../features/productBlueprint/presentation/hooks/create/useProductBlueprintCreate";

function shouldShowApparelVariationCards(
  categoryCode: string,
): boolean {
  return (
    categoryCode === "apparel.tops" ||
    categoryCode === "apparel.bottoms" ||
    categoryCode === "apparel.dress" ||
    categoryCode === "apparel.outerwear" ||
    categoryCode === "apparel.shoes"
  );
}

function shouldShowAlcoholVariationCards(
  categoryCode: string,
): boolean {
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
    productBlueprintCategoryId,
    productBlueprintCategory,
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

    // alcoholバリエーション
    volumes,
    alcoholModelNumbers,

    onChangeProductName,
    onChangeProductBlueprintCategory,
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

    // alcohol容量操作
    onAddVolume,
    onRemoveVolume,
    onChangeVolume,

    // alcoholモデルナンバー操作
    onChangeAlcoholModelNumber,

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

  const categoryCode = String(
    productBlueprintCategory?.code ?? "",
  ).trim();

  const showApparelVariationCards =
    React.useMemo(
      () =>
        isApparelCategory &&
        shouldShowApparelVariationCards(
          categoryCode,
        ),
      [
        isApparelCategory,
        categoryCode,
      ],
    );

  const showAlcoholVariationCards =
    React.useMemo(
      () =>
        isAlcoholCategory &&
        shouldShowAlcoholVariationCards(
          categoryCode,
        ),
      [
        isAlcoholCategory,
        categoryCode,
      ],
    );

  const showCategoryOnlyMessage =
    Boolean(productBlueprintCategory) &&
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
        <ProductBlueprintCategoryCard
          mode="edit"
          productBlueprintCategoryId={
            productBlueprintCategoryId
          }
          productBlueprintCategory={
            productBlueprintCategory
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
          onChangeProductBlueprintCategory={
            onChangeProductBlueprintCategory
          }
        />

        <ProductBlueprintCard
          mode="edit"
          productName={productName}
          productBlueprintCategory={
            productBlueprintCategory
          }
          onChangeProductName={
            onChangeProductName
          }
        />

        {!productBlueprintCategory && (
          <p className="mt-2 text-xs text-slate-500">
            商品カテゴリを選択すると、カテゴリに応じた入力欄が表示されます。
          </p>
        )}

        {productBlueprintCategory && (
          <CategoryFieldsCard
            categoryCode={categoryCode}
            categoryFields={
              categoryFields
            }
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
              onRemoveColor={
                onRemoveColor
              }
              colorRgbMap={colorRgbMap}
              onChangeColorRgb={
                onChangeColorRgb
              }
            />

            <SizeVariationCard
              sizes={sizes}
              onRemove={onRemoveSize}
              onChangeSize={
                onChangeSize
              }
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
          </>
        )}

        {showAlcoholVariationCards && (
          <>
            <VolumeCard
              volumes={volumes}
              mode="edit"
              onAddVolume={
                onAddVolume
              }
              onRemoveVolume={
                onRemoveVolume
              }
              onChangeVolume={
                onChangeVolume
              }
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

        <ProductBlueprintBrandCard
          mode="edit"
          brandId={brandId}
          brandName={brandName}
          brandOptions={brandOptions}
          brandLoading={brandLoading}
          brandError={brandError}
          onChangeBrandId={
            onChangeBrandId
          }
        />
      </div>
    </PageStyle>
  );
}