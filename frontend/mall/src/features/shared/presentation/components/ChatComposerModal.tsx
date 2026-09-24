// frontend/amol/src/features/shared/presentation/components/ChatComposerModal.tsx

import { useId, type ReactNode } from "react";

import Alert from "../../../../components/ui/Alert";
import Button from "../../../../components/ui/Button";
import Modal, {
  ModalBody,
  ModalDescription,
  ModalFooter,
  ModalHeader,
  ModalTitle,
} from "../../../../components/ui/Modal";
import Textbox from "../../../../components/ui/Textbox";

export type ChatComposerModalProps = {
  open: boolean;
  title: string;
  content: string;
  placeholder: string;
  error?: string | null;
  submitting: boolean;
  canSubmit: boolean;
  submitLabel: string;
  submittingLabel: string;
  onContentChange: (value: string) => void;
  onCancel: () => void;
  onSubmit: () => void;
  description?: ReactNode;
  afterInput?: ReactNode;
  inputAriaLabel?: string;
  rows?: number;
  maxLength?: number | null;
};

export default function ChatComposerModal({
  open,
  title,
  content,
  placeholder,
  error,
  submitting,
  canSubmit,
  submitLabel,
  submittingLabel,
  onContentChange,
  onCancel,
  onSubmit,
  description,
  afterInput,
  inputAriaLabel,
  rows = 6,
  maxLength = 500,
}: ChatComposerModalProps) {
  const titleId = useId();
  const descriptionId = useId();
  const closeHandler = submitting ? undefined : onCancel;

  return (
    <Modal
      open={open}
      onClose={closeHandler}
      size="md"
      mobilePosition="bottom"
      closeOnBackdrop={!submitting}
      closeOnEscape={!submitting}
      ariaLabelledBy={titleId}
      ariaDescribedBy={description ? descriptionId : undefined}
      ariaBusy={submitting}
    >
      <ModalHeader onClose={closeHandler} closeLabel="閉じる">
        <ModalTitle id={titleId}>{title}</ModalTitle>
      </ModalHeader>

      <ModalBody>
        {description ? (
          <ModalDescription id={descriptionId}>
            {description}
          </ModalDescription>
        ) : null}

        <Textbox
          value={content}
          placeholder={placeholder}
          aria-label={inputAriaLabel ?? placeholder}
          rows={rows}
          maxLength={maxLength ?? undefined}
          disabled={submitting}
          onChange={(event) => onContentChange(event.currentTarget.value)}
        />

        {afterInput}

        {error ? (
          <Alert variant="error">
            {error}
          </Alert>
        ) : null}
      </ModalBody>

      <ModalFooter>
        <Button
          variant="primary"
          size="md"
          disabled={!canSubmit || submitting}
          aria-busy={submitting || undefined}
          onClick={onSubmit}
        >
          {submitting ? submittingLabel : submitLabel}
        </Button>
      </ModalFooter>
    </Modal>
  );
}