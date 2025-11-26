// frontend/console/productBlueprint/src/presentation/pages/productBlueprintDetail.tsx

import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import ProductBlueprintCard from "../components/productBlueprintCard";
import ColorVariationCard from "../../../../model/src/presentation/components/ColorVariationCard";
import SizeVariationCard from "../../../../model/src/presentation/components/SizeVariationCard";
import ModelNumberCard from "../../../../model/src/presentation/components/ModelNumberCard";
import { useProductBlueprintDetail } from "../hook/useProductBlueprintDetail";

// ItemType / 採寸オプションを productBlueprint の catalog から利用
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

    // variations
    colors,
    colorInput,
    sizes,
    modelNumbers,
    colorRgbMap,

    assignee,
    creator,
    createdAt,

    onBack,
    onSave,

    // 編集ハンドラ
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

  return (
    <PageStyle
      layout="grid-2"
      title={pageTitle}
      onBack={onBack}
      onSave={onSave}
    >
      {/* --- 左ペイン --- */}
      <div>
        {/* ▼ ProductBlueprintCard だけ View モードに変更 */}
        <ProductBlueprintCard
          mode="view"
          productName={productName}
          brand={brand}
          itemType={normalizedItemType}
          fit={fit}
          materials={materials}
          weight={weight}
          washTags={washTags}
          productIdTag={productIdTag}
        />

        {/* ColorVariationCard は従来どおり edit */}
        <ColorVariationCard
          mode="edit"
          colors={colors}
          colorInput={colorInput}
          onChangeColorInput={onChangeColorInput}
          onAddColor={onAddColor}
          onRemoveColor={onRemoveColor}
          colorRgbMap={colorRgbMap}
        />

        {/* SizeVariationCard も edit */}
        <SizeVariationCard
          mode="edit"
          sizes={sizes}
          onRemove={onRemoveSize}
          measurementOptions={measurementOptions}
        />

        {/* ModelNumberCard も edit */}
        <ModelNumberCard
          mode="edit"
          sizes={sizes}
          colors={colors}
          getCode={getCode}
        />
      </div>

      {/* --- 右ペイン（管理情報） --- */}
      <AdminCard
        title="管理情報"
        assigneeName={assignee}
        createdByName={creator}
        createdAt={createdAt}
        onEditAssignee={onEditAssignee}
        onClickAssignee={onClickAssignee}
      />
    </PageStyle>
  );
}
