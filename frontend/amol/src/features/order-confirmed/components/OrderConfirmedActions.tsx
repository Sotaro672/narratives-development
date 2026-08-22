// frontend/amol/src/features/order-confirmed/components/OrderConfirmedActions.tsx

type OrderConfirmedActionsProps = {
  onGoToOrderDetail: () => void;
  onGoToLists: () => void;
};

export function OrderConfirmedActions({
  onGoToOrderDetail,
  onGoToLists,
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

      <button
        type="button"
        className="order-confirmed-page__secondary-button"
        onClick={onGoToLists}
      >
        商品一覧へ
      </button>
    </div>
  );
}