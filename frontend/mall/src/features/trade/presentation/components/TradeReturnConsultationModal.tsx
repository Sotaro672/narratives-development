// frontend/mall/src/features/trade/presentation/components/TradeReturnConsultationModal.tsx

import Alert from "../../../../components/ui/Alert";
import Button from "../../../../components/ui/Button";
import Chip from "../../../../components/ui/Chip";
import Modal, {
  ModalBody,
  ModalDescription,
  ModalFooter,
  ModalHeader,
  ModalTitle,
} from "../../../../components/ui/Modal";
import Textbox from "../../../../components/ui/Textbox";
import {
  TRADE_RETURN_CONSULTATION_REASONS,
  type TradeReturnConsultationReason,
} from "../../../shared/types/trade";

export type TradeReturnConsultationModalProps = {
  open: boolean;
  reason: TradeReturnConsultationReason | null;
  detail: string;
  error?: string | null;
  submitting: boolean;
  onReasonChange: (value: TradeReturnConsultationReason) => void;
  onDetailChange: (value: string) => void;
  onCancel: () => void;
  onSubmit: () => void;
};

const MAX_DETAIL_LENGTH = 5000;

function getReasonLabel(
  reason: TradeReturnConsultationReason,
): string {
  switch (reason) {
    case "not_as_described":
      return "商品説明と状態が異なる";
    case "damaged":
      return "商品が破損している";
    case "wrong_item":
      return "異なる商品が届いた";
    case "other":
      return "その他";
  }
}

export default function TradeReturnConsultationModal({
  open,
  reason,
  detail,
  error,
  submitting,
  onReasonChange,
  onDetailChange,
  onCancel,
  onSubmit,
}: TradeReturnConsultationModalProps) {
  const normalizedDetail = detail.trim();

  const canSubmit =
    reason !== null &&
    normalizedDetail.length > 0 &&
    normalizedDetail.length <= MAX_DETAIL_LENGTH &&
    !submitting;

  const handleClose = submitting ? undefined : onCancel;

  const handleSubmit = () => {
    if (!canSubmit) {
      return;
    }

    onSubmit();
  };

  return (
    <Modal
      open={open}
      onClose={handleClose}
      size="md"
      mobilePosition="bottom"
      closeOnBackdrop={!submitting}
      closeOnEscape={!submitting}
      ariaLabelledBy="trade-return-consultation-modal-title"
      ariaDescribedBy="trade-return-consultation-modal-description"
      ariaBusy={submitting}
    >
      <ModalHeader onClose={handleClose}>
        <ModalTitle id="trade-return-consultation-modal-title">
          返品について相談する
        </ModalTitle>
      </ModalHeader>

      <ModalBody>
        <ModalDescription id="trade-return-consultation-modal-description">
          返品を希望する理由を選択し、商品の状態や問題点を具体的に入力してください。この操作だけでは返品は確定しません。
        </ModalDescription>

        <section aria-labelledby="trade-return-consultation-reason-label">
          <ModalDescription id="trade-return-consultation-reason-label">
            返品理由
          </ModalDescription>

          <div
            role="group"
            aria-labelledby="trade-return-consultation-reason-label"
          >
            {TRADE_RETURN_CONSULTATION_REASONS.map((value) => (
              <Chip
                key={value}
                selected={reason === value}
                disabled={submitting}
                onClick={() => onReasonChange(value)}
              >
                {getReasonLabel(value)}
              </Chip>
            ))}
          </div>
        </section>

        <Textbox
          id="trade-return-consultation-detail"
          label="詳細"
          value={detail}
          rows={6}
          required
          disabled={submitting}
          maxLength={MAX_DETAIL_LENGTH}
          placeholder="商品の状態や商品説明と異なる点などを具体的に入力してください"
          counterText={`${detail.length} / ${MAX_DETAIL_LENGTH}`}
          onChange={(event) => {
            onDetailChange(event.target.value);
          }}
        />

        {error ? (
          <Alert variant="error">
            {error}
          </Alert>
        ) : null}
      </ModalBody>

      <ModalFooter>
        <Button
          type="button"
          variant="primary"
          size="md"
          fullWidth
          disabled={!canSubmit}
          onClick={handleSubmit}
        >
          {submitting
            ? "送信中..."
            : "返品について相談する"}
        </Button>
      </ModalFooter>
    </Modal>
  );
}