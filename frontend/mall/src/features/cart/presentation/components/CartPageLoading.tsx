// frontend/amol/src/features/cart/presentation/components/CartPageLoading.tsx

export default function CartPageLoading() {
  return (
    <div
      className="cart-page-empty"
      role="status"
      aria-live="polite"
      aria-busy="true"
    >
      <div
        className="cart-page-empty__icon"
        aria-hidden="true"
      >
        🛒
      </div>

      <h1 className="cart-page-empty__title">
        カートを読み込んでいます
      </h1>

      <p className="cart-page-empty__text">
        追加済みのアイテムを確認しています。
      </p>
    </div>
  );
}