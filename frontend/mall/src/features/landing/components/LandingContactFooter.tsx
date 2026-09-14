// frontend/mall/src/features/landing/components/LandingContactFooter.tsx

import FooterNav from "../../../components/layout/FooterNav";
import ContactUploadProgressModal from "../../contact/components/ContactUploadProgressModal";

type ContactUploadProgressModalProps =
  Parameters<typeof ContactUploadProgressModal>[0];

type LandingContactFooterProps = {
  isDesktop: boolean;
  isContactSectionVisible: boolean;
  shouldShowFooterNav: boolean;
  submitButtonLabel: string;
  submitting: boolean;
  uploadingAttachments: ContactUploadProgressModalProps["open"];
  uploadProgress: ContactUploadProgressModalProps["progress"];
  uploadFileProgress: ContactUploadProgressModalProps["fileProgress"];
  uploadFileIndex: ContactUploadProgressModalProps["fileIndex"];
  uploadFileCount: ContactUploadProgressModalProps["fileCount"];
  onSubmit: () => void;
};

export default function LandingContactFooter({
  isDesktop,
  isContactSectionVisible,
  shouldShowFooterNav,
  submitButtonLabel,
  submitting,
  uploadingAttachments,
  uploadProgress,
  uploadFileProgress,
  uploadFileIndex,
  uploadFileCount,
  onSubmit,
}: LandingContactFooterProps) {
  return (
    <>
      <ContactUploadProgressModal
        open={uploadingAttachments}
        progress={uploadProgress}
        fileProgress={uploadFileProgress}
        fileIndex={uploadFileIndex}
        fileCount={uploadFileCount}
      />

      {!isDesktop && isContactSectionVisible ? (
        <FooterNav
          variant="action"
          buttonLabel={submitButtonLabel}
          disabled={submitting}
          onButtonClick={onSubmit}
        />
      ) : shouldShowFooterNav ? (
        <FooterNav />
      ) : null}
    </>
  );
}