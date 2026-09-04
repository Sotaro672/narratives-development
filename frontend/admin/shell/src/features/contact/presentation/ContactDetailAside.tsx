// frontend/admin/shell/src/features/contact/presentation/ContactDetailAside.tsx
import type { Contact } from "../infrastructure/contactApi";

type ContactDetailAsideProps = {
  contact: Contact;
};

export default function ContactDetailAside({
  contact,
}: ContactDetailAsideProps) {
  return (
    <>
      <section className="ui-detail-section">
        <h2 className="ui-detail-section__title">
          管理情報
        </h2>
        <dl className="ui-detail-definition-list">
          <dt>ステータス</dt>
          <dd>{contact.status}</dd>

          <dt>受信日時</dt>
          <dd>{formatCreatedAt(contact.createdAt)}</dd>

          <dt>送信元</dt>
          <dd>{contact.source || "-"}</dd>
        </dl>
      </section>

      <section className="ui-detail-section">
        <h2 className="ui-detail-section__title">
          送信者情報
        </h2>
        <dl className="ui-detail-definition-list">
          <dt>名前</dt>
          <dd>{contact.name}</dd>

          <dt>会社名</dt>
          <dd>{contact.company || "-"}</dd>

          <dt>メールアドレス</dt>
          <dd>
            <a href={`mailto:${contact.email}`}>
              {contact.email}
            </a>
          </dd>
        </dl>
      </section>
    </>
  );
}

function formatCreatedAt(value: string): string {
  if (!value) {
    return "-";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString("ja-JP");
}