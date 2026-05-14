// frontend/console/productBlueprint/src/presentation/pages/productBlueprintDetail.tsx

import * as React from "react";

import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import ProductBlueprintCard from "../cards/productBlueprintForm";
import ProductBlueprintClassificationCard from "../cards/classification/ProductBlueprintClassificationCard";
import CategoryFieldsCard from "../cards/categoryFields";
import ColorVariationCard from "../../../../model/src/presentation/components/ColorVariationCard";
import SizeVariationCard from "../../../../model/src/presentation/components/SizeVariationCard";
import ModelNumberCard from "../../../../model/src/presentation/components/ModelNumberCard";
import LogCard from "../../../../log/src/presentation/LogCard";

import { useProductBlueprintDetail } from "../hooks/detail/useProductBlueprintDetail";

import {
  APPAREL_CATEGORY_MEASUREMENT_OPTIONS,
  isApparelCategoryCode,
  type Fit,
} from "../../domain/entity/apparel";

import type {
  CategoryFieldValue,
  CategoryFieldValues,
} from "../../domain/entity/productBlueprintCategory";

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

function toNullableString(value: CategoryFieldValue): string {
  return typeof value === "string" ? value : "";
}

function toNumber(value: CategoryFieldValue): number {
  return typeof value === "number" && !Number.isNaN(value) ? value : 0;
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

  const mergedCategoryFields = React.useMemo<CategoryFieldValues>(() => {
    return {
      fit,
      material: String(materials ?? ""),
      weight: toSafeNumber(weight, 0),
      washTags: toSafeStringArray(washTags),
    };
  }, [fit, materials, weight, washTags]);

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

  const handleChangeCategoryField = React.useCallback(
    (key: string, value: CategoryFieldValue) => {
      if (!editMode) return;

      if (key === "fit") {
        onChangeFit(value as Fit);
        return;
      }

      if (key === "material") {
        onChangeMaterials(toNullableString(value));
        return;
      }

      if (key === "weight") {
        onChangeWeight(toNumber(value));
        return;
      }

      if (key === "washTags" || key === "qualityAssurance") {
        onChangeWashTags(toSafeStringArray(value));
      }
    },
    [
      editMode,
      onChangeFit,
      onChangeMaterials,
      onChangeWeight,
      onChangeWashTags,
    ],
  );

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
          brandName={brand}
          productBlueprintCategory={productBlueprintCategory}
          onChangeProductName={editMode ? onChangeProductName : undefined}
        />

        {!productBlueprintCategory && (
          <p className="mt-2 text-xs text-slate-500">
            商品カテゴリが未設定です。
          </p>
        )}

        {productBlueprintCategory && (
          <CategoryFieldsCard
            categoryCode={categoryCode}
            categoryFields={mergedCategoryFields}
            mode={editMode ? "edit" : "view"}
            onChangeCategoryField={
              editMode ? handleChangeCategoryField : undefined
            }
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

      <div className="space-y-4">
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

        {editMode && (
          <ProductBlueprintClassificationCard
            mode="edit"
            brandId={brandId}
            brandName={brand}
            brandOptions={brandOptions}
            brandLoading={brandLoading}
            brandError={brandError}
            onChangeBrandId={onChangeBrandId}
            productBlueprintCategoryId={productBlueprintCategoryId}
            productBlueprintCategory={productBlueprintCategory}
            onChangeProductBlueprintCategory={onChangeProductBlueprintCategory}
          />
        )}

        <LogCard />
      </div>
    </PageStyle>
  );
}