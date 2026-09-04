// frontend/admin/shell/src/pages/InquiryDetailPage.tsx
import { useLocation, useNavigate, useParams } from "react-router-dom";

import type { Contact } from "../features/contact/contactApi";
import Page from "../shared/ui/Page/Page";

type InquiryDetailLocationState = {
  contact?: Contact;
};

export default function InquiryDetailPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const { inquiryId } = useParams();

  const state = location.state as InquiryDetailLocationState | null;
  const contact = state?.contact;

  if (!contact || contact.id !== inquiryId) {
    return (
      <Page>
        <h1>問い合わせ詳細</h1>
        <p>問い合わせ情報を取得できませんでした。</p>
        <button type="button" onClick={() => navigate("/inquiries")}>
          問い合わせ一覧へ戻る
        </button>
      </Page>
    );
  }

  return (
    <Page>
      <div>
        <button type="button" onClick={() => navigate("/inquiries")}>
          問い合わせ一覧へ戻る
        </button>
      </div>

      <h1>問い合わせ詳細</h1>

      <dl>
        <dt>受信日時</dt>
        <dd>{formatCreatedAt(contact.createdAt)}</dd>

        <dt>名前</dt>
        <dd>{contact.name}</dd>

        <dt>会社名</dt>
        <dd>{contact.company || "-"}</dd>

        <dt>メールアドレス</dt>
        <dd>
          <a href={`mailto:${contact.email}`}>{contact.email}</a>
        </dd>

        <dt>ステータス</dt>
        <dd>{contact.status}</dd>

        <dt>送信元</dt>
        <dd>{contact.source || "-"}</dd>

        <dt>問い合わせ内容</dt>
        <dd style={{ whiteSpace: "pre-wrap", overflowWrap: "anywhere" }}>
          {contact.message}
        </dd>
      </dl>
    </Page>
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