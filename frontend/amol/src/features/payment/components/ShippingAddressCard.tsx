//frontend\amol\src\features\payment\components\ShippingAddressCard.tsx
import type { CanonicalShippingAddress } from "../types";

type ShippingAddressCardProps = {
  primaryShippingAddress: CanonicalShippingAddress | null;
  shippingAddressLabel: string;
  userFullName: string;
  onGoToShippingAddress: () => void;
};

export function ShippingAddressCard({
  primaryShippingAddress,
  shippingAddressLabel,
  userFullName,
  onGoToShippingAddress,
}: ShippingAddressCardProps) {
  return (
    <section className="payment-page__card">
      <div className="payment-page__section-header">
        <h2 className="payment-page__section-title">配送先情報</h2>
        <button
          type="button"
          className="payment-page__text-button"
          onClick={onGoToShippingAddress}
        >
          配送先を管理
        </button>
      </div>

      {primaryShippingAddress ? (
        <div className="payment-page__shipping-address">
          {userFullName ? (
            <p className="payment-page__shipping-address-name">
              {userFullName}
            </p>
          ) : null}

          {shippingAddressLabel.split("\n").map((line) => (
            <p className="payment-page__shipping-address-line" key={line}>
              {line}
            </p>
          ))}
        </div>
      ) : (
        <div className="payment-page__empty-block">
          <p>配送先情報が登録されていません。</p>
          <button
            type="button"
            className="payment-page__primary-button"
            onClick={onGoToShippingAddress}
          >
            配送先情報を登録する
          </button>
        </div>
      )}
    </section>
  );
}