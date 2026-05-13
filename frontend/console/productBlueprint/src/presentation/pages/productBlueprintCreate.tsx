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
    productBlueprintCategoryId,
    productBlueprintCategory,
    productBlueprintCategoryLabel,
    isApparelCategory,
    fit,
    material,
    weight,
    qualityAssurance,

    // 商品カテゴリから導出された採寸項目
    measurementOptions,

    // バリエーション
    colorInput,
    colors,
    colorRgbMap,
    sizes,
    modelNumbers,
    onChangeProductName,
    onChangeProductBlueprintCategory,
    onChangeFit,
    onChangeMaterial,
    onChangeWeight,
    onChangeQualityAssurance,
    onChangeColorInput,
    onAddColor,
    onRemoveColor,
    onChangeColorRgb,

    // サイズ操作
    onAddSize,
    onRemoveSize,
    onChangeSize,

    // モデルナンバー操作（アプリケーション層）
    onChangeModelNumber,

    // 管理情報
    assigneeId,
    assigneeName,
    onSelectAssignee,
    onEditAssignee,
    onClickAssignee,

    // 画面アクション
    onCreate,
    onBack,
  } = useProductBlueprintCreate();

  // -----------------------------
  // モデルナンバー表示用の hook（model 側）
  //   ※ rgb を hook 経由で渡す場合は、ここで colorRgbMap を useModelCard に渡す
  // -----------------------------
  const { getCode, onChangeModelNumber: uiOnChangeModelNumber } = useModelCard({
    sizes,
    colors,
    modelNumbers,
    colorRgbMap,
  });

  // UI 変更時に「model 側の内部状態」と「productBlueprintCreate の状態」の両方を更新
  const handleChangeModelNumber = (
    sizeLabel: string,
    color: string,
    nextCode: string,
  ) => {
    uiOnChangeModelNumber(sizeLabel, color, nextCode);
    onChangeModelNumber(sizeLabel, color, nextCode);
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
          productBlueprintCategoryId={productBlueprintCategoryId}
          productBlueprintCategory={productBlueprintCategory}
          onChangeProductBlueprintCategory={onChangeProductBlueprintCategory}
          fit={fit}
          materials={material}
          weight={weight}
          washTags={qualityAssurance}
          onChangeProductName={onChangeProductName}
          onChangeFit={onChangeFit}
          onChangeMaterials={onChangeMaterial}
          onChangeWeight={onChangeWeight}
          onChangeWashTags={onChangeQualityAssurance}
        />

        {!productBlueprintCategory && (
          <p className="mt-2 text-xs text-slate-500">
            商品カテゴリを選択すると、カテゴリに応じた入力欄が表示されます。
          </p>
        )}

        {productBlueprintCategory && !isApparelCategory && (
          <p className="mt-2 text-xs text-slate-500">
            選択中の商品カテゴリ: {productBlueprintCategoryLabel}
          </p>
        )}

        {isApparelCategory && (
          <>
            <ColorVariationCard
              colors={colors}
              colorInput={colorInput}
              onChangeColorInput={onChangeColorInput}
              onAddColor={onAddColor}
              onRemoveColor={onRemoveColor}
              colorRgbMap={colorRgbMap}
              onChangeColorRgb={onChangeColorRgb}
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
              getCode={getCode}
              onChangeModelNumber={handleChangeModelNumber}
            />
          </>
        )}
      </div>

      <AdminCard
        mode="edit"
        assigneeId={assigneeId}
        assigneeName={assigneeName || "未設定"}
        onSelectAssignee={onSelectAssignee}
        onEditAssignee={onEditAssignee}
        onClickAssignee={onClickAssignee}
      />
    </PageStyle>
  );
}