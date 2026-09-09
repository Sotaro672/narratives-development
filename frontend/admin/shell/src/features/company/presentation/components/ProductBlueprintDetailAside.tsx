// frontend/admin/shell/src/features/company/presentation/components/ProductBlueprintDetailAside.tsx

import type { Company } from "../../../../shared/type/company";
import type { ContractProductBlueprintDetail } from "../../../../shared/type/contractProductBlueprintDetail";
import { formatDateTime } from "../../../../shared/util/dateFormat";

type ProductBlueprintDetailAsideProps = {
  company: Company;
  productBlueprint: ContractProductBlueprintDetail;
  onOpenReport: (reportCaseId: string) => void;
};

export default function ProductBlueprintDetailAside({
  company,
  productBlueprint,
  onOpenReport,
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

          <dt>コメント数</dt>
          <dd>{productBlueprint.commentCount.toLocaleString()}</dd>

          <dt>評価平均</dt>
          <dd>
            {productBlueprint.commentCount > 0
              ? productBlueprint.averageRating.toFixed(1)
              : "-"}
          </dd>

          <dt>通報数</dt>
          <dd>
            {productBlueprint.reportCount > 0 && productBlueprint.latestReportCaseId ? (
              <button
                type="button"
                className="product-blueprint-detail-page__report-link"
                onClick={() => onOpenReport(productBlueprint.latestReportCaseId)}
              >
                {productBlueprint.reportCount.toLocaleString()}
              </button>
            ) : (
              productBlueprint.reportCount.toLocaleString()
            )}
          </dd>

          <dt>作成日時</dt>
          <dd>{formatDateTime(productBlueprint.createdAt)}</dd>

          <dt>最終更新日時</dt>
          <dd>{formatDateTime(productBlueprint.updatedAt)}</dd>
        </dl>
      </section>
    </div>
  );
}