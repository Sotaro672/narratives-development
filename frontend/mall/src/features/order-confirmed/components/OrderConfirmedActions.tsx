// frontend/mall/src/features/order-confirmed/components/OrderConfirmedActions.tsx

import Button from "../../../components/ui/Button";

type OrderConfirmedActionsProps = {
  onGoToOrderDetail: () => void;
  onGoToTrade?: () => void;
};

export function OrderConfirmedActions({
  onGoToOrderDetail,
}: OrderConfirmedActionsProps) {
  return (
    <div className="order-confirmed-page__actions">
      <Button
        type="button"
        variant="primary"
        size="lg"
        fullWidth
        onClick={onGoToOrderDetail}
      >
        注文詳細へ
      </Button>
    </div>
  );
}