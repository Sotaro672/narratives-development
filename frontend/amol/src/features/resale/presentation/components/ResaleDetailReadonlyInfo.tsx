// frontend/amol/src/features/resale/presentation/components/ResaleDetailReadonlyInfo.tsx

import ProductDescription from "../../../shared/presentation/components/ProductDescription";
import ProductPrice from "../../../shared/presentation/components/ProductPrice";

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
      <ProductPrice priceLabel={priceLabel} />

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

      <ProductDescription description={description || "説明文はありません。"} />
    </>
  );
}