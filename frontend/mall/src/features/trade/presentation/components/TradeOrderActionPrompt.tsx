// frontend/mall/src/features/trade/presentation/components/TradeOrderActionPrompt.tsx

import Alert from "../../../../components/ui/Alert";
import Button from "../../../../components/ui/Button";
import Card from "../../../../components/ui/Card";
import ChatMessageHeader from "../../../shared/presentation/components/ChatMessageHeader";
import type { TradeOrderActionKind } from "../util/tradeChatDetail";

type TradeOrderActionPromptProps = {
  action: TradeOrderActionKind;
  processing: boolean;
  error?: string | null;
  identityVerificationRequired?: boolean;
  identityVerificationLoading?: boolean;
  onAction: () => void;
};

function getPromptText(
  action: TradeOrderActionKind,
  identityVerificationRequired: boolean,
): string {
  switch (action) {
    case "cancel":
      return "注文をキャンセルしますか？";
    case "dispatch":
      return "商品を発送しますか？";
    case "start-return-consultation":
      return identityVerificationRequired
        ? "返品について相談するには、本人確認が必要です。"
        : "返品について相談しますか？";
    case "respond-return-consultation":
      return "購入者から返品についての相談が届いています。";
    case "update-return-proposal":
      return "返品相談への回答を変更できます。";
    case "review-return-proposal":
      return "出品者から返品条件が提示されています。";
    case "report-return-dispute":
      return "出品者が返品に合意しませんでした。解決できない場合は運営へ報告できます。";
    case "prepare-return-shipment":
      return "返品用QRを表示しますか？";
    case "receive-return":
      return "返品商品の受領確認と返金処理を進めますか？";
  }
}

function getActionLabel(
  action: TradeOrderActionKind,
  processing: boolean,
  identityVerificationRequired: boolean,
  identityVerificationLoading: boolean,
): string {
  if (
    action === "start-return-consultation" &&
    identityVerificationLoading
  ) {
    return "本人確認情報を確認中...";
  }

  if (
    action === "start-return-consultation" &&
    identityVerificationRequired
  ) {
    return "本人確認を行う";
  }

  if (processing) {
    switch (action) {
      case "cancel":
        return "キャンセル中...";
      case "dispatch":
        return "発送処理中...";
      case "start-return-consultation":
        return "送信中...";
      case "respond-return-consultation":
        return "回答中...";
      case "update-return-proposal":
        return "更新中...";
      case "review-return-proposal":
        return "処理中...";
      case "report-return-dispute":
        return "報告中...";
      case "prepare-return-shipment":
        return "QR準備中...";
      case "receive-return":
        return "受領・返金処理中...";
    }
  }

  switch (action) {
    case "cancel":
      return "注文をキャンセル";
    case "dispatch":
      return "発送する";
    case "start-return-consultation":
      return "返品について相談する";
    case "respond-return-consultation":
      return "返品相談に回答する";
    case "update-return-proposal":
      return "回答を更新する";
    case "review-return-proposal":
      return "返品条件を確認する";
    case "report-return-dispute":
      return "運営へ報告する";
    case "prepare-return-shipment":
      return "返品用QRを表示する";
    case "receive-return":
      return "返品受領・返金を進める";
  }
}

export default function TradeOrderActionPrompt({
  action,
  processing,
  error,
  identityVerificationRequired = false,
  identityVerificationLoading = false,
  onAction,
}: TradeOrderActionPromptProps) {
  const actionProcessing =
    processing || identityVerificationLoading;

  return (
    <Card
      as="article"
      padding="md"
      className="chat-detail-page__reply chat-detail-page__reply--system"
    >
      <ChatMessageHeader
        name="システム"
        showAvatar={false}
      />

      <p className="chat-detail-page__content">
        {getPromptText(
          action,
          identityVerificationRequired,
        )}
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
          onClick={onAction}
          disabled={actionProcessing}
          aria-busy={actionProcessing}
        >
          {getActionLabel(
            action,
            processing,
            identityVerificationRequired,
            identityVerificationLoading,
          )}
        </Button>
      </div>
    </Card>
  );
}