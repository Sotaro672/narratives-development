// frontend/mall/src/features/contact/hooks/useSettingsInquiryPage.ts

import {
  useEffect,
  useMemo,
  useState,
} from "react";
import { getAuth } from "firebase/auth";

import { fetchCurrentUserProfile } from "../../auth/api/userApi";
import type { ChatComposerConfig } from "../../shared/types/chatComposer";
import { useContactAttachments } from "./useContactAttachments";
import { useContactSubmit } from "./useContactSubmit";

export function useSettingsInquiryPage() {
  const auth = getAuth();
  const currentUser = auth.currentUser;

  const [userName, setUserName] = useState("");
  const [loadingUser, setLoadingUser] = useState(true);
  const [userError, setUserError] = useState<string | null>(null);

  const {
    attachments,
    setAttachments,
    setCarouselIndex,
    handleFilesAdd,
    handleRemoveFile,
    revokeAllAttachmentPreviewUrls,
  } = useContactAttachments();

  useEffect(() => {
    let active = true;

    const loadUser = async (): Promise<void> => {
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
          throw new Error("ユーザー名を確認できませんでした。");
        }

        setUserName(resolvedName);
      } catch (caught) {
        if (!active) {
          return;
        }

        setUserName("");
        setUserError(
          caught instanceof Error
            ? caught.message
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

  const files = useMemo(
    () => attachments.map((attachment) => attachment.file),
    [attachments],
  );

  const canSubmit =
    Boolean(currentUser?.email) &&
    Boolean(userName) &&
    Boolean(message.trim()) &&
    !loadingUser &&
    !userError &&
    !submitting;

  const composer = useMemo<ChatComposerConfig>(
    () => ({
      content: message,
      placeholder: "お問い合わせ内容を入力",
      files,
      error: null,
      submitting,
      canSubmit,
      disabled: loadingUser || Boolean(userError),
      submitLabel: "送信",
      submittingLabel: "送信中",
      maxLength: null,
      maxFiles: 10,
      accept: "image/*",
      onContentChange: setMessage,
      onFilesAdd: handleFilesAdd,
      onRemoveFile: handleRemoveFile,
      onSubmit: handleSubmit,
    }),
    [
      message,
      files,
      submitting,
      canSubmit,
      loadingUser,
      userError,
      setMessage,
      handleFilesAdd,
      handleRemoveFile,
      handleSubmit,
    ],
  );

  return {
    currentUser,
    userName,
    loadingUser,
    userError,
    composer,
    uploadingAttachments,
    uploadProgress,
    uploadFileProgress,
    uploadFileIndex,
    uploadFileCount,
  };
}