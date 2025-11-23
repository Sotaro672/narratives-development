// frontend/console/productBlueprint/src/presentation/pages/productBlueprintCreate.tsx

import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import { AdminCard } from "../../../../admin/src/presentation/components/AdminCard";
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
    itemType,
    fit,
    material,
    weight,
    qualityAssurance,
    productIdTagType,

    // アイテム種別から導出された採寸項目
    measurementOptions,

    // バリエーション
    colorInput,
    colors,
    sizes,
    modelNumbers,
    onChangeProductName,
    onChangeItemType,
    onChangeFit,
    onChangeMaterial,
    onChangeWeight,
    onChangeQualityAssurance,
    onChangeProductIdTagType,
    onChangeColorInput,
    onAddColor,
    onRemoveColor,

    // サイズ操作
    onAddSize,
    onRemoveSize,
    onChangeSize,

    // モデルナンバー操作 ★追加
    onChangeModelNumber,

    // 管理情報
    assigneeId,
    onEditAssignee,
    onClickAssignee,

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
          itemType={itemType}
          fit={fit}
          materials={material}
          weight={weight}
          washTags={qualityAssurance}
          productIdTag={productIdTagType}
          onChangeProductName={onChangeProductName}
          onChangeItemType={onChangeItemType}
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
          onChangeSize={onChangeSize}   // ★ 入力を反映
          measurementOptions={measurementOptions}
          mode="edit"
          onAddSize={onAddSize}
        />

        <ModelNumberCard
          sizes={sizes}
          colors={colors}
          modelNumbers={modelNumbers}
          onChangeModelNumber={onChangeModelNumber}
        />

      </div>

      <AdminCard
        assigneeName={assigneeId || "未設定"}
        onEditAssignee={onEditAssignee}
        onClickAssignee={onClickAssignee}
      />
    </PageStyle>
  );
}
