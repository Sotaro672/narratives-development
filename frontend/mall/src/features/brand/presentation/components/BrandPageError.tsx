// frontend/amol/src/features/brand/presentation/components/BrandPageError.tsx

import {
  Link,
} from "react-router-dom";

type BrandPageErrorProps = {
  error: string;
  onBack: () => void;
  onRetry?: () =>
    | void
    | Promise<void>;
};

export default function BrandPageError({
  error,
  onBack,
  onRetry,
}: BrandPageErrorProps) {
  const message =
    error.trim() ||
    "ブランド情報の取得に失敗しました。";

  return (
    <div className="brand-page brand-page-centered">
      <div
        className="brand-page-error-card"
        role="alert"
      >
        <h1>
          ブランド情報を取得できませんでした
        </h1>

        <p>{message}</p>

        <div className="brand-page-error-actions">
          <button
            type="button"
            onClick={onBack}
          >
            戻る
          </button>

          {onRetry ? (
            <button
              type="button"
              onClick={() => {
                void onRetry();
              }}
            >
              再読み込み
            </button>
          ) : null}

          <Link to="/">
            トップへ
          </Link>
        </div>
      </div>
    </div>
  );
}