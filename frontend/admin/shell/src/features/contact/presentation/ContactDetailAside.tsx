// frontend/admin/shell/src/features/contact/presentation/ContactDetailAside.tsx
import type { Contact } from "../../../shared/type/contact";

type ContactDetailAsideProps = {
  contact: Contact;
};

export default function ContactDetailAside({
  contact,
}: ContactDetailAsideProps) {
  return (
    <section className="ui-detail-section">
      <h2 className="ui-detail-section__title">送信者情報</h2>
      <dl className="ui-detail-definition-list">
        <dt>名前</dt>
        <dd>{contact.name}</dd>

        <dt>会社名</dt>
        <dd>{contact.company || "-"}</dd>

        <dt>メールアドレス</dt>
        <dd>
          <a href={`mailto:${contact.email}`}>{contact.email}</a>
        </dd>
      </dl>
    </section>
  );
}