// frontend/amol/src/features/resale/presentation/components/ResaleDetailReadonlyInfo.tsx

import ProductDescription from "../../../shared/presentation/components/ProductDescription";
import ProductMetaList from "../../../shared/presentation/components/ProductMetaList";
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

      <ProductMetaList
        items={[
          {
            label: "商品の状態",
            value: conditionLabel,
          },
          {
            label: "出品ステータス",
            value: statusLabel,
          },
          {
            label: "出品日時",
            value: createdAtLabel,
          },
          {
            label: "更新日時",
            value: updatedAtLabel,
          },
        ]}
      />

      <ProductDescription description={description || "説明文はありません。"} />
    </>
  );
}