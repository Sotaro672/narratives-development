import * as React from "react";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import ProductBlueprintCard from "../components/productBlueprintCard";
import ColorVariationCard from "../../../../model/src/presentation/components/ColorVariationCard";
import SizeVariationCard from "../../../../model/src/presentation/components/SizeVariationCard";
import ModelNumberCard from "../../../../model/src/presentation/components/ModelNumberCard";

import { useProductBlueprintCreate } from "../hook/useProductBlueprintCreate";

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
    fit,
    material,
    weight,
    qualityAssurance,
    productIdTagType,

    // バリエーション
    colorInput,
    colors,
    sizes,
    modelNumbers,
    onChangeProductName,
    onChangeFit,
    onChangeMaterial,
    onChangeWeight,
    onChangeQualityAssurance,
    onChangeProductIdTagType,
    onChangeColorInput,
    onAddColor,
    onRemoveColor,
    onRemoveSize,

    // 管理情報
    assigneeId,
    createdBy,
    createdAt,
    onEditAssignee,
    onClickAssignee,
    onClickCreatedBy,

    // 画面アクション
    onCreate,
    onBack,
  } = useProductBlueprintCreate();

  return (
    <PageStyle
      layout="grid-2"
      title="商品設計を作成"
      onBack={onBack}
      onSave={onCreate}
    >
      <div>
        <ProductBlueprintCard
          mode="edit"
          productName={productName}
          brand={brandName}
          brandId={brandId}
          brandOptions={brandOptions}
          brandLoading={brandLoading}
          brandError={brandError}
          onChangeBrandId={onChangeBrandId}
          fit={fit}
          materials={material}
          weight={weight}
          washTags={qualityAssurance}
          productIdTag={productIdTagType}
          onChangeProductName={onChangeProductName}
          onChangeFit={onChangeFit}
          onChangeMaterials={onChangeMaterial}
          onChangeWeight={onChangeWeight}
          onChangeWashTags={onChangeQualityAssurance}
          onChangeProductIdTag={(v) => onChangeProductIdTagType(v as any)}
        />

        <ColorVariationCard
          colors={colors}
          colorInput={colorInput}
          onChangeColorInput={onChangeColorInput}
          onAddColor={onAddColor}
          onRemoveColor={onRemoveColor}
        />

        <SizeVariationCard
          sizes={sizes}
          onRemove={onRemoveSize}
        />

        <ModelNumberCard
          sizes={sizes}
          colors={colors}
          modelNumbers={modelNumbers}
        />
      </div>

      <AdminCard
        title="管理情報"
        assigneeName={assigneeId || "未設定"}
        createdByName={createdBy || "未設定"}
        createdAt={createdAt || "未設定"}
        onEditAssignee={onEditAssignee}
        onClickAssignee={onClickAssignee}
        onClickCreatedBy={onClickCreatedBy}
      />
    </PageStyle>
  );
}
