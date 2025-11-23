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
    // left pane
    productName,
    brand,
    fit,
    materials,
    weight,
    washTags,
    productIdTag,
    colors,
    colorInput,
    sizes,
    modelNumbers,
    // right pane (admin)
    assignee,
    creator,
    createdAt,
    // handlers
    onBack,
    onSave,
    onChangeProductName,
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
    onClickCreatedBy,
  } = useProductBlueprintDetail();

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
          fit={fit}
          materials={materials}
          weight={weight}
          washTags={washTags}
          productIdTag={productIdTag}
          onChangeProductName={onChangeProductName}
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
          modelNumbers={modelNumbers}
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
        onClickCreatedBy={onClickCreatedBy}
      />
    </PageStyle>
  );
}
