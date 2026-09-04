// frontend/admin/shell/src/pages/InquiryDetailPage.tsx
import { useLocation, useNavigate, useParams } from "react-router-dom";

import type { Contact } from "../features/contact/infrastructure/contactApi";
import Page, {
  DetailPageBody,
  PageHeader,
} from "../shared/ui/Page/Page";

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
        <PageHeader
          title="問い合わせ詳細"
          leading={
            <button type="button" onClick={() => navigate("/inquiries")}>
              戻る
            </button>
          }
        />

        <p>問い合わせ情報を取得できませんでした。</p>
      </Page>
    );
  }

  return (
    <Page>
      <PageHeader
        title="問い合わせ詳細"
        leading={
          <button type="button" onClick={() => navigate("/inquiries")}>
            戻る
          </button>
        }
      />

      <DetailPageBody
        main={
          <>
            <section>
              <h2>問い合わせ内容</h2>
              <p
                style={{
                  whiteSpace: "pre-wrap",
                  overflowWrap: "anywhere",
                }}
              >
                {contact.message}
              </p>
            </section>

            <section>
              <h2>送信者情報</h2>

              <dl>
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
        }
        aside={
          <>
            <section>
              <h2>管理情報</h2>

              <dl>
                <dt>ステータス</dt>
                <dd>{contact.status}</dd>

                <dt>受信日時</dt>
                <dd>{formatCreatedAt(contact.createdAt)}</dd>

                <dt>送信元</dt>
                <dd>{contact.source || "-"}</dd>
              </dl>
            </section>

            <section>
              <h2>問い合わせID</h2>
              <p>{contact.id}</p>
            </section>
          </>
        }
      />
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