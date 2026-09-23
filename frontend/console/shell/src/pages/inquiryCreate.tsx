// frontend/console/shell/src/pages/inquiryCreate.tsx

import InquiryCreateForm from "../features/inquiry/presentation/components/inquiryCreateForm";
import { useInquiryCreate } from "../features/inquiry/presentation/hooks/useInquiryCreate";
import PageStyle from "../layout/PageStyle/PageStyle";

export default function InquiryCreate() {
  const {
    message,
    attachments,
    submitting,
    uploadProgress,
    errorMessage,
    canSubmit,
    maxMessageLength,
    maxImages,
    maxImageSizeMB,
    handleMessageChange,
    handleFilesSelected,
    handleRemoveAttachment,
    handleBack,
    handleSubmit,
  } = useInquiryCreate();

  return (
    <PageStyle
      layout="single"
      title="AMOLに問い合わせ"
      onBack={handleBack}
      onSave={undefined}
      onSend={handleSubmit}
      isSending={submitting}
      sendDisabled={!canSubmit}
    >
      <div className="mx-auto w-full max-w-4xl">
        <InquiryCreateForm
          message={message}
          attachments={attachments}
          submitting={submitting}
          uploadProgress={uploadProgress}
          errorMessage={errorMessage}
          maxMessageLength={maxMessageLength}
          maxImages={maxImages}
          maxImageSizeMB={maxImageSizeMB}
          onChangeMessage={handleMessageChange}
          onChangeFiles={handleFilesSelected}
          onRemoveAttachment={handleRemoveAttachment}
        />
      </div>
    </PageStyle>
  );
}