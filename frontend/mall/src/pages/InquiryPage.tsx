// frontend/amol/src/pages/InquiryPage.tsx

import { getAuth } from "firebase/auth";

import "../styles/page-layout.css";
import "../styles/form.css";
import "../styles/contact-page.css";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";
import Input from "../components/ui/Input";
import MediaUploader from "../components/ui/MediaUploader";
import Textbox from "../components/ui/Textbox";
import ContactUploadProgressModal from "../features/contact/components/ContactUploadProgressModal";
import { useContactAttachments } from "../features/contact/hooks/useContactAttachments";
import { useContactSubmit } from "../features/contact/hooks/useContactSubmit";

export default function InquiryPage() {
  const auth = getAuth();
  const currentUser = auth.currentUser;

  const {
    mediaInputRef,
    carouselRef,
    carouselIndex,
    attachments,
    setAttachments,
    setCarouselIndex,
    handleFilesSelected,
    handleRemoveAttachment,
    handleCarouselScroll,
    handleMoveToSlide,
    revokeAllAttachmentPreviewUrls,
  } = useContactAttachments();

  const {
    name,
    setName,
    message,
    setMessage,
    submitting,
    uploadingAttachments,
    uploadProgress,
    uploadFileProgress,
    uploadFileIndex,
    uploadFileCount,
    handleSubmit,
  } = useContactSubmit({
    currentUser,
    isLoggedIn: Boolean(currentUser),
    attachments,
    setAttachments,
    setCarouselIndex,
    revokeAllAttachmentPreviewUrls,
  });

  const canSubmit =
    Boolean(currentUser?.email) &&
    Boolean(name.trim()) &&
    Boolean(message.trim()) &&
    !submitting;

  return (
    <>
      <Layout
        title="問い合わせ"
        titleClickable={false}
        showBackButton
        mode="default"
        backTo="/lists"
        hideHamburgerMenu
        hideSettingsButton
      >
        <section className="page-section content-page-section">
          <p className="page-description">
            AMOLへのお問い合わせ内容を入力してください。
          </p>

          <div className="form-block">
            <Input
              id="inquiry-name"
              name="name"
              label="お名前"
              value={name}
              placeholder="お名前を入力してください"
              disabled={submitting}
              required
              onChange={(event) => setName(event.target.value)}
            />

            <Textbox
              id="inquiry-message"
              label="お問い合わせ内容"
              value={message}
              placeholder="お問い合わせ内容を入力してください"
              rows={8}
              disabled={submitting}
              required
              onChange={(event) => setMessage(event.target.value)}
            />

            <MediaUploader
              label="添付ファイル画像"
              hint="アップロードできるのは画像のみです。"
              emptyText="添付ファイルはまだ選択されていません。"
              accept="image/*"
              multiple
              items={attachments}
              currentIndex={carouselIndex}
              inputRef={mediaInputRef}
              carouselRef={carouselRef}
              onFilesSelected={handleFilesSelected}
              onRemoveItem={handleRemoveAttachment}
              onCarouselScroll={handleCarouselScroll}
              onMoveToSlide={handleMoveToSlide}
              selectButtonLabel="ファイルを選択"
              disabled={submitting}
            />
          </div>

          <p className="page-description">
            お問い合わせ内容によっては、ご回答までにお時間をいただく場合があります。
          </p>

          <div className="page-actions">
            <Button
              variant="primary"
              size="md"
              disabled={!canSubmit}
              onClick={() => void handleSubmit()}
            >
              {submitting ? "送信中..." : "問い合わせを送信"}
            </Button>
          </div>
        </section>
      </Layout>

      <ContactUploadProgressModal
        open={uploadingAttachments}
        progress={uploadProgress}
        fileProgress={uploadFileProgress}
        fileIndex={uploadFileIndex}
        fileCount={uploadFileCount}
      />
    </>
  );
}