// frontend/amol/src/features/payment/components/PaymentItemsCard.tsx

import { formatPrice } from "../../../components/utils/price";
import type { CartDisplayItem } from "../../shared/types/cart";

type PaymentItemsCardProps = {
  amount: number;
  cartItems: CartDisplayItem[];
  shippingAmount: number;
  subtotalAmount: number;
  taxAmount: number;
};

function getItemTitle(item: CartDisplayItem): string {
  return item.productName || item.title || "商品名未設定";
}

function getItemPrice(item: CartDisplayItem): number | null {
  return item.price ?? null;
}

function getAlcoholModelLabel(item: CartDisplayItem): string {
  if (item.modelLabel) {
    return item.modelLabel;
  }

  const volumeLabel =
    item.volumeValue !== undefined && item.volumeUnit
      ? `${item.volumeValue}${item.volumeUnit}`
      : "";

  return [item.modelNumber, volumeLabel].filter(Boolean).join(" / ");
}

function getApparelModelLabel(item: CartDisplayItem): string {
  return [
    item.color ? `カラー: ${item.color}` : "",
    item.size ? `サイズ: ${item.size}` : "",
  ]
    .filter(Boolean)
    .join(" / ");
}

function getItemModelLabel(item: CartDisplayItem): string {
  return item.modelKind === "alcohol"
    ? getAlcoholModelLabel(item)
    : getApparelModelLabel(item);
}

export function PaymentItemsCard({
  amount,
  cartItems,
  shippingAmount,
  subtotalAmount,
  taxAmount,
}: PaymentItemsCardProps) {
  return (
    <section className="payment-page__card">
      <h2 className="payment-page__section-title">注文内容</h2>

      {cartItems.length > 0 ? (
        <ul className="payment-page__items">
          {cartItems.map((item) => {
            const price = getItemPrice(item);
            const lineAmount = price === null ? null : price * item.qty;
            const title = getItemTitle(item);
            const modelLabel = getItemModelLabel(item);

            return (
              <li className="payment-page__item" key={item.itemKey}>
                <div>
                  <p className="payment-page__item-title">{title}</p>

                  {modelLabel ? (
                    <p className="payment-page__item-meta">{modelLabel}</p>
                  ) : null}

                  <p className="payment-page__item-meta">数量: {item.qty}</p>
                </div>

                <p className="payment-page__item-price">
                  {formatPrice(lineAmount)}
                </p>
              </li>
            );
          })}
        </ul>
      ) : (
        <p className="payment-page__empty">カート情報がありません。</p>
      )}

      <div className="payment-page__total">
        <span>商品小計（税抜）</span>
        <strong>{formatPrice(subtotalAmount)}</strong>
      </div>

      <div className="payment-page__total">
        <span>送料（税抜）</span>
        <strong>{formatPrice(shippingAmount)}</strong>
      </div>

      <div className="payment-page__total">
        <span>消費税</span>
        <strong>{formatPrice(taxAmount)}</strong>
      </div>

      <div className="payment-page__total">
        <span>支払額（税込）</span>
        <strong>{formatPrice(amount)}</strong>
      </div>
    </section>
  );
}