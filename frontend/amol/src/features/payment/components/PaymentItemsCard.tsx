//frontend\amol\src\features\payment\components\PaymentItemsCard.tsx
import { formatYen, getModelPrice, getModelVariation } from "../../cart/utils/cartUtils";
import type { CanonicalCartDisplayItem } from "../types";

type PaymentItemsCardProps = {
  amount: number;
  cartItems: CanonicalCartDisplayItem[];
};

export function PaymentItemsCard({ amount, cartItems }: PaymentItemsCardProps) {
  return (
    <section className="payment-page__card">
      <h2 className="payment-page__section-title">注文内容</h2>

      {cartItems.length > 0 ? (
        <ul className="payment-page__items">
          {cartItems.map((item) => {
            const catalog = item.catalog;
            const model = getModelVariation(catalog, item.modelId);
            const price = getModelPrice(catalog, item.modelId);
            const lineAmount = price === null ? null : price * item.qty;
            const title =
              catalog?.productBlueprint.productName ||
              catalog?.list.title ||
              "商品名未設定";

            return (
              <li className="payment-page__item" key={item.itemKey}>
                <div>
                  <p className="payment-page__item-title">{title}</p>

                  <p className="payment-page__item-meta">
                    {model?.colorName ? `カラー: ${model.colorName}` : ""}
                    {model?.colorName && model?.size ? " / " : ""}
                    {model?.size ? `サイズ: ${model.size}` : ""}
                  </p>

                  <p className="payment-page__item-meta">数量: {item.qty}</p>
                </div>

                <p className="payment-page__item-price">
                  {lineAmount === null ? "価格未設定" : formatYen(lineAmount)}
                </p>
              </li>
            );
          })}
        </ul>
      ) : (
        <p className="payment-page__empty">カート情報がありません。</p>
      )}

      <div className="payment-page__total">
        <span>合計</span>
        <strong>{formatYen(amount)}</strong>
      </div>
    </section>
  );
}