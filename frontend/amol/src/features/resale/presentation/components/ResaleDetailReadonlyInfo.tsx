// frontend/amol/src/features/resale/presentation/components/ResaleDetailReadonlyInfo.tsx

import type {
  ResaleDetailReadonlyInfoProps,
} from "../types/resaleDetailPageTypes";

export default function ResaleDetailReadonlyInfo({
  priceLabel,
  conditionLabel,
  statusLabel,
  createdAtLabel,
  updatedAtLabel,
  description,
}: ResaleDetailReadonlyInfoProps) {
  return (
    <section className="page-card">
      <div className="page-stack">
        <dl className="page-definition-list">
          <div className="page-definition-list__row">
            <dt>販売価格</dt>
            <dd>{priceLabel}</dd>
          </div>

          <div className="page-definition-list__row">
            <dt>商品の状態</dt>
            <dd>{conditionLabel}</dd>
          </div>

          <div className="page-definition-list__row">
            <dt>出品ステータス</dt>
            <dd>{statusLabel}</dd>
          </div>

          <div className="page-definition-list__row">
            <dt>出品日時</dt>
            <dd>{createdAtLabel}</dd>
          </div>

          <div className="page-definition-list__row">
            <dt>更新日時</dt>
            <dd>{updatedAtLabel}</dd>
          </div>
        </dl>

        <div>
          <h3 className="page-card__subtitle">説明文</h3>
          <p className="page-card__text resale-detail-page__description">
            {description || "説明文はありません。"}
          </p>
        </div>
      </div>
    </section>
  );
}