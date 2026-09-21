// frontend/mall/src/features/trade/presentation/components/TradeReturnProposalModal.tsx

import Alert from "../../../../components/ui/Alert";
import Button from "../../../../components/ui/Button";
import Chip from "../../../../components/ui/Chip";
import Input from "../../../../components/ui/Input";
import Modal, {
  ModalBody,
  ModalDescription,
  ModalFooter,
  ModalHeader,
  ModalTitle,
} from "../../../../components/ui/Modal";
import type {
  TradeReturnAgreement,
  TradeReturnRequirement,
} from "../../../shared/types/trade";

export type TradeReturnProposalModalProps = {
  open: boolean;
  agreement: TradeReturnAgreement | null;
  returnRequirement: TradeReturnRequirement | null;
  refundAmount: number | "";
  refundAmountMax: number;
  error?: string | null;
  submitting: boolean;
  onAgreementChange: (value: TradeReturnAgreement) => void;
  onReturnRequirementChange: (value: TradeReturnRequirement) => void;
  onRefundAmountChange: (value: number | "") => void;
  onCancel: () => void;
  onSubmit: () => void;
};

function formatCurrency(value: number): string {
  if (!Number.isFinite(value) || value < 0) {
    return "-";
  }

  return `${value.toLocaleString("ja-JP")}円`;
}

export default function TradeReturnProposalModal({
  open,
  agreement,
  returnRequirement,
  refundAmount,
  refundAmountMax,
  error,
  submitting,
  onAgreementChange,
  onReturnRequirementChange,
  onRefundAmountChange,
  onCancel,
  onSubmit,
}: TradeReturnProposalModalProps) {
  const agreed = agreement === "agree";

  const validRefundAmount =
    typeof refundAmount === "number" &&
    Number.isInteger(refundAmount) &&
    refundAmount > 0 &&
    refundAmount <= refundAmountMax;

  const canSubmit =
    !submitting &&
    agreement !== null &&
    (agreement === "disagree" ||
      (returnRequirement !== null && validRefundAmount));

  const handleAgreementChange = (
    value: TradeReturnAgreement,
  ): void => {
    onAgreementChange(value);
  };

  const handleRefundAmountChange = (
    value: string,
  ): void => {
    if (value === "") {
      onRefundAmountChange("");
      return;
    }

    const parsed = Number(value);

    if (!Number.isFinite(parsed)) {
      return;
    }

    onRefundAmountChange(parsed);
  };

  const handleSubmit = (): void => {
    if (!canSubmit) {
      return;
    }

    onSubmit();
  };

  const handleClose = submitting ? undefined : onCancel;

  return (
    <Modal
      open={open}
      onClose={handleClose}
      size="md"
      mobilePosition="bottom"
      closeOnBackdrop={!submitting}
      closeOnEscape={!submitting}
      ariaLabelledBy="trade-return-proposal-modal-title"
      ariaDescribedBy="trade-return-proposal-modal-description"
      ariaBusy={submitting}
    >
      <ModalHeader onClose={handleClose}>
        <ModalTitle id="trade-return-proposal-modal-title">
          返品相談に回答する
        </ModalTitle>
      </ModalHeader>

      <ModalBody>
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            gap: 24,
          }}
        >
          <ModalDescription id="trade-return-proposal-modal-description">
            購入者からの返品相談に対する回答内容を入力してください。提示した条件は購入者に表示され、購入者が同意した場合に返品手続きへ進みます。
          </ModalDescription>

          <section aria-labelledby="trade-return-agreement-label">
            <ModalDescription id="trade-return-agreement-label">
              返品に合意しますか？
            </ModalDescription>

            <div
              role="group"
              aria-labelledby="trade-return-agreement-label"
              style={{
                display: "flex",
                flexWrap: "wrap",
                gap: 8,
                marginTop: 12,
              }}
            >
              <Chip
                selected={agreement === "agree"}
                disabled={submitting}
                onClick={() => handleAgreementChange("agree")}
              >
                返品に合意する
              </Chip>

              <Chip
                selected={agreement === "disagree"}
                disabled={submitting}
                onClick={() => handleAgreementChange("disagree")}
              >
                返品に合意しない
              </Chip>
            </div>
          </section>

          {agreed ? (
            <>
              <section aria-labelledby="trade-return-requirement-label">
                <ModalDescription id="trade-return-requirement-label">
                  商品を返品してもらいますか？
                </ModalDescription>

                <div
                  role="group"
                  aria-labelledby="trade-return-requirement-label"
                  style={{
                    display: "flex",
                    flexWrap: "wrap",
                    gap: 8,
                    marginTop: 12,
                  }}
                >
                  <Chip
                    selected={returnRequirement === "required"}
                    disabled={submitting}
                    onClick={() =>
                      onReturnRequirementChange("required")
                    }
                  >
                    商品を返品してもらう
                  </Chip>

                  <Chip
                    selected={returnRequirement === "not_required"}
                    disabled={submitting}
                    onClick={() =>
                      onReturnRequirementChange("not_required")
                    }
                  >
                    商品を返品してもらわない
                  </Chip>
                </div>
              </section>

              <Input
                id="trade-return-proposal-refund-amount"
                type="number"
                inputMode="numeric"
                min={1}
                max={
                  refundAmountMax > 0
                    ? refundAmountMax
                    : undefined
                }
                step={1}
                label="返金額"
                required
                value={refundAmount}
                disabled={submitting}
                helperText={`返金上限: ${formatCurrency(
                  refundAmountMax,
                )}`}
                onChange={(event) => {
                  handleRefundAmountChange(event.target.value);
                }}
              />
            </>
          ) : null}

          {agreement === "disagree" ? (
            <Alert variant="info">
              返品に合意しない場合、商品返品および返金は行わず、購入者とのメッセージ相談を継続します。合意に至らない場合は運営への対応依頼に進むことができます。
            </Alert>
          ) : null}

          {agreed && returnRequirement === "required" ? (
            <Alert variant="info">
              購入者が提示条件に同意した後、PUDO匿名返品の手続きへ進みます。
            </Alert>
          ) : null}

          {agreed && returnRequirement === "not_required" ? (
            <Alert variant="info">
              商品は購入者が保持したまま、指定した金額を返金する条件として提示します。
            </Alert>
          ) : null}

          {error ? (
            <Alert variant="error">
              {error}
            </Alert>
          ) : null}
        </div>
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
            : agreement === "disagree"
              ? "返品に合意しない"
              : "返品条件を提示する"}
        </Button>
      </ModalFooter>
    </Modal>
  );
}