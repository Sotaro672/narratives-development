// frontend/mall/src/features/scan-result/presentation/components/ScanTransferConfirmModal.tsx

import Button from "../../../../components/ui/Button";
import Modal, {
  ModalBody,
  ModalFooter,
  ModalHeader,
  ModalTitle,
} from "../../../../components/ui/Modal";
import TextState from "../../../../components/ui/TextState";

type Props = {
  open: boolean;
  loading: boolean;
  error?: string | null;
  onCancel: () => void;
  onConfirm: () => void | Promise<void>;
};

export default function ScanTransferConfirmModal({
  open,
  loading,
  error = null,
  onCancel,
  onConfirm,
}: Props) {
  const handleConfirm = async () => {
    if (loading) {
      return;
    }

    await onConfirm();
  };

  return (
    <Modal
      open={open}
      onClose={loading ? undefined : onCancel}
      size="md"
      mobilePosition="bottom"
      closeOnBackdrop={!loading}
      closeOnEscape={!loading}
      ariaLabelledBy="scan-transfer-confirm-modal-title"
      ariaBusy={loading}
    >
      <ModalHeader onClose={loading ? undefined : onCancel}>
        <ModalTitle id="scan-transfer-confirm-modal-title">
          トークン移譲の確認
        </ModalTitle>
      </ModalHeader>

      <ModalBody>
        <TextState>
          トークン移譲後は返品ができません。よろしければ承諾ボタンを押下してください。トークンが移譲されます。
        </TextState>

        {error ? (
          <TextState variant="error">
            {error}
          </TextState>
        ) : null}
      </ModalBody>

      <ModalFooter className="scan-transfer-modal__footer">
        <Button
          type="button"
          variant="secondary"
          fullWidth
          disabled={loading}
          onClick={onCancel}
        >
          キャンセル
        </Button>

        <Button
          type="button"
          fullWidth
          disabled={loading}
          onClick={() => {
            void handleConfirm();
          }}
        >
          {loading ? "移譲中..." : "承諾"}
        </Button>
      </ModalFooter>
    </Modal>
  );
}