// frontend/mall/src/features/trade/presentation/components/TradeReturnReceiptModal.tsx

import Alert from "../../../../components/ui/Alert";
import Button from "../../../../components/ui/Button";
import Modal, {
  ModalBody,
  ModalDescription,
  ModalFooter,
  ModalHeader,
  ModalTitle,
} from "../../../../components/ui/Modal";
import type { TradeReturnProposal } from "../../../shared/types/trade";

export type TradeReturnReceiptModalProps = {
  open: boolean;
  proposal: TradeReturnProposal | null;
  refundAmount: number;
  error?: string | null;
  submitting: boolean;
  canSubmit: boolean;
  onCancel: () => void;
  onSubmit: () => void;
};

function formatCurrency(value: number): string {
  if (!Number.isInteger(value) || value <= 0) {
    return "-";
  }

  return `${value.toLocaleString("ja-JP")}円`;
}

export default function TradeReturnReceiptModal({
  open,
  proposal,
  refundAmount,
  error,
  submitting,
  canSubmit,
  onCancel,
  onSubmit,
}: TradeReturnReceiptModalProps) {
  const handleClose = submitting ? undefined : onCancel;

  const handleSubmit = (): void => {
    if (!canSubmit || submitting) {
      return;
    }

    onSubmit();
  };

  const validProposal =
    proposal !== null &&
    proposal.id.trim() !== "" &&
    proposal.agreement === "agree" &&
    !proposal.rejectedAt &&
    proposal.returnRequirement === "required" &&
    proposal.refundAmount !== undefined &&
    Number.isInteger(proposal.refundAmount) &&
    proposal.refundAmount > 0;

  return (
    <Modal
      open={open}
      onClose={handleClose}
      size="md"
      mobilePosition="bottom"
      closeOnBackdrop={!submitting}
      closeOnEscape={!submitting}
      ariaLabelledBy="trade-return-receipt-modal-title"
      ariaDescribedBy="trade-return-receipt-modal-description"
      ariaBusy={submitting}
    >
      <ModalHeader onClose={handleClose}>
        <ModalTitle id="trade-return-receipt-modal-title">
          返品受領・返金
        </ModalTitle>
      </ModalHeader>

      <ModalBody>
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            gap: 20,
          }}
        >
          <ModalDescription id="trade-return-receipt-modal-description">
            購入者と合意済みの返品条件を確認し、返品商品の受領後に返金処理を進めてください。
          </ModalDescription>

          <Alert variant="warning">
            実際に返品商品を受領したことを確認してから処理してください。現在のAMOLは運送業者から配送状況を取得しません。
          </Alert>

          {validProposal ? (
            <section
              aria-label="合意済みの返品条件"
              style={{
                display: "flex",
                flexDirection: "column",
                gap: 12,
                padding: 16,
                border: "1px solid #e5e7eb",
                borderRadius: 16,
                background: "#ffffff",
              }}
            >
              <strong>
                合意済みの返品条件
              </strong>

              <div
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  gap: 16,
                }}
              >
                <span>商品の返送</span>
                <strong>必要</strong>
              </div>

              <div
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  gap: 16,
                }}
              >
                <span>商品代金の返金額（税込）</span>
                <strong>{formatCurrency(refundAmount)}</strong>
              </div>
            </section>
          ) : (
            <Alert variant="error">
              合意済みの返品条件を取得できません。取引情報を再読み込みしてください。
            </Alert>
          )}

          <Alert variant="info">
            返金額は購入者が同意した返品条件から自動的に適用されます。この画面では返金条件を変更できません。
          </Alert>

          {error ? (
            <Alert variant="error">
              {error}
            </Alert>
          ) : null}
        </div>
      </ModalBody>

      <ModalFooter>
        <div
          style={{
            display: "flex",
            width: "100%",
            flexDirection: "column",
            gap: 8,
          }}
        >
          <Button
            type="button"
            variant="primary"
            size="md"
            fullWidth
            disabled={!canSubmit || submitting || !validProposal}
            onClick={handleSubmit}
          >
            {submitting
              ? "受領・返金処理中..."
              : "返品受領・返金を進める"}
          </Button>

          <Button
            type="button"
            variant="secondary"
            size="md"
            fullWidth
            disabled={submitting}
            onClick={onCancel}
          >
            戻る
          </Button>
        </div>
      </ModalFooter>
    </Modal>
  );
}