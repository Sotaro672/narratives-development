// frontend/admin/shell/src/pages/InquiryDetailPage.tsx
import { useNavigate, useParams } from "react-router-dom";

import { useContactDetail } from "../features/contact/hooks/useContactDetail";
import ContactDetailAside from "../features/contact/presentation/ContactDetailAside";
import ContactDetailMain from "../features/contact/presentation/ContactDetailMain";
import Page, {
  DetailPageBody,
  PageHeader,
} from "../shared/ui/Page/Page";

export default function InquiryDetailPage() {
  const navigate = useNavigate();
  const { inquiryId } = useParams();
  const { contact, loading, error } = useContactDetail(inquiryId);

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

      {loading && <p>問い合わせ情報を読み込んでいます。</p>}

      {!loading && error && (
        <p role="alert">
          問い合わせ情報を取得できませんでした。{error}
        </p>
      )}

      {!loading && !error && !contact && (
        <p role="alert">問い合わせ情報を取得できませんでした。</p>
      )}

      {!loading && !error && contact && (
        <DetailPageBody
          main={<ContactDetailMain contact={contact} />}
          aside={<ContactDetailAside contact={contact} />}
        />
      )}
    </Page>
  );
}