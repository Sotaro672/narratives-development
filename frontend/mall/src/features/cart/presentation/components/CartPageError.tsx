// frontend/amol/src/features/cart/presentation/components/CartPageError.tsx

import Button from "../../../../components/ui/Button";
import StatePanel from "../../../../components/ui/StatePanel";

type CartPageErrorProps = {
  error: string;
  onRetry: () => void | Promise<void>;
};

export default function CartPageError({
  error,
  onRetry,
}: CartPageErrorProps) {
  const message =
    error.trim() ||
    "カートの取得中にエラーが発生しました。";

  return (
    <StatePanel
      variant="error"
      icon="⚠️"
      title="カートを取得できませんでした"
      description={message}
      className="cart-page-empty"
      action={
        <Button
          type="button"
          variant="primary"
          onClick={() => {
            void onRetry();
          }}
        >
          再読み込み
        </Button>
      }
    />
  );
}