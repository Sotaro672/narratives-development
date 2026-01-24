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

import type { ItemType } from "../../domain/entity/catalog";
import { ITEM_TYPE_MEASUREMENT_OPTIONS } from "../../domain/entity/catalog";

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

    itemType,
    fit,
    materials,
    weight,
    washTags,
    productIdTag,

    colors,
    colorInput,
    sizes,
    colorRgbMap,

    assignee,
    creator,
    createdAt,
    updater,
    updatedAt,

    onBack,

    onSave,
    onDelete,
    onChangeProductName,
    onChangeItemType,
    onChangeFit,
    onChangeMaterials,
    onChangeWeight,
    onChangeWashTags,
    onChangeProductIdTag,
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

  const normalizedItemType = (itemType || undefined) as ItemType | undefined;

  const measurementOptions =
    normalizedItemType != null
      ? ITEM_TYPE_MEASUREMENT_OPTIONS[normalizedItemType]
      : undefined;

  // ----------------------------------------
  // 編集モード管理
  // ----------------------------------------
  const [editMode, setEditMode] = React.useState(false);
  const noop = () => {};
  const noopStr = (_: string) => {};
  const noopColor = (_: string) => {};

  const handleSave = () => {
    if (onSave) onSave();
    setEditMode(false);
  };

  const handleDelete = () => {
    if (!onDelete) return;
    onDelete();
  };

  return (
    <PageStyle
      layout="grid-2"
      title={pageTitle}
      onBack={onBack}
      onSave={editMode ? handleSave : undefined}
      onEdit={!editMode ? () => setEditMode(true) : undefined}
      onDelete={editMode ? handleDelete : undefined}
      onCancel={editMode ? () => setEditMode(false) : undefined}
    >
      {/* --- 左ペイン --- */}
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
          itemType={normalizedItemType}
          fit={fit}
          materials={materials}
          weight={weight}
          washTags={washTags}
          productIdTag={productIdTag}
          onChangeProductName={editMode ? onChangeProductName : undefined}
          onChangeItemType={editMode ? onChangeItemType : undefined}
          onChangeFit={editMode ? onChangeFit : undefined}
          onChangeMaterials={editMode ? onChangeMaterials : undefined}
          onChangeWeight={editMode ? onChangeWeight : undefined}
          onChangeWashTags={editMode ? onChangeWashTags : undefined}
          onChangeProductIdTag={editMode ? onChangeProductIdTag : undefined}
        />

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
      </div>

      {/* --- 右ペイン：管理情報 + ログ --- */}
      <div>
        <AdminCard
          title="管理情報"
          assigneeName={assignee}
          createdByName={creator}
          createdAt={createdAt}
          // ★ 追加：最終更新者 / 最終更新日
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
