// frontend/admin/shell/src/features/company/presentation/components/ProductBlueprintDetailAside.tsx

import type { Company } from "../../../../shared/type/company";
import type { ContractProductBlueprintDetail } from "../../../../shared/type/contractProductBlueprintDetail";
import { formatDateTime } from "../../../../shared/util/dateFormat";

type ProductBlueprintDetailAsideProps = {
  company: Company;
  productBlueprint: ContractProductBlueprintDetail;
};

export default function ProductBlueprintDetailAside({
  company,
  productBlueprint,
}: ProductBlueprintDetailAsideProps) {
  return (
    <div className="product-blueprint-detail-page__aside">
      <section className="ui-detail-section">
        <dl className="ui-detail-definition-list product-blueprint-detail-page__definition-list">
          <dt>企業名</dt>
          <dd>{company.name || "-"}</dd>

          <dt>ブランド</dt>
          <dd>{productBlueprint.brandName || "-"}</dd>

          <dt>担当者</dt>
          <dd>{productBlueprint.assigneeName || "-"}</dd>

          <dt>タグ種別</dt>
          <dd>{productBlueprint.productIdTagType || "-"}</dd>

          <dt>作成日時</dt>
          <dd>{formatDateTime(productBlueprint.createdAt)}</dd>

          <dt>最終更新日時</dt>
          <dd>{formatDateTime(productBlueprint.updatedAt)}</dd>
        </dl>
      </section>
    </div>
  );
}