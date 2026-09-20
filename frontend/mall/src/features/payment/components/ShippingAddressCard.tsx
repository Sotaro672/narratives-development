// frontend/mall/src/features/payment/components/ShippingAddressCard.tsx

import Button from "../../../components/ui/Button";
import Card from "../../../components/ui/Card";
import SectionHeader from "../../../components/ui/SectionHeader";
import TextButton from "../../../components/ui/TextButton";
import TextState from "../../../components/ui/TextState";
import type { CanonicalShippingAddress } from "../../shared/types/payment";

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
    <Card as="section" variant="panel" className="payment-page__card">
      <SectionHeader
        title="配送先情報"
        titleAs="h2"
        className="payment-page__section-header"
        right={
          <TextButton type="button" onClick={onGoToShippingAddress}>
            配送先を管理
          </TextButton>
        }
      />

      {primaryShippingAddress ? (
        <div className="payment-page__shipping-address">
          {userFullName ? (
            <p className="payment-page__shipping-address-name">{userFullName}</p>
          ) : null}

          {shippingAddressLabel.split("\n").map((line, index) => (
            <p
              className="payment-page__shipping-address-line"
              key={`${line}-${index}`}
            >
              {line}
            </p>
          ))}
        </div>
      ) : (
        <div className="payment-page__empty-block">
          <TextState variant="empty">
            配送先情報が登録されていません。
          </TextState>

          <Button
            type="button"
            variant="primary"
            fullWidth
            onClick={onGoToShippingAddress}
          >
            配送先情報を登録する
          </Button>
        </div>
      )}
    </Card>
  );
}