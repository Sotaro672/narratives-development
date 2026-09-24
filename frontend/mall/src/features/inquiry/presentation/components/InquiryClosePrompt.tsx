// frontend/amol/src/features/inquiry/presentation/components/InquiryClosePrompt.tsx

import Alert from "../../../../components/ui/Alert";
import Button from "../../../../components/ui/Button";
import Card from "../../../../components/ui/Card";
import ChatMessageHeader from "../../../shared/presentation/components/ChatMessageHeader";

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
    <Card
      as="article"
      padding="md"
      className="chat-detail-page__reply chat-detail-page__reply--system"
    >
      <ChatMessageHeader
        name="テナント"
        showAvatar={false}
      />

      <p className="chat-detail-page__content">
        クローズしますか？
      </p>

      {error ? (
        <Alert variant="error">
          {error}
        </Alert>
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
    </Card>
  );
}