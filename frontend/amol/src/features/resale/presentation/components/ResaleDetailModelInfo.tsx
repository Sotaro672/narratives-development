// frontend/amol/src/features/resale/presentation/components/ResaleDetailModelInfo.tsx

import SectionHeader from "../../../../components/ui/SectionHeader";

import type {
  ResaleDetailModelInfoProps,
} from "../types/resaleDetailPageTypes";

export default function ResaleDetailModelInfo({
  hasModelInfo,
  kindLabel,
  modelNumber,
  size,
  colorLabel,
  measurementsLabel,
  volumeLabel,
}: ResaleDetailModelInfoProps) {
  if (!hasModelInfo) {
    return null;
  }

  return (
    <section className="page-card">
      <SectionHeader
        title="商品情報"
        titleAs="h2"
      />

      <dl className="page-definition-list resale-detail-page__readonly-meta">
        {kindLabel ? (
          <div className="page-definition-list__row">
            <dt>種別</dt>
            <dd>{kindLabel}</dd>
          </div>
        ) : null}

        {modelNumber ? (
          <div className="page-definition-list__row">
            <dt>モデル番号</dt>
            <dd>{modelNumber}</dd>
          </div>
        ) : null}

        {size ? (
          <div className="page-definition-list__row">
            <dt>サイズ</dt>
            <dd>{size}</dd>
          </div>
        ) : null}

        {colorLabel && colorLabel !== "-" ? (
          <div className="page-definition-list__row">
            <dt>カラー</dt>
            <dd>{colorLabel}</dd>
          </div>
        ) : null}

        {measurementsLabel &&
        measurementsLabel !== "-" ? (
          <div className="page-definition-list__row">
            <dt>採寸</dt>
            <dd>{measurementsLabel}</dd>
          </div>
        ) : null}

        {volumeLabel && volumeLabel !== "-" ? (
          <div className="page-definition-list__row">
            <dt>容量</dt>
            <dd>{volumeLabel}</dd>
          </div>
        ) : null}
      </dl>
    </section>
  );
}