// frontend/console/productBlueprint/src/presentation/pages/productBlueprintDetail.tsx
import * as React from "react";

import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import ProductBlueprintCard from "../components/productBlueprintCard";
import ColorVariationCard from "../../../../model/src/presentation/components/ColorVariationCard";
import SizeVariationCard from "../../../../model/src/presentation/components/SizeVariationCard";
import ModelNumberCard from "../../../../model/src/presentation/components/ModelNumberCard";
import LogCard from "../../../../log/src/presentation/LogCard";

import { useProductBlueprintDetail } from "../hook/useProductBlueprintDetail";

import {
  APPAREL_CATEGORY_MEASUREMENT_OPTIONS,
  isApparelCategoryCode,
} from "../../domain/entity/apparel";

function shouldShowModelVariationCards(categoryCode: string): boolean {
  return (
    categoryCode === "apparel.tops" ||
    categoryCode === "apparel.bottoms" ||
    categoryCode === "apparel.dress" ||
    categoryCode === "apparel.outerwear" ||
    categoryCode === "apparel.shoes"
  );
}

export default function ProductBlueprintDetail() {
  const {
    pageTitle,
    productName,
    brand,
    brandId,
    brandOptions,
    brandLoading,
    brandError,
    onChangeBrandId,

    productBlueprintCategoryId,
    productBlueprintCategory,
    productBlueprintCategoryLabel,
    isApparelCategory,

    fit,
    materials,
    weight,
    washTags,

    colors,
    colorInput,
    sizes,
    colorRgbMap,

    assignee,
    creator,
    createdAt,
    updater,
    updatedAt,

    printed,

    onBack,

    onSave,
    onDelete,
    onChangeProductName,
    onChangeProductBlueprintCategory,
    onChangeFit,
    onChangeMaterials,
    onChangeWeight,
    onChangeWashTags,
    onChangeColorInput,
    onAddColor,
    onRemoveColor,
    onChangeColorRgb,

    onAddSize,
    onRemoveSize,
    onChangeSize,

    onChangeModelNumber,

    onClickAssignee,

    getCode,
  } = useProductBlueprintDetail();

  const categoryCode = String(productBlueprintCategory?.code ?? "").trim();

  const measurementOptions = isApparelCategoryCode(categoryCode)
    ? APPAREL_CATEGORY_MEASUREMENT_OPTIONS[categoryCode]
    : undefined;

  const showModelVariationCards = React.useMemo(
    () => isApparelCategory && shouldShowModelVariationCards(categoryCode),
    [isApparelCategory, categoryCode],
  );

  const [editMode, setEditMode] = React.useState(false);

  const noop = React.useCallback(() => {}, []);
  const noopStr = React.useCallback((_: string) => {}, []);
  const noopColor = React.useCallback((_: string) => {}, []);

  const handleSave = React.useCallback(() => {
    onSave();
    setEditMode(false);
  }, [onSave]);

  const handleDelete = React.useCallback(() => {
    onDelete();
  }, [onDelete]);

  React.useEffect(() => {
    if (printed && editMode) {
      setEditMode(false);
    }
  }, [printed, editMode]);

  const canEdit = !printed;

  return (
    <PageStyle
      layout="grid-2"
      title={pageTitle}
      onBack={onBack}
      onSave={editMode ? handleSave : undefined}
      onEdit={!editMode && canEdit ? () => setEditMode(true) : undefined}
      onDelete={editMode ? handleDelete : undefined}
      onCancel={editMode ? () => setEditMode(false) : undefined}
    >
      <div>
        <ProductBlueprintCard
          mode={editMode ? "edit" : "view"}
          productName={productName}
          brand={brand}
          brandId={brandId}
          brandOptions={brandOptions}
          brandLoading={brandLoading}
          brandError={brandError}
          onChangeBrandId={editMode ? onChangeBrandId : undefined}
          productBlueprintCategoryId={productBlueprintCategoryId}
          productBlueprintCategory={productBlueprintCategory}
          onChangeProductBlueprintCategory={
            editMode ? onChangeProductBlueprintCategory : undefined
          }
          fit={fit}
          materials={materials}
          weight={weight}
          washTags={washTags}
          onChangeProductName={editMode ? onChangeProductName : undefined}
          onChangeFit={editMode ? onChangeFit : undefined}
          onChangeMaterials={editMode ? onChangeMaterials : undefined}
          onChangeWeight={editMode ? onChangeWeight : undefined}
          onChangeWashTags={editMode ? onChangeWashTags : undefined}
        />

        {!productBlueprintCategory && (
          <p className="mt-2 text-xs text-slate-500">
            商品カテゴリが未設定です。
          </p>
        )}

        {productBlueprintCategory && !showModelVariationCards && (
          <p className="mt-2 text-xs text-slate-500">
            選択中の商品カテゴリ: {productBlueprintCategoryLabel}
          </p>
        )}

        {showModelVariationCards && (
          <>
            <ColorVariationCard
              mode={editMode ? "edit" : "view"}
              colors={colors}
              colorInput={colorInput}
              colorRgbMap={colorRgbMap}
              onChangeColorInput={editMode ? onChangeColorInput : noopStr}
              onAddColor={editMode ? onAddColor : noop}
              onRemoveColor={editMode ? onRemoveColor : noopColor}
              onChangeColorRgb={editMode ? onChangeColorRgb : undefined}
            />

            <SizeVariationCard
              mode={editMode ? "edit" : "view"}
              sizes={sizes}
              measurementOptions={measurementOptions}
              onAddSize={editMode ? onAddSize : undefined}
              onRemove={editMode ? onRemoveSize : noop}
              onChangeSize={editMode ? onChangeSize : undefined}
            />

            <ModelNumberCard
              mode={editMode ? "edit" : "view"}
              sizes={sizes}
              colors={colors}
              getCode={getCode}
              onChangeModelNumber={editMode ? onChangeModelNumber : undefined}
            />
          </>
        )}
      </div>

      <div>
        <AdminCard
          title="管理情報"
          assigneeName={assignee}
          createdByName={creator}
          createdAt={createdAt}
          updatedByName={updater}
          updatedAt={updatedAt}
          mode={editMode ? "edit" : "view"}
          onClickAssignee={editMode ? onClickAssignee : noop}
        />

        <div className="section-gap">
          <LogCard />
        </div>
      </div>
    </PageStyle>
  );
}