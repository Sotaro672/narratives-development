// frontend/mall/src/features/contact/hooks/useContactSubmit.ts

import { Dispatch, SetStateAction, useRef, useState } from "react";
import type { User } from "firebase/auth";

import type { ContactAttachmentItem } from "../../shared/types/contact";
import { uploadContactAttachments } from "../utils/upload";

type UseContactSubmitParams = {
  currentUser: User | null;
  isLoggedIn: boolean;
  attachments: ContactAttachmentItem[];
  setAttachments: Dispatch<SetStateAction<ContactAttachmentItem[]>>;
  setCarouselIndex: Dispatch<SetStateAction<number>>;
  revokeAllAttachmentPreviewUrls: () => void;
  source?: string;
  nameOverride?: string;
  companyOverride?: string;
};

type ContactErrorResponse = {
  error?: string;
  status?: string;
};

function getBackendUrl(): string {
  const backendUrl = import.meta.env.VITE_API_BASE_URL ?? "";

  if (!backendUrl) {
    throw new Error("VITE_API_BASE_URLが設定されていません。");
  }

  return backendUrl.endsWith("/") ? backendUrl.slice(0, -1) : backendUrl;
}

async function readJsonSafe(
  response: Response,
): Promise<ContactErrorResponse | null> {
  const contentType = response.headers.get("content-type") || "";

  if (!contentType.includes("application/json")) {
    return null;
  }

  try {
    return await response.json();
  } catch {
    return null;
  }
}

export function useContactSubmit({
  currentUser,
  isLoggedIn,
  attachments,
  setAttachments,
  setCarouselIndex,
  revokeAllAttachmentPreviewUrls,
  source = "web-amol",
  nameOverride,
  companyOverride,
}: UseContactSubmitParams) {
  const submittingRef = useRef(false);
  const [name, setName] = useState("");
  const [guestEmail, setGuestEmail] = useState("");
  const [company, setCompany] = useState("");
  const [message, setMessage] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [uploadingAttachments, setUploadingAttachments] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [uploadFileProgress, setUploadFileProgress] = useState(0);
  const [uploadFileIndex, setUploadFileIndex] = useState(0);
  const [uploadFileCount, setUploadFileCount] = useState(0);

  const resetUploadProgress = () => {
    setUploadingAttachments(false);
    setUploadProgress(0);
    setUploadFileProgress(0);
    setUploadFileIndex(0);
    setUploadFileCount(0);
  };

  const resetForm = () => {
    setName("");
    setGuestEmail("");
    setCompany("");
    setMessage("");
    revokeAllAttachmentPreviewUrls();
    setAttachments([]);
    setCarouselIndex(0);
  };

  const handleSubmit = async () => {
    if (submittingRef.current) {
      return;
    }

    const contactName = (nameOverride ?? name).trim();
    const contactCompany = (companyOverride ?? company).trim();
    const contactEmail = isLoggedIn
      ? currentUser?.email?.trim() ?? ""
      : guestEmail.trim();
    const contactMessage = message.trim();

    if (contactName === "") {
      window.alert(
        nameOverride !== undefined
          ? "お名前を確認できませんでした。"
          : "お名前を入力してください。",
      );
      return;
    }

    if (contactEmail === "") {
      window.alert("メールアドレスを入力してください。");
      return;
    }

    if (contactMessage === "") {
      window.alert("お問い合わせ内容を入力してください。");
      return;
    }

    submittingRef.current = true;
    setSubmitting(true);

    if (attachments.length > 0) {
      setUploadProgress(0);
      setUploadFileProgress(0);
      setUploadFileIndex(1);
      setUploadFileCount(attachments.length);
      setUploadingAttachments(true);
    }

    try {
      const backendUrl = getBackendUrl();

      const uploadedAttachments = await uploadContactAttachments({
        attachments,
        onProgress: ({
          fileIndex,
          fileCount,
          fileProgress,
          totalProgress,
        }) => {
          setUploadFileIndex(fileIndex);
          setUploadFileCount(fileCount);
          setUploadFileProgress(fileProgress);
          setUploadProgress(totalProgress);
        },
      });

      setUploadingAttachments(false);

      const attachmentImageIds = uploadedAttachments.map(
        (attachment) => attachment.imageId,
      );

      const headers: Record<string, string> = {
        "Content-Type": "application/json",
        Accept: "application/json",
      };

      if (currentUser) {
        const idToken = await currentUser.getIdToken();
        headers.Authorization = `Bearer ${idToken}`;
      }

      const response = await fetch(`${backendUrl}/introduction/contacts`, {
        method: "POST",
        headers,
        body: JSON.stringify({
          name: contactName,
          email: contactEmail,
          company: contactCompany,
          message: contactMessage,
          attachmentImageIds,
          source,
        }),
      });

      const responseBody = await readJsonSafe(response);

      if (!response.ok) {
        throw new Error(
          responseBody?.error || "お問い合わせの送信に失敗しました。",
        );
      }

      resetForm();
      resetUploadProgress();
      window.alert("お問い合わせを受け付けました。");
    } catch (error) {
      resetUploadProgress();

      window.alert(
        error instanceof Error
          ? error.message
          : "お問い合わせの送信に失敗しました。",
      );
    } finally {
      submittingRef.current = false;
      setSubmitting(false);
      setUploadingAttachments(false);
    }
  };

  return {
    name,
    setName,
    guestEmail,
    setGuestEmail,
    company,
    setCompany,
    message,
    setMessage,
    submitting,
    uploadingAttachments,
    uploadProgress,
    uploadFileProgress,
    uploadFileIndex,
    uploadFileCount,
    handleSubmit,
  };
}