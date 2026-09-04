// frontend/admin/shell/src/features/contact/hooks/useContactMessage.ts
import { useEffect, useMemo, useState } from "react";

import { parseContactMessage } from "../application/contactMessage";
import {
  loadContactAttachmentImages,
  type ContactAttachmentImage,
} from "../infrastructure/contactAttachmentStorage";

export function useContactMessage(message: string) {
  const parsed = useMemo(
    () => parseContactMessage(message),
    [message],
  );

  const [attachmentImages, setAttachmentImages] =
    useState<ContactAttachmentImage[]>([]);
  const [attachmentsLoading, setAttachmentsLoading] = useState(false);
  const [attachmentError, setAttachmentError] =
    useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    if (parsed.attachments.length === 0) {
      setAttachmentImages([]);
      setAttachmentsLoading(false);
      setAttachmentError(null);
      return;
    }

    const load = async () => {
      setAttachmentsLoading(true);
      setAttachmentError(null);

      try {
        const images = await loadContactAttachmentImages(
          parsed.attachments,
        );

        if (!cancelled) {
          setAttachmentImages(images);
        }
      } catch {
        if (!cancelled) {
          setAttachmentImages([]);
          setAttachmentError("添付画像の取得に失敗しました。");
        }
      } finally {
        if (!cancelled) {
          setAttachmentsLoading(false);
        }
      }
    };

    void load();

    return () => {
      cancelled = true;
    };
  }, [parsed.attachments]);

  return {
    message: parsed.message,
    attachments: parsed.attachments,
    attachmentImages,
    attachmentsLoading,
    attachmentError,
  };
}