// frontend/amol/src/features/resale/presentation/components/ResaleDetailReadonlyInfo.tsx

import type { ResaleDetailReadonlyInfoProps } from "../types/resaleDetailPageTypes";

export default function ResaleDetailReadonlyInfo({
  priceLabel,
  conditionLabel,
  statusLabel,
  createdAtLabel,
  updatedAtLabel,
  description,
}: ResaleDetailReadonlyInfoProps) {
  return (
    <>
      <p className="product-detail__price">{priceLabel}</p>

      <dl className="product-detail__meta">
        <div className="product-detail__meta-row">
          <dt>商品の状態</dt>
          <dd>{conditionLabel}</dd>
        </div>

        <div className="product-detail__meta-row">
          <dt>出品ステータス</dt>
          <dd>{statusLabel}</dd>
        </div>

        <div className="product-detail__meta-row">
          <dt>出品日時</dt>
          <dd>{createdAtLabel}</dd>
        </div>

        <div className="product-detail__meta-row">
          <dt>更新日時</dt>
          <dd>{updatedAtLabel}</dd>
        </div>
      </dl>

      <div className="product-detail__description">
        <h2>商品説明</h2>
        <p>{description || "説明文はありません。"}</p>
      </div>
    </>
  );
}