// frontend/amol/src/features/resale/presentation/components/ResaleDetailReadonlyInfo.tsx

import ProductMetaList from "../../../shared/presentation/components/ProductMetaList";
import type { ResaleDetailReadonlyInfoProps } from "../types/resaleDetailPageTypes";

type ResaleDetailReadonlyInfoViewProps = Pick<
  ResaleDetailReadonlyInfoProps,
  "statusLabel" | "createdAtLabel" | "updatedAtLabel"
>;

export default function ResaleDetailReadonlyInfo({
  statusLabel,
  createdAtLabel,
  updatedAtLabel,
}: ResaleDetailReadonlyInfoViewProps) {
  return (
    <div className="resale-detail-page__listing-meta-card">
      <div className="resale-detail-page__listing-status-tab">
        {statusLabel}
      </div>

      <ProductMetaList
        className="resale-detail-page__listing-meta"
        items={[
          { label: "出品日時", value: createdAtLabel },
          { label: "更新日時", value: updatedAtLabel },
        ]}
      />
    </div>
  );
}