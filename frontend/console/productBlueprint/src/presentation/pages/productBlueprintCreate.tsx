// frontend/console/productBlueprint/src/presentation/pages/productBlueprintCreate.tsx 

import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import { AdminCard } from "../../../../admin/src/presentation/components/AdminCard";
import ProductBlueprintCard from "../components/productBlueprintCard";
import ColorVariationCard from "../../../../model/src/presentation/components/ColorVariationCard";
import SizeVariationCard from "../../../../model/src/presentation/components/SizeVariationCard";
import ModelNumberCard from "../../../../model/src/presentation/components/ModelNumberCard";

// ★ モデルナンバー用のロジックは model 側の hook を利用
import { useModelCard } from "../../../../model/src/presentation/hook/useModelCard";

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

    // モデルナンバー操作（アプリケーション層）
    onChangeModelNumber,

    // 管理情報
    assigneeId,
    onEditAssignee,
    onClickAssignee,

    // 画面アクション
    onCreate,
    onBack,
  } = useProductBlueprintCreate();

  // -----------------------------
  // モデルナンバー表示用の hook（model 側）
  // -----------------------------
  const { getCode, onChangeModelNumber: uiOnChangeModelNumber } = useModelCard({
    sizes,
    colors,
    modelNumbers,
  });

  // UI 変更時に「model 側の内部状態」と「productBlueprintCreate の状態」の両方を更新
  const handleChangeModelNumber = (
    sizeLabel: string,
    color: string,
    nextCode: string,
  ) => {
    uiOnChangeModelNumber(sizeLabel, color, nextCode); // UI 側の codeMap を更新
    onChangeModelNumber(sizeLabel, color, nextCode);   // 画面のアプリケーション状態を更新
  };

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
          onChangeSize={onChangeSize}
          measurementOptions={measurementOptions}
          mode="edit"
          onAddSize={onAddSize}
        />

        <ModelNumberCard
          sizes={sizes}
          colors={colors}
          // ★ model 側 hook から取得した getter / handler を渡す
          getCode={getCode}
          onChangeModelNumber={handleChangeModelNumber}
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
