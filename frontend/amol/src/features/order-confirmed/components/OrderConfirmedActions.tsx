//frontend\amol\src\features\order-confirmed\components\OrderConfirmedActions.tsx
type OrderConfirmedActionsProps = {
  onGoToWallet: () => void;
  onGoToLists: () => void;
};

export function OrderConfirmedActions({
  onGoToWallet,
  onGoToLists,
}: OrderConfirmedActionsProps) {
  return (
    <div className="order-confirmed-page__actions">
      <button
        type="button"
        className="order-confirmed-page__primary-button"
        onClick={onGoToWallet}
      >
        ウォレットへ
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