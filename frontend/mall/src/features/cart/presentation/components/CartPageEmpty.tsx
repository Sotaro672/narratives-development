// frontend/amol/src/features/cart/presentation/components/CartPageEmpty.tsx

export default function CartPageEmpty() {
  return (
    <div className="cart-page-empty">
      <div
        className="cart-page-empty__icon"
        aria-hidden="true"
      >
        🛒
      </div>

      <h1 className="cart-page-empty__title">
        カートは空です
      </h1>

      <p className="cart-page-empty__text">
        応援したいリストやアイテムを追加すると、ここに表示されます。
      </p>
    </div>
  );
}