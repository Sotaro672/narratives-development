// frontend/amol/src/features/order-confirmed/components/OrderConfirmedActions.tsx

type OrderConfirmedActionsProps = {
  onGoToOrderDetail: () => void;
  onGoToTrade?: () => void;
};

export function OrderConfirmedActions({
  onGoToOrderDetail,
  onGoToTrade,
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

      {onGoToTrade ? (
        <button
          type="button"
          className="order-confirmed-page__secondary-button"
          onClick={onGoToTrade}
        >
          取引画面へ
        </button>
      ) : null}
    </div>
  );
}