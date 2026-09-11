// frontend/admin/shell/src/features/company/presentation/components/ListDetailPriceList.tsx

import type { ContractListPriceRow } from "../../model/listDetail";
import { formatModelMeta } from "../../../../shared/util/modelMetaFormat";

type ListDetailPriceListProps = {
  prices: ContractListPriceRow[];
};

export default function ListDetailPriceList({
  prices,
}: ListDetailPriceListProps) {
  return (
    <section className="ui-detail-section">
      <h2 className="ui-detail-section__title">価格</h2>

      {prices.length > 0 ? (
        <dl className="ui-detail-definition-list ui-list-detail-price-list">
          {prices.map((price) => (
            <div
              className="ui-detail-price-row"
              key={price.modelId}
            >
              <dt>{formatModelMeta(price)}</dt>
              <dd>{price.price.toLocaleString("ja-JP")}円</dd>
            </div>
          ))}
        </dl>
      ) : (
        <p>価格情報はありません。</p>
      )}
    </section>
  );
}