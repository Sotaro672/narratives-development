// frontend/productBlueprint/src/presentation/pages/productBlueprintDetail.tsx

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
    itemType, // ItemType | ""
    fit,
    materials,
    weight,
    washTags,
    productIdTag,

    // ★ backend/model_handler.go → listModelVariationsByProductBlueprintId の結果
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

    // ★ ModelNumberCard用
    getCode,
  } = useProductBlueprintDetail();

  // ------------------------------------
  // デバッグログ
  // ------------------------------------
  console.log("[ProductBlueprintDetail] Current values:", {
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
  });

  // models（color / size / modelNumber）の専用ログ
  console.log("[ProductBlueprintDetail] models debug:", {
    colors,
    sizes,
    modelNumbers,
    colorRgbMap,
  });

  // itemType を ItemType | undefined に正規化
  const normalizedItemType = (itemType || undefined) as ItemType | undefined;

  // アイテム種別に応じた採寸オプション
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
        <ProductBlueprintCard
          mode="edit"
          productName={productName}
          brand={brand}
          itemType={normalizedItemType}
          fit={fit}
          materials={materials}
          weight={weight}
          washTags={washTags}
          productIdTag={productIdTag}
          onChangeProductName={onChangeProductName}
          onChangeItemType={onChangeItemType}
          onChangeFit={onChangeFit}
          onChangeMaterials={onChangeMaterials}
          onChangeWeight={onChangeWeight}
          onChangeWashTags={onChangeWashTags}
          onChangeProductIdTag={onChangeProductIdTag}
        />

        {/* ★ color variations from backend */}
        <ColorVariationCard
          mode="edit"
          colors={colors}
          colorInput={colorInput}
          onChangeColorInput={onChangeColorInput}
          onAddColor={onAddColor}
          onRemoveColor={onRemoveColor}
          // Firestore から復元した RGB(HEX) をテーブルに反映
          colorRgbMap={colorRgbMap}
        />

        {/* ★ size variations from backend */}
        <SizeVariationCard
          mode="edit"
          sizes={sizes}
          onRemove={onRemoveSize}
          measurementOptions={measurementOptions}
        />

        {/* ★ モデルナンバー（size × color × code） */}
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
