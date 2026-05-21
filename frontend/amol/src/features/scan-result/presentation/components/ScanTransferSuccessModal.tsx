// frontend/amol/src/features/scan-result/presentation/components/ScanTransferSuccessModal.tsx
import type { ScanTransferSuccessModalViewModel } from "../../application";

type Props = {
  open: boolean;
  loading: boolean;
  error: string | null;
  viewModel: ScanTransferSuccessModalViewModel | null;
  resolvedContentsReady: boolean;
  onClose: () => void;
  onOpenContents: () => void;
};

function displayText(value: string): string {
  return value.trim() || "-";
}

export default function ScanTransferSuccessModal({
  open,
  loading,
  error,
  viewModel,
  resolvedContentsReady,
  onClose,
  onOpenContents,
}: Props) {
  if (!open) return null;

  const canOpenContents = Boolean(viewModel?.mintAddress);

  return (
    <div className="scan-transfer-modal-backdrop" onClick={onClose}>
      <div
        className="scan-transfer-modal"
        role="dialog"
        aria-modal="true"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="scan-transfer-modal__header">
          <h2 className="scan-transfer-modal__title">トークン移譲結果</h2>

          <button
            type="button"
            className="scan-transfer-modal__close"
            onClick={onClose}
            aria-label="閉じる"
          >
            ×
          </button>
        </div>

        {loading ? (
          <p className="scan-transfer-modal__message">移譲処理中です...</p>
        ) : error ? (
          <p className="scan-transfer-modal__error">{error}</p>
        ) : viewModel ? (
          <div className="scan-transfer-modal__body">
            <p className="scan-transfer-modal__message">
              トークンの移譲が完了しました。
            </p>

            <dl className="scan-transfer-modal__list">
              <div>
                <dt>商品名</dt>
                <dd>{displayText(viewModel.productName)}</dd>
              </div>

              <div>
                <dt>トークン名</dt>
                <dd>{displayText(viewModel.tokenName)}</dd>
              </div>

              <div>
                <dt>移譲元</dt>
                <dd>{displayText(viewModel.fromName)}</dd>
              </div>

              <div>
                <dt>移譲先</dt>
                <dd>{displayText(viewModel.toName)}</dd>
              </div>

              {viewModel.walletUpdated ? (
                <div>
                  <dt>Wallet 更新</dt>
                  <dd>完了</dd>
                </div>
              ) : null}
            </dl>

            {resolvedContentsReady ? (
              <p className="scan-transfer-modal__message">
                トークンコンテンツを表示できる状態になりました。
              </p>
            ) : null}
          </div>
        ) : (
          <p className="scan-transfer-modal__message">
            移譲結果を取得できませんでした。
          </p>
        )}

        <div className="scan-transfer-modal__footer">
          {canOpenContents ? (
            <button
              type="button"
              className="scan-transfer-modal__button"
              onClick={onOpenContents}
            >
              コンテンツを見る
            </button>
          ) : null}

          <button
            type="button"
            className="scan-transfer-modal__button scan-transfer-modal__button--secondary"
            onClick={onClose}
          >
            閉じる
          </button>
        </div>
      </div>
    </div>
  );
}