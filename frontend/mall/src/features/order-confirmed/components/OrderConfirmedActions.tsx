// frontend/amol/src/features/order-confirmed/components/OrderConfirmedActions.tsx

type OrderConfirmedActionsProps = {
  onGoToOrderDetail: () => void;
  onGoToTrade?: () => void;
};

export function OrderConfirmedActions({
  onGoToOrderDetail,
}: OrderConfirmedActionsProps) {
  return (
    <div className="order-confirmed-page__actions">
      <button
        type="button"
        className="order-confirmed-page__primary-button"
        onClick={onGoToOrderDetail}
      >
        注文詳細へ
      </button>
    </div>
  );
}