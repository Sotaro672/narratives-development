// frontend/admin/shell/src/features/contact/hooks/useContactMessage.ts
import { useEffect, useState } from "react";

import type { ContactAttachmentImage } from "../../../shared/type/contact";
import { loadContactAttachmentImages } from "../infrastructure/contactAttachmentStorage";

export function useContactMessage(
  message: string,
  attachmentImageIds: string[],
) {
  const [attachmentImages, setAttachmentImages] =
    useState<ContactAttachmentImage[]>([]);
  const [attachmentsLoading, setAttachmentsLoading] = useState(false);
  const [attachmentError, setAttachmentError] =
    useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    if (attachmentImageIds.length === 0) {
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
          attachmentImageIds,
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
  }, [attachmentImageIds]);

  return {
    message,
    attachmentImages,
    attachmentsLoading,
    attachmentError,
  };
}