// frontend/amol/src/pages/InquiryPage.tsx

import { useEffect, useMemo, useState } from "react";
import { getAuth } from "firebase/auth";

import "../styles/page-layout.css";
import "../styles/form.css";
import "../styles/contact-page.css";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";
import MediaUploader from "../components/ui/MediaUploader";
import Textbox from "../components/ui/Textbox";
import { fetchCurrentUserProfile } from "../features/auth/api/userApi";
import ContactUploadProgressModal from "../features/contact/components/ContactUploadProgressModal";
import { useContactAttachments } from "../features/contact/hooks/useContactAttachments";
import { useContactSubmit } from "../features/contact/hooks/useContactSubmit";

export default function InquiryPage() {
  const auth = getAuth();
  const currentUser = auth.currentUser;

  const [userName, setUserName] = useState("");
  const [loadingUser, setLoadingUser] = useState(true);
  const [userError, setUserError] = useState<string | null>(null);

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

  useEffect(() => {
    let active = true;

    const loadUser = async () => {
      setLoadingUser(true);
      setUserError(null);

      try {
        const profile = await fetchCurrentUserProfile();

        if (!active) {
          return;
        }

        const resolvedName = [
          profile.last_name?.trim(),
          profile.first_name?.trim(),
        ]
          .filter(Boolean)
          .join(" ");

        if (!resolvedName) {
          throw new Error(
            "ユーザー名を確認できませんでした。",
          );
        }

        setUserName(resolvedName);
      } catch (error) {
        if (!active) {
          return;
        }

        setUserName("");
        setUserError(
          error instanceof Error
            ? error.message
            : "ユーザー情報の取得に失敗しました。",
        );
      } finally {
        if (active) {
          setLoadingUser(false);
        }
      }
    };

    void loadUser();

    return () => {
      active = false;
    };
  }, []);

  const {
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
    source: "mall",
    nameOverride: userName,
    companyOverride: "-",
  });

  const canSubmit = useMemo(() => {
    return (
      Boolean(currentUser?.email) &&
      Boolean(userName) &&
      Boolean(message.trim()) &&
      !loadingUser &&
      !submitting
    );
  }, [
    currentUser?.email,
    userName,
    message,
    loadingUser,
    submitting,
  ]);

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

          {userError ? (
            <p className="page-description">
              {userError}
            </p>
          ) : null}

          <div className="form-block">
            <Textbox
              id="inquiry-message"
              label="お問い合わせ内容"
              value={message}
              placeholder="お問い合わせ内容を入力してください"
              rows={8}
              disabled={submitting || loadingUser}
              required
              onChange={(event) =>
                setMessage(event.target.value)
              }
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
              disabled={submitting || loadingUser}
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
              {loadingUser
                ? "ユーザー情報確認中..."
                : submitting
                  ? "送信中..."
                  : "問い合わせを送信"}
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