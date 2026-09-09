// frontend/admin/shell/src/features/company/presentation/components/ListDetailAside.tsx

import type { Company } from "../../../../shared/type/company";
import type { ContractListDetail } from "../../../../shared/type/contractListDetail";
import { formatDateTime } from "../../../../shared/util/dateFormat";

type ListDetailAsideProps = {
  company: Company;
  list: ContractListDetail;
  onOpenProductBlueprint: () => void;
  onOpenTokenBlueprint: () => void;
};

export default function ListDetailAside({
  company,
  list,
  onOpenProductBlueprint,
  onOpenTokenBlueprint,
}: ListDetailAsideProps) {
  return (
    <>
      <section className="ui-detail-section">
        <dl className="ui-detail-definition-list ui-list-detail-meta-list">
          <dt>企業名</dt>
          <dd>{company.name || "-"}</dd>
        </dl>
      </section>

      <section className="ui-detail-section">
        <dl className="ui-detail-definition-list ui-list-detail-meta-list">
          <dt>商品名</dt>
          <dd>
            <button
              type="button"
              className="ui-list-detail-meta-link"
              onClick={onOpenProductBlueprint}
            >
              {list.productName || "-"}
            </button>
          </dd>

          <dt>商品ブランド名</dt>
          <dd>{list.productBrandName || "-"}</dd>
        </dl>
      </section>

      <section className="ui-detail-section">
        <dl className="ui-detail-definition-list ui-list-detail-meta-list">
          <dt>トークン名</dt>
          <dd>
            <button
              type="button"
              className="ui-list-detail-meta-link"
              onClick={onOpenTokenBlueprint}
            >
              {list.tokenName || "-"}
            </button>
          </dd>

          <dt>トークンブランド名</dt>
          <dd>{list.tokenBrandName || "-"}</dd>
        </dl>
      </section>

      <section className="ui-detail-section">
        <dl className="ui-detail-definition-list ui-list-detail-meta-list">
          <dt>出品ID</dt>
          <dd>{list.readableId || list.id || "-"}</dd>

          <dt>担当者</dt>
          <dd>{list.assigneeName || "-"}</dd>

          <dt>作成日時</dt>
          <dd className="ui-detail-definition-list__nowrap">
            {formatDateTime(list.createdAt)}
          </dd>

          <dt>最終更新日時</dt>
          <dd className="ui-detail-definition-list__nowrap">
            {formatDateTime(list.updatedAt)}
          </dd>
        </dl>
      </section>
    </>
  );
}