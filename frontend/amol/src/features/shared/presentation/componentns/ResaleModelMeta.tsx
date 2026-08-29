// frontend/amol/src/features/shared/presentation/componentns/ResaleModelMeta.tsx

import type {
  ResaleModelDisplay,
} from "../utils/resaleModelDisplay";

export type ResaleModelMetaProps = {
  conditionLabel?: string;
  model: ResaleModelDisplay;
};

export default function ResaleModelMeta({
  conditionLabel,
  model,
}: ResaleModelMetaProps) {
  const {
    hasModelInfo,
    kindLabel,
    modelNumber,
    size,
    colorLabel,
    colorCssValue,
    measurementsLabel,
    volumeLabel,
  } = model;

  if (!conditionLabel && !hasModelInfo) {
    return null;
  }

  return (
    <dl className="resale-product-detail__meta">
      {conditionLabel ? (
        <div className="resale-product-detail__meta-row">
          <dt>商品の状態</dt>
          <dd>{conditionLabel}</dd>
        </div>
      ) : null}

      {kindLabel ? (
        <div className="resale-product-detail__meta-row">
          <dt>種別</dt>
          <dd>{kindLabel}</dd>
        </div>
      ) : null}

      {modelNumber ? (
        <div className="resale-product-detail__meta-row">
          <dt>モデル番号</dt>
          <dd>{modelNumber}</dd>
        </div>
      ) : null}

      {size ? (
        <div className="resale-product-detail__meta-row">
          <dt>サイズ</dt>
          <dd>{size}</dd>
        </div>
      ) : null}

      {colorLabel || colorCssValue ? (
        <div className="resale-product-detail__meta-row">
          <dt>カラー</dt>
          <dd>
            <span className="resale-product-detail__color-value">
              {colorCssValue ? (
                <span
                  className="resale-product-detail__color-swatch"
                  style={{ backgroundColor: colorCssValue }}
                  aria-label={colorLabel || "商品カラー"}
                  title={colorLabel || "商品カラー"}
                />
              ) : null}

              <span>
                {colorLabel || colorCssValue || "カラー未設定"}
              </span>
            </span>
          </dd>
        </div>
      ) : null}

      {measurementsLabel !== "-" ? (
        <div className="resale-product-detail__meta-row">
          <dt>採寸</dt>
          <dd>{measurementsLabel}</dd>
        </div>
      ) : null}

      {volumeLabel !== "-" ? (
        <div className="resale-product-detail__meta-row">
          <dt>容量</dt>
          <dd>{volumeLabel}</dd>
        </div>
      ) : null}
    </dl>
  );
}