// frontend/amol/src/features/shared/presentation/components/ProductModelMeta.tsx

import ProductMetaList, {
  type ProductMetaListItem,
} from "./ProductMetaList";

import type { ProductModelDisplay } from "../utils/productModelDisplay";

export type ProductModelMetaProps = {
  conditionLabel?: string;
  model: ProductModelDisplay;
};

export default function ProductModelMeta({
  conditionLabel,
  model,
}: ProductModelMetaProps) {
  const {
    hasModelInfo,
    kindLabel,
    size,
    colorLabel,
    colorCssValue,
    measurementsLabel,
    volumeLabel,
  } = model;

  if (!conditionLabel && !hasModelInfo) {
    return null;
  }

  const items: ProductMetaListItem[] = [];

  if (conditionLabel) {
    items.push({
      label: "商品の状態",
      value: conditionLabel,
    });
  }

  if (kindLabel && kindLabel !== "アパレル") {
    items.push({
      label: "種別",
      value: kindLabel,
    });
  }

  if (size) {
    items.push({
      label: "サイズ",
      value: size,
    });
  }

  if (colorLabel || colorCssValue) {
    items.push({
      label: "カラー",
      value: (
        <span className="product-detail__color-value">
          {colorCssValue ? (
            <span
              className="product-detail__color-swatch"
              style={{ backgroundColor: colorCssValue }}
              aria-label={colorLabel || "商品カラー"}
              title={colorLabel || "商品カラー"}
            />
          ) : null}
          <span>{colorLabel || colorCssValue || "カラー未設定"}</span>
        </span>
      ),
    });
  }

  if (measurementsLabel !== "-") {
    items.push({
      label: "採寸",
      value: measurementsLabel,
    });
  }

  if (volumeLabel !== "-") {
    items.push({
      label: "容量",
      value: volumeLabel,
    });
  }

  return <ProductMetaList items={items} />;
}