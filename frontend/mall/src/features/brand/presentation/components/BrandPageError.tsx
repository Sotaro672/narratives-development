// frontend/amol/src/features/brand/presentation/components/BrandPageError.tsx

import { useNavigate } from "react-router-dom";

import Alert from "../../../../components/ui/Alert";
import Button from "../../../../components/ui/Button";

type BrandPageErrorProps = {
  error: string;
  onBack: () => void;
  onRetry?: () => void | Promise<void>;
};

export default function BrandPageError({
  error,
  onBack,
  onRetry,
}: BrandPageErrorProps) {
  const navigate = useNavigate();
  const message = error.trim() || "ブランド情報の取得に失敗しました。";

  return (
    <div className="brand-page brand-page-centered">
      <Alert variant="error" className="brand-page-error-card">
        <h1 className="brand-page-error-title">
          ブランド情報を取得できませんでした
        </h1>

        <p className="brand-page-error-message">{message}</p>

        <div className="brand-page-error-actions">
          <Button type="button" variant="secondary" onClick={onBack}>
            戻る
          </Button>

          {onRetry ? (
            <Button
              type="button"
              variant="primary"
              onClick={() => {
                void onRetry();
              }}
            >
              再読み込み
            </Button>
          ) : null}

          <Button type="button" variant="ghost" onClick={() => navigate("/")}>
            トップへ
          </Button>
        </div>
      </Alert>
    </div>
  );
}