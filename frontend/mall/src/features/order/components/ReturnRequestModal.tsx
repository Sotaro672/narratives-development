// frontend/mall/src/features/order/components/ReturnRequestModal.tsx

import { useEffect, useState } from "react";

import Alert from "../../../components/ui/Alert";
import Button from "../../../components/ui/Button";
import Checkbox from "../../../components/ui/Checkbox";
import Chip from "../../../components/ui/Chip";
import Modal, {
  ModalBody,
  ModalDescription,
  ModalFooter,
  ModalHeader,
  ModalTitle,
} from "../../../components/ui/Modal";
import Textbox from "../../../components/ui/Textbox";

export type ReturnPackageState = "unopened" | "opened";

export type ReturnRequestModalProps = {
  open: boolean;
  packageState: ReturnPackageState | null;
  reason: string;
  error?: string | null;
  submitting: boolean;
  onPackageStateChange: (value: ReturnPackageState) => void;
  onReasonChange: (value: string) => void;
  onCancel: () => void;
  onSubmit: () => void;
};

export default function ReturnRequestModal({
  open,
  packageState,
  reason,
  error,
  submitting,
  onPackageStateChange,
  onReasonChange,
  onCancel,
  onSubmit,
}: ReturnRequestModalProps) {
  const [agreedToReturnConditions, setAgreedToReturnConditions] =
    useState(false);

  useEffect(() => {
    if (!open) {
      setAgreedToReturnConditions(false);
    }
  }, [open]);

  useEffect(() => {
    if (packageState !== "unopened") {
      setAgreedToReturnConditions(false);
    }
  }, [packageState]);

  const normalizedReason = reason.trim();

  const canSubmit =
    packageState === "unopened"
      ? agreedToReturnConditions &&
        normalizedReason.length > 0 &&
        !submitting
      : packageState === "opened"
        ? normalizedReason.length > 0 && !submitting
        : false;

  const handleSubmit = () => {
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
      ariaLabelledBy="order-detail-return-modal-title"
      ariaDescribedBy={
        packageState === "unopened"
          ? "order-detail-return-modal-conditions"
          : undefined
      }
      ariaBusy={submitting}
    >
      <ModalHeader onClose={handleClose}>
        <ModalTitle id="order-detail-return-modal-title">
          返品を申請する
        </ModalTitle>
      </ModalHeader>

      <ModalBody className="order-detail-page__return-modal-body">
        <section
          className="order-detail-page__return-package-section"
          aria-labelledby="order-detail-return-package-label"
        >
          <ModalDescription id="order-detail-return-package-label">
            商品の包装紙は開封されていますか？
          </ModalDescription>

          <div
            className="order-detail-page__return-package-options"
            role="group"
            aria-labelledby="order-detail-return-package-label"
          >
            <Chip
              selected={packageState === "unopened"}
              disabled={submitting}
              onClick={() => onPackageStateChange("unopened")}
            >
              開封前
            </Chip>

            <Chip
              selected={packageState === "opened"}
              disabled={submitting}
              onClick={() => onPackageStateChange("opened")}
            >
              開封済
            </Chip>
          </div>
        </section>

        {packageState === "unopened" ? (
          <>
            <Alert
              id="order-detail-return-modal-conditions"
              variant="warning"
              className="order-detail-page__return-conditions"
            >
              <h3 className="order-detail-page__return-conditions-title">
                返品条件
              </h3>

              <ol className="order-detail-page__return-condition-list">
                <li>
                  返品が承認された場合、返金対象は商品代金（税込）のみです。
                </li>
                <li>
                  商品代金（税込）には、商品本体価格とその商品にかかる消費税が含まれます。
                </li>
                <li>
                  ご購入時の配送料および配送料にかかる消費税は返金対象外です。
                </li>
                <li>
                  返品商品の返送にかかる配送料はお客様のご負担となります。
                </li>
                <li>
                  返品手続き中は、商品が入っている配送用梱包材を開けないでください。
                </li>
              </ol>
            </Alert>

            <Checkbox
              label="返品条件に合意する"
              checked={agreedToReturnConditions}
              disabled={submitting}
              onChange={(event) => {
                setAgreedToReturnConditions(event.target.checked);
              }}
            />
          </>
        ) : null}

        {packageState !== null ? (
          <Textbox
            id="order-detail-return-reason"
            label="返品理由"
            value={reason}
            rows={6}
            required
            disabled={submitting}
            placeholder="返品理由を入力してください"
            onChange={(event) => {
              onReasonChange(event.target.value);
            }}
          />
        ) : null}

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
          {submitting ? "申請中..." : "返品を申請する"}
        </Button>
      </ModalFooter>
    </Modal>
  );
}