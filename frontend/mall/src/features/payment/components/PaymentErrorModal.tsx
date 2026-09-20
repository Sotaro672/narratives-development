// frontend/mall/src/features/payment/components/PaymentErrorModal.tsx

import Button from "../../../components/ui/Button";
import Modal, {
  ModalBody,
  ModalDescription,
  ModalFooter,
  ModalHeader,
  ModalTitle,
} from "../../../components/ui/Modal";

type PaymentErrorModalProps = {
  message: string;
  onClose: () => void;
};

export function PaymentErrorModal({
  message,
  onClose,
}: PaymentErrorModalProps) {
  return (
    <Modal
      open={Boolean(message)}
      onClose={onClose}
      size="sm"
      ariaLabelledBy="payment-error-modal-title"
      ariaDescribedBy="payment-error-modal-description"
    >
      <ModalHeader>
        <ModalTitle id="payment-error-modal-title">
          注文または決済処理に失敗しました
        </ModalTitle>
      </ModalHeader>

      <ModalBody>
        <ModalDescription id="payment-error-modal-description">
          {message}
        </ModalDescription>
      </ModalBody>

      <ModalFooter>
        <Button
          type="button"
          variant="primary"
          fullWidth
          onClick={onClose}
        >
          閉じる
        </Button>
      </ModalFooter>
    </Modal>
  );
}