// frontend/src/features/contact/hooks/useContactSubmit.ts
import { Dispatch, SetStateAction, useState } from "react";
import type { User } from "firebase/auth";

import type { ContactAttachmentItem } from "../types";
import { uploadContactAttachments } from "../utils/upload";

type UseContactSubmitParams = {
  currentUser: User | null;
  isLoggedIn: boolean;
  attachments: ContactAttachmentItem[];
  setAttachments: Dispatch<SetStateAction<ContactAttachmentItem[]>>;
  setCarouselIndex: Dispatch<SetStateAction<number>>;
  revokeAllAttachmentPreviewUrls: () => void;
};

type ContactErrorResponse = {
  error?: string;
  status?: string;
};

function getBackendUrl() {
  const backendUrl =
    import.meta.env.VITE_API_BASE_URL ?? "";

  if (!backendUrl) {
    throw new Error(
      "VITE_API_BASE_URLが設定されていません。"
    );
  }

  return backendUrl.endsWith("/") ? backendUrl.slice(0, -1) : backendUrl;
}

async function readJsonSafe(
  response: Response
): Promise<ContactErrorResponse | null> {
  const contentType = response.headers.get("content-type") || "";

  if (!contentType.includes("application/json")) {
    return null;
  }

  return response.json();
}

export function useContactSubmit({
  currentUser,
  isLoggedIn,
  attachments,
  setAttachments,
  setCarouselIndex,
  revokeAllAttachmentPreviewUrls,
}: UseContactSubmitParams) {
  const [name, setName] = useState("");
  const [guestEmail, setGuestEmail] = useState("");
  const [company, setCompany] = useState("");
  const [message, setMessage] = useState("");
  const [submitting, setSubmitting] = useState(false);

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
    if (submitting) {
      return;
    }

    const trimmedName = name.trim();
    const trimmedGuestEmail = guestEmail.trim();
    const trimmedCompany = company.trim();
    const trimmedMessage = message.trim();

    const contactEmail = isLoggedIn
      ? currentUser?.email?.trim() ?? ""
      : trimmedGuestEmail;

    if (trimmedName === "") {
      window.alert("お名前を入力してください。");
      return;
    }

    if (contactEmail === "") {
      window.alert("メールアドレスを入力してください。");
      return;
    }

    if (trimmedMessage === "") {
      window.alert("お問い合わせ内容を入力してください。");
      return;
    }

    try {
      setSubmitting(true);

      const backendUrl = getBackendUrl();

      const uploadedAttachments = await uploadContactAttachments({
        attachments,
        ownerId: currentUser?.uid ?? "guest",
      });

      const attachmentText =
        uploadedAttachments.length > 0
          ? `\n\n--- 添付ファイル ---\n${uploadedAttachments
              .map((item, index) =>
                [
                  `${index + 1}. ${item.fileName}`,
                  `URL: ${item.downloadUrl}`,
                  `Content Type: ${item.contentType}`,
                  `Size: ${item.size}`,
                ].join("\n")
              )
              .join("\n\n")}`
          : "";

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
          name: trimmedName,
          email: contactEmail,
          company: trimmedCompany,
          message: `${trimmedMessage}${attachmentText}`,
          source: "web-amol",
        }),
      });

      const responseBody = await readJsonSafe(response);

      if (!response.ok) {
        throw new Error(
          responseBody?.error || "お問い合わせの送信に失敗しました。"
        );
      }

      resetForm();
      window.alert("お問い合わせを受け付けました。");
    } catch (error) {
      console.error(error);

      if (error instanceof Error) {
        window.alert(error.message);
      } else {
        window.alert("お問い合わせの送信に失敗しました。");
      }
    } finally {
      setSubmitting(false);
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
    handleSubmit,
  };
}