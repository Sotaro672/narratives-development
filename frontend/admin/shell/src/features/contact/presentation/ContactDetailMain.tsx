// frontend/admin/shell/src/features/contact/presentation/ContactDetailMain.tsx
import type { Contact } from "../infrastructure/contactApi";
import { useContactMessage } from "../hooks/useContactMessage";

type ContactDetailMainProps = {
  contact: Contact;
};

export default function ContactDetailMain({
  contact,
}: ContactDetailMainProps) {
  const {
    message,
    attachments,
    attachmentImages,
    attachmentsLoading,
    attachmentError,
  } = useContactMessage(contact.message);

  return (
    <>
      <section className="ui-detail-section">
        <h2 className="ui-detail-section__title">
          問い合わせ内容
        </h2>
        <p className="ui-detail-section__text">
          {message}
        </p>
      </section>

      {attachments.length > 0 && (
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
                {attachmentImages.map((attachment) => (
                  <figure
                    key={attachment.storagePath}
                    className="ui-detail-attachment"
                  >
                    <img
                      src={attachment.imageUrl}
                      alt={attachment.fileName}
                      className="ui-detail-attachment__image"
                    />
                    <figcaption className="ui-detail-attachment__caption">
                      {attachment.fileName}
                    </figcaption>
                  </figure>
                ))}
              </div>
            )}
        </section>
      )}
    </>
  );
}