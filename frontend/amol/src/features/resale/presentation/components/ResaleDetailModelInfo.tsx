// frontend/amol/src/features/resale/presentation/components/ResaleDetailModelInfo.tsx

import type {
  ResaleDetailModelInfoProps,
} from "../types/resaleDetailPageTypes";

export default function ResaleDetailModelInfo({
  hasModelInfo,
  kindLabel,
  modelNumber,
  size,
  colorLabel,
  colorCssValue,
  measurementsLabel,
  volumeLabel,
}: ResaleDetailModelInfoProps) {
  if (!hasModelInfo) {
    return null;
  }

  return (
    <dl className="resale-detail-page__meta">
      {kindLabel ? (
        <div className="resale-detail-page__meta-row">
          <dt>種別</dt>
          <dd>{kindLabel}</dd>
        </div>
      ) : null}

      {modelNumber ? (
        <div className="resale-detail-page__meta-row">
          <dt>モデル番号</dt>
          <dd>{modelNumber}</dd>
        </div>
      ) : null}

      {size ? (
        <div className="resale-detail-page__meta-row">
          <dt>サイズ</dt>
          <dd>{size}</dd>
        </div>
      ) : null}

      {colorLabel || colorCssValue ? (
        <div className="resale-detail-page__meta-row">
          <dt>カラー</dt>
          <dd>
            <span className="resale-detail-page__color-value">
              {colorCssValue ? (
                <span
                  className="resale-detail-page__color-swatch"
                  style={{ backgroundColor: colorCssValue }}
                  aria-label={colorLabel || "商品カラー"}
                  title={colorLabel || "商品カラー"}
                />
              ) : null}
              <span>{colorLabel || colorCssValue || "カラー未設定"}</span>
            </span>
          </dd>
        </div>
      ) : null}

      {measurementsLabel && measurementsLabel !== "-" ? (
        <div className="resale-detail-page__meta-row">
          <dt>採寸</dt>
          <dd>{measurementsLabel}</dd>
        </div>
      ) : null}

      {volumeLabel && volumeLabel !== "-" ? (
        <div className="resale-detail-page__meta-row">
          <dt>容量</dt>
          <dd>{volumeLabel}</dd>
        </div>
      ) : null}
    </dl>
  );
}