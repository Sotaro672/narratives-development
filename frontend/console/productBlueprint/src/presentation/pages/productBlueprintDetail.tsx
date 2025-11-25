// frontend/productBlueprint/src/presentation/pages/productBlueprintDetail.tsx

import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import ProductBlueprintCard from "../components/productBlueprintCard";
import ColorVariationCard from "../../../../model/src/presentation/components/ColorVariationCard";
import SizeVariationCard from "../../../../model/src/presentation/components/SizeVariationCard";
import ModelNumberCard from "../../../../model/src/presentation/components/ModelNumberCard";
import { useProductBlueprintDetail } from "../hook/useProductBlueprintDetail";

export default function ProductBlueprintDetail() {
  const {
    pageTitle,
    productName,
    brand,
    itemType,          // ★ ItemType | ""
    fit,
    materials,
    weight,
    washTags,
    productIdTag,
    colors,
    colorInput,
    sizes,
    modelNumbers,
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

    // ★ ＋ ModelNumberCard 用
    getCode,
  } = useProductBlueprintDetail();

  // ------------------------------------
  // ★ 現在取得しているデータ値のログ
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
    assignee,
    creator,
    createdAt,
  });

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
          itemType={itemType || undefined}
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

        <ColorVariationCard
          mode="edit"
          colors={colors}
          colorInput={colorInput}
          onChangeColorInput={onChangeColorInput}
          onAddColor={onAddColor}
          onRemoveColor={onRemoveColor}
        />

        <SizeVariationCard
          mode="edit"
          sizes={sizes}
          onRemove={onRemoveSize}
        />

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
