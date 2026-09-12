// frontend/admin/shell/src/features/productBlueprint/presentation/components/ProductBlueprintDetailAside.tsx

import type { Company } from "../../../../shared/type/company";
import TextLink from "../../../../shared/ui/TextLink/TextLink";
import { formatDateTime } from "../../../../shared/util/dateFormat";
import type { ContractProductBlueprintDetail } from "../../model/productBlueprintDetail";

type ProductBlueprintDetailAsideProps = {
  company: Company;
  productBlueprint: ContractProductBlueprintDetail;
  onOpenBrand: () => void;
  onOpenReport: (reportCaseId: string) => void;
};

export default function ProductBlueprintDetailAside({
  company,
  productBlueprint,
  onOpenBrand,
  onOpenReport,
}: ProductBlueprintDetailAsideProps) {
  return (
    <div className="product-blueprint-detail-page__aside">
      <section className="ui-detail-section">
        <dl className="ui-detail-definition-list ui-detail-definition-list--meta">
          <dt>企業名</dt>
          <dd>{company.name || "-"}</dd>

          <dt>ブランド</dt>
          <dd>
            {productBlueprint.brandId && productBlueprint.brandName ? (
              <TextLink tone="accent" onClick={onOpenBrand}>
                {productBlueprint.brandName}
              </TextLink>
            ) : (
              productBlueprint.brandName || "-"
            )}
          </dd>

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
            {productBlueprint.reportCount > 0 &&
            productBlueprint.latestReportCaseId ? (
              <TextLink
                tone="accent"
                onClick={() =>
                  onOpenReport(productBlueprint.latestReportCaseId)
                }
              >
                {productBlueprint.reportCount.toLocaleString()}
              </TextLink>
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