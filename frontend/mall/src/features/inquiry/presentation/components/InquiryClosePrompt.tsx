// frontend/amol/src/features/inquiry/presentation/components/InquiryClosePrompt.tsx

import Button from "../../../../components/ui/Button";

type InquiryClosePromptProps = {
  error?: string | null;
  closing: boolean;
  onClose: () => void;
};

export default function InquiryClosePrompt({
  error,
  closing,
  onClose,
}: InquiryClosePromptProps) {
  return (
    <article className="chat-detail-page__reply chat-detail-page__reply--system">
      <div className="chat-detail-page__message-head">
        <div>
          <span className="chat-detail-page__sender">
            テナント
          </span>
        </div>
      </div>

      <p className="chat-detail-page__content">
        クローズしますか？
      </p>

      {error ? (
        <div
          className="chat-detail-page__modal-error"
          role="alert"
        >
          {error}
        </div>
      ) : null}

      <div className="chat-detail-page__close-prompt-actions">
        <Button
          variant="primary"
          size="sm"
          onClick={onClose}
          disabled={closing}
        >
          {closing ? "クローズ中..." : "クローズする"}
        </Button>
      </div>
    </article>
  );
}