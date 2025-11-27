import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import ProductBlueprintCard from "../components/productBlueprintCard";
import ColorVariationCard from "../../../../model/src/presentation/components/ColorVariationCard";
import SizeVariationCard from "../../../../model/src/presentation/components/SizeVariationCard";
import ModelNumberCard from "../../../../model/src/presentation/components/ModelNumberCard";

import type { ItemType } from "../../domain/entity/catalog";
import { ITEM_TYPE_MEASUREMENT_OPTIONS } from "../../domain/entity/catalog";

import { useProductBlueprintDeletedDetail } from "../hook/useProductBlueprintDeletedDetail";

/**
 * 削除済み商品設計の詳細画面（閲覧専用）
 * - PageStyle レイアウトは ProductBlueprintDetail と同じ grid-2
 * - 本画面では常に view モード
 * - ヘッダーには編集ボタンを出さず、「復旧 / 物理削除」ボタンのみを表示
 */
export default function ProductBlueprintDeletedDetail() {
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
    colorRgbMap,

    assignee,
    creator,
    createdAt,

    getCode,
    onBack,
    onRestore,
    onPurge,
  } = useProductBlueprintDeletedDetail();

  const normalizedItemType = (itemType || undefined) as ItemType | undefined;

  const measurementOptions =
    normalizedItemType != null
      ? ITEM_TYPE_MEASUREMENT_OPTIONS[normalizedItemType]
      : undefined;

  // 削除済み画面では常に view モード
  const noop = () => {};

  return (
    <PageStyle
      layout="grid-2"
      title={`${pageTitle}（削除済み）`}
      onBack={onBack}
      // 削除済み専用ページなので編集系のボタンは渡さない
      onSave={undefined}
      onEdit={undefined}
      onDelete={undefined}
      onCancel={undefined}
      // 代わりに復旧 / 物理削除ボタンをヘッダーに表示
      onRestore={onRestore}
      onPurge={onPurge}
    >
      {/* --- 左ペイン --- */}
      <div>
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
          // 削除済み画面では編集不可なので onChange 系は全て undefined
          onChangeProductName={undefined}
          onChangeItemType={undefined}
          onChangeFit={undefined}
          onChangeMaterials={undefined}
          onChangeWeight={undefined}
          onChangeWashTags={undefined}
          onChangeProductIdTag={undefined}
        />

        <ColorVariationCard
          mode="view"
          colors={colors}
          colorInput={colorInput}
          colorRgbMap={colorRgbMap}
          onChangeColorInput={noop}
          onAddColor={noop}
          onRemoveColor={noop}
        />

        <SizeVariationCard
          mode="view"
          sizes={sizes}
          measurementOptions={measurementOptions}
          onAddSize={undefined}
          onRemove={noop}
          onChangeSize={undefined}
        />

        <ModelNumberCard
          mode="view"
          sizes={sizes}
          colors={colors}
          getCode={getCode}
          onChangeModelNumber={undefined}
        />
      </div>

      {/* --- 右ペイン：管理情報 --- */}
      <AdminCard
        title="管理情報（削除済み）"
        assigneeName={assignee}
        createdByName={creator}
        createdAt={createdAt}
        mode="view"
        onClickAssignee={noop}
      />
    </PageStyle>
  );
}
