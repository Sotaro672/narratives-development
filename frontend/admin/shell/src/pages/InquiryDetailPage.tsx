// frontend/admin/shell/src/pages/InquiryDetailPage.tsx
import { useNavigate, useParams } from "react-router-dom";

import { useContactDetail } from "../features/contact/hooks/useContactDetail";
import ContactDetailAside from "../features/contact/presentation/ContactDetailAside";
import ContactDetailMain from "../features/contact/presentation/ContactDetailMain";
import { getContactSourceLabel } from "../features/contact/presentation/model/contactSourcePresentation";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";
import { formatDateTime } from "../shared/util/dateFormat";

export default function InquiryDetailPage() {
  const navigate = useNavigate();
  const { inquiryId } = useParams();
  const { contact, loading, error } = useContactDetail(inquiryId);

  const pageTitle = contact
    ? getContactSourceLabel(contact.source)
    : "問い合わせ詳細";

  return (
    <Page>
      <PageHeader
        title={pageTitle}
        meta={contact ? formatDateTime(contact.createdAt) : undefined}
        leading={
          <button
            type="button"
            className="ui-page-header__back"
            aria-label="戻る"
            onClick={() => navigate("/inquiries")}
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <path d="M19 12H5" />
              <path d="M12 19l-7-7 7-7" />
            </svg>
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