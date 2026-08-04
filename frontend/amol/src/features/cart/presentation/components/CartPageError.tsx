// frontend/amol/src/features/cart/presentation/components/CartPageError.tsx

type CartPageErrorProps = {
  error: string;
  onRetry: () =>
    | void
    | Promise<void>;
};

export default function CartPageError({
  error,
  onRetry,
}: CartPageErrorProps) {
  const message =
    error.trim() ||
    "カートの取得中にエラーが発生しました。";

  return (
    <div
      className="cart-page-empty"
      role="alert"
    >
      <div
        className="cart-page-empty__icon"
        aria-hidden="true"
      >
        ⚠️
      </div>

      <h1 className="cart-page-empty__title">
        カートを取得できませんでした
      </h1>

      <p className="cart-page-empty__text">
        {message}
      </p>

      <button
        type="button"
        className="cart-page-empty__retry-button"
        onClick={() => {
          void onRetry();
        }}
      >
        再読み込み
      </button>
    </div>
  );
}