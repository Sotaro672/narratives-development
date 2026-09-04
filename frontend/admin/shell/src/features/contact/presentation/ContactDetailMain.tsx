// frontend/admin/shell/src/features/contact/presentation/ContactDetailMain.tsx
import { useEffect, useState } from "react";

import type {
  Contact,
  ContactAttachmentImage,
} from "../../../shared/type/contact";
import MediaGallery, {
  type MediaGalleryItem,
} from "../../../shared/ui/MediaGallery/MediaGallery";
import {
  downloadContactAttachment,
  loadContactAttachmentImages,
} from "../infrastructure/contactAttachmentStorage";

type ContactDetailMainProps = {
  contact: Contact;
};

export default function ContactDetailMain({
  contact,
}: ContactDetailMainProps) {
  const attachmentImageIds = contact.attachmentImageIds ?? [];
  const [attachmentImages, setAttachmentImages] =
    useState<ContactAttachmentImage[]>([]);
  const [attachmentsLoading, setAttachmentsLoading] = useState(false);
  const [attachmentError, setAttachmentError] = useState<string | null>(null);

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
        const images = await loadContactAttachmentImages(attachmentImageIds);

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
  }, [contact.attachmentImageIds]);

  const mediaItems: MediaGalleryItem[] = attachmentImages.map((attachment) => ({
    id: attachment.imageId,
    url: attachment.imageUrl,
    fileName: attachment.fileName,
  }));

  const handleDownload = async (item: MediaGalleryItem) => {
    await downloadContactAttachment(item.id, item.fileName);
  };

  return (
    <>
      <section className="ui-detail-section">
        <h2 className="ui-detail-section__title">問い合わせ内容</h2>
        <p className="ui-detail-section__text">{contact.message}</p>
      </section>

      {attachmentImageIds.length > 0 && (
        <section className="ui-detail-section">
          <h2 className="ui-detail-section__title">添付ファイル</h2>

          {attachmentsLoading && <p>添付画像を読み込んでいます。</p>}

          {!attachmentsLoading && attachmentError && (
            <p role="alert">{attachmentError}</p>
          )}

          {!attachmentsLoading &&
            !attachmentError &&
            mediaItems.length > 0 && (
              <MediaGallery
                items={mediaItems}
                altFallback="問い合わせ添付画像"
                onDownload={handleDownload}
              />
            )}
        </section>
      )}
    </>
  );
}