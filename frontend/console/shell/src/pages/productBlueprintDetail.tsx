// frontend/console/shell/src/pages/productBlueprintDetail.tsx

import * as React from "react";

import PageStyle from "../layout/PageStyle/PageStyle";
import AdminCard from "../features/admin/presentation/components/AdminCard";
import ProductBlueprintCard from "../features/productBlueprint/presentation/cards/productBlueprintForm";
import CategoryFieldsCard from "../features/productBlueprint/presentation/cards/categoryFields";
import ColorVariationCard from "../features/model/presentation/components/ColorVariationCard";
import SizeVariationCard from "../features/model/presentation/components/SizeVariationCard";
import ModelNumberCard from "../features/model/presentation/components/ModelNumberCard";
import VolumeCard from "../features/model/presentation/components/VolumeCard";
import AlcoholModelNumberCard from "../features/model/presentation/components/AlcoholModelNumberCard";
import LogCard from "../features/log/presentation/LogCard";
import { useProductBlueprintDetail } from "../features/productBlueprint/presentation/hooks/detail/useProductBlueprintDetail";
import {
  APPAREL_CATEGORY_MEASUREMENT_OPTIONS,
  isApparelCategoryCode,
} from "../shared/types/apparel";

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

export default function ProductBlueprintDetail() {
  const {
    pageTitle,
    productName,
    brand,

    productBlueprintCategory,
    productBlueprintCategoryLabel,
    isApparelCategory,
    isAlcoholCategory,

    categoryFields,
    onChangeCategoryField,

    // apparel variations
    colors,
    colorInput,
    sizes,
    colorRgbMap,

    // alcohol variations
    volumes,
    alcoholModelNumbers,

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
    onChangeColorInput,
    onAddColor,
    onRemoveColor,
    onChangeColorRgb,
    onAddSize,
    onRemoveSize,
    onChangeSize,
    onChangeModelNumber,
    onAddVolume,
    onRemoveVolume,
    onChangeVolume,
    onChangeAlcoholModelNumber,
    onClickAssignee,
    getCode,
  } = useProductBlueprintDetail();

  const categoryCode = String(productBlueprintCategory?.code ?? "").trim();

  const measurementOptions = isApparelCategoryCode(categoryCode)
    ? APPAREL_CATEGORY_MEASUREMENT_OPTIONS[categoryCode]
    : undefined;

  const showApparelVariationCards = React.useMemo(
    () => isApparelCategory && shouldShowApparelVariationCards(categoryCode),
    [isApparelCategory, categoryCode],
  );

  const showAlcoholVariationCards = React.useMemo(
    () => isAlcoholCategory && shouldShowAlcoholVariationCards(categoryCode),
    [isAlcoholCategory, categoryCode],
  );

  const showCategoryOnlyMessage =
    Boolean(productBlueprintCategory) &&
    !showApparelVariationCards &&
    !showAlcoholVariationCards;

  const [editMode, setEditMode] = React.useState(false);

  const noop = React.useCallback(() => {}, []);
  const noopStr = React.useCallback((_value: string) => {}, []);
  const noopColor = React.useCallback((_value: string) => {}, []);

  const noopVolumePatch = React.useCallback(
    (_id: string, _patch: Parameters<typeof onChangeVolume>[1]) => {},
    [],
  );

  const noopAlcoholModelNumber = React.useCallback(
    (_volumeLabel: string, _nextCode: string) => {},
    [],
  );

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
      <div className="space-y-4">
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
            categoryFields={categoryFields}
            mode={editMode ? "edit" : "view"}
            onChangeCategoryField={
              editMode ? onChangeCategoryField : undefined
            }
          />
        )}

        {showCategoryOnlyMessage && (
          <p className="mt-2 text-xs text-slate-500">
            選択中の商品カテゴリ: {productBlueprintCategoryLabel}
          </p>
        )}

        {showApparelVariationCards && (
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
              onChangeModelNumber={
                editMode ? onChangeModelNumber : undefined
              }
            />
          </>
        )}

        {showAlcoholVariationCards && (
          <>
            <VolumeCard
              mode={editMode ? "edit" : "view"}
              volumes={volumes}
              onAddVolume={editMode ? onAddVolume : undefined}
              onRemoveVolume={editMode ? onRemoveVolume : undefined}
              onChangeVolume={editMode ? onChangeVolume : noopVolumePatch}
            />

            <AlcoholModelNumberCard
              mode={editMode ? "edit" : "view"}
              volumes={volumes}
              modelNumbers={alcoholModelNumbers}
              onChangeModelNumber={
                editMode
                  ? onChangeAlcoholModelNumber
                  : noopAlcoholModelNumber
              }
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

        <LogCard />
      </div>
    </PageStyle>
  );
}