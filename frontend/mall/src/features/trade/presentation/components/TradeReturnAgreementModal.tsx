// frontend/mall/src/features/trade/presentation/components/TradeReturnAgreementModal.tsx

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

import "../../../../styles/trade.css";

export type TradeReturnAgreementModalProps = {
  open: boolean;
  proposal: TradeReturnProposal | null;
  error?: string | null;
  submitting: boolean;
  accepting: boolean;
  rejecting: boolean;
  onCancel: () => void;
  onAccept: () => void;
  onReject: () => void;
};

function formatCurrency(value: number | undefined): string {
  if (
    value === undefined ||
    !Number.isInteger(value) ||
    value <= 0
  ) {
    return "-";
  }

  return `${value.toLocaleString("ja-JP")}円`;
}

function getReturnRequirementLabel(
  proposal: TradeReturnProposal | null,
): string {
  switch (proposal?.returnRequirement) {
    case "required":
      return "商品の返送が必要";

    case "not_required":
      return "商品の返送は不要";

    default:
      return "-";
  }
}

function isValidProposal(
  proposal: TradeReturnProposal | null,
): proposal is TradeReturnProposal {
  return (
    proposal !== null &&
    proposal.id.trim() !== "" &&
    proposal.agreement === "agree" &&
    !proposal.rejectedAt &&
    (proposal.returnRequirement === "required" ||
      proposal.returnRequirement === "not_required") &&
    proposal.refundAmount !== undefined &&
    Number.isInteger(proposal.refundAmount) &&
    proposal.refundAmount > 0
  );
}

export default function TradeReturnAgreementModal({
  open,
  proposal,
  error,
  submitting,
  accepting,
  rejecting,
  onCancel,
  onAccept,
  onReject,
}: TradeReturnAgreementModalProps) {
  const validProposal = isValidProposal(proposal);
  const handleClose = submitting ? undefined : onCancel;

  const handleAccept = (): void => {
    if (submitting || !validProposal) {
      return;
    }

    onAccept();
  };

  const handleReject = (): void => {
    if (submitting || !validProposal) {
      return;
    }

    onReject();
  };

  return (
    <Modal
      open={open}
      onClose={handleClose}
      size="md"
      mobilePosition="bottom"
      closeOnBackdrop={!submitting}
      closeOnEscape={!submitting}
      ariaLabelledBy="trade-return-agreement-modal-title"
      ariaDescribedBy="trade-return-agreement-modal-description"
      ariaBusy={submitting}
    >
      <ModalHeader onClose={handleClose}>
        <ModalTitle id="trade-return-agreement-modal-title">
          返品条件を確認する
        </ModalTitle>
      </ModalHeader>

      <ModalBody>
        <div className="trade-return-agreement-modal__body">
          <ModalDescription id="trade-return-agreement-modal-description">
            出品者から返品条件が提示されました。内容を確認し、同意するか選択してください。同意するとこの条件が返品・返金条件として確定します。
          </ModalDescription>

          {validProposal ? (
            <>
              <section
                className="trade-return-agreement-modal__section"
                aria-labelledby="trade-return-agreement-refund-label"
              >
                <ModalDescription id="trade-return-agreement-refund-label">
                  返金額
                </ModalDescription>

                <strong className="trade-return-agreement-modal__refund-value">
                  {formatCurrency(proposal.refundAmount)}
                </strong>
              </section>

              <section
                className="trade-return-agreement-modal__section"
                aria-labelledby="trade-return-agreement-requirement-label"
              >
                <ModalDescription id="trade-return-agreement-requirement-label">
                  商品の返送
                </ModalDescription>

                <strong className="trade-return-agreement-modal__requirement-value">
                  {getReturnRequirementLabel(proposal)}
                </strong>
              </section>

              {proposal.returnRequirement === "required" ? (
                <Alert variant="info">
                  この条件に同意すると、PUDOを利用した匿名返品の手続きへ進みます。返送手続きが開始された後は、確定した返金額や返送条件を変更できません。
                </Alert>
              ) : null}

              {proposal.returnRequirement === "not_required" ? (
                <Alert variant="info">
                  この条件に同意した場合、商品を返送する必要はありません。商品を保持したまま、確定した金額の返金手続きへ進みます。
                </Alert>
              ) : null}
            </>
          ) : (
            <Alert variant="error">
              返品条件を確認できません。取引情報を再読み込みしてください。
            </Alert>
          )}

          {error ? <Alert variant="error">{error}</Alert> : null}
        </div>
      </ModalBody>

      <ModalFooter>
        <div className="trade-return-modal__footer-actions">
          <Button
            type="button"
            variant="primary"
            size="md"
            fullWidth
            disabled={submitting || !validProposal}
            onClick={handleAccept}
          >
            {accepting ? "同意中..." : "この条件に同意する"}
          </Button>

          <Button
            type="button"
            variant="secondary"
            size="md"
            fullWidth
            disabled={submitting || !validProposal}
            onClick={handleReject}
          >
            {rejecting ? "送信中..." : "この条件に同意しない"}
          </Button>
        </div>
      </ModalFooter>
    </Modal>
  );
}