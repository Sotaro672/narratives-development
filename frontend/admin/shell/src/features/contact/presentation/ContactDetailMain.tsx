// frontend/admin/shell/src/features/contact/presentation/ContactDetailMain.tsx
import { useEffect, useState } from "react";

import type {
  Contact,
  ContactAttachmentImage,
} from "../../../shared/type/contact";
import { loadContactAttachmentImages } from "../infrastructure/contactAttachmentStorage";

type ContactDetailMainProps = {
  contact: Contact;
};

export default function ContactDetailMain({
  contact,
}: ContactDetailMainProps) {
  const [attachmentImages, setAttachmentImages] =
    useState<ContactAttachmentImage[]>([]);
  const [attachmentsLoading, setAttachmentsLoading] = useState(false);
  const [attachmentError, setAttachmentError] =
    useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    if (contact.attachmentImageIds.length === 0) {
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
          contact.attachmentImageIds,
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
  }, [contact.attachmentImageIds]);

  return (
    <>
      <section className="ui-detail-section">
        <h2 className="ui-detail-section__title">
          問い合わせ内容
        </h2>

        <p className="ui-detail-section__text">
          {contact.message}
        </p>
      </section>

      {contact.attachmentImageIds.length > 0 && (
        <section className="ui-detail-section">
          <h2 className="ui-detail-section__title">
            添付ファイル
          </h2>

          {attachmentsLoading && (
            <p>添付画像を読み込んでいます。</p>
          )}

          {!attachmentsLoading && attachmentError && (
            <p role="alert">{attachmentError}</p>
          )}

          {!attachmentsLoading &&
            !attachmentError &&
            attachmentImages.length > 0 && (
              <div className="ui-detail-attachments">
                {attachmentImages.map((attachment, index) => (
                  <figure
                    key={attachment.imageId}
                    className="ui-detail-attachment"
                  >
                    <img
                      src={attachment.imageUrl}
                      alt={`添付画像 ${index + 1}`}
                      className="ui-detail-attachment__image"
                    />
                  </figure>
                ))}
              </div>
            )}
        </section>
      )}
    </>
  );
}