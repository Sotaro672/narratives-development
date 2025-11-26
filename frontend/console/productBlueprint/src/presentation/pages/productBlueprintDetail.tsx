// frontend/console/productBlueprint/src/presentation/pages/productBlueprintDetail.tsx

import * as React from "react";

import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import ProductBlueprintCard from "../components/productBlueprintCard";
import ColorVariationCard from "../../../../model/src/presentation/components/ColorVariationCard";
import SizeVariationCard from "../../../../model/src/presentation/components/SizeVariationCard";
import ModelNumberCard from "../../../../model/src/presentation/components/ModelNumberCard";

import { useProductBlueprintDetail } from "../hook/useProductBlueprintDetail";

import type { ItemType } from "../../domain/entity/catalog";
import { ITEM_TYPE_MEASUREMENT_OPTIONS } from "../../domain/entity/catalog";

export default function ProductBlueprintDetail() {
  const {
    pageTitle,
    productName,
    brand,
    itemType,
    fit,
    materials,
    weight,
    washTags,
    productIdTag,

    colors,
    colorInput,
    sizes,
    modelNumbers,
    colorRgbMap,

    assignee,
    creator,
    createdAt,

    onBack,

    // 編集用
    onSave,
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
    onRemoveSize,
    onEditAssignee,
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

  // 保存押下時
  const handleSave = () => {
    if (onSave) onSave();
    setEditMode(false);
  };

  // 削除押下時（未実装なので今は空）
  const handleDelete = () => {
    console.log("DELETE clicked");
  };

  return (
    <PageStyle
      layout="grid-2"
      title={pageTitle}
      onBack={onBack}
      onSave={editMode ? handleSave : undefined}
      onEdit={!editMode ? () => setEditMode(true) : undefined}
      onDelete={editMode ? handleDelete : undefined}
      // ★ 追加：編集中のみキャンセルボタンを表示
      onCancel={editMode ? () => setEditMode(false) : undefined}
    >
      {/* --- 左ペイン --- */}
      <div>
        <ProductBlueprintCard
          mode={editMode ? "edit" : "view"}
          productName={productName}
          brand={brand}
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
          onChangeColorInput={editMode ? onChangeColorInput : noop}
          onAddColor={editMode ? onAddColor : noop}
          onRemoveColor={editMode ? onRemoveColor : noop}
        />

        <SizeVariationCard
          mode={editMode ? "edit" : "view"}
          sizes={sizes}
          measurementOptions={measurementOptions}
          onRemove={editMode ? onRemoveSize : noop}
        />

        <ModelNumberCard
          mode={editMode ? "edit" : "view"}
          sizes={sizes}
          colors={colors}
          getCode={getCode}
        />
      </div>

      {/* --- 右ペイン：管理情報 --- */}
      <AdminCard
        title="管理情報"
        assigneeName={assignee}
        createdByName={creator}
        createdAt={createdAt}
        onEditAssignee={editMode ? onEditAssignee : undefined}
        onClickAssignee={editMode ? onClickAssignee : noop}
      />
    </PageStyle>
  );
}
