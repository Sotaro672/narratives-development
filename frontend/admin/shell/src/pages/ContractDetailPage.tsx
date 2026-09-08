// frontend/admin/shell/src/pages/ContractDetailPage.tsx

import { useLocation, useNavigate, useParams } from "react-router-dom";

import type { Company } from "../shared/type/company";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";
import { formatDateTime } from "../shared/util/dateFormat";

type ContractDetailLocationState = {
  company?: Company;
};

export default function ContractDetailPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const { companyId = "" } = useParams<{ companyId?: string }>();

  const locationState = location.state as ContractDetailLocationState | null;
  const stateCompany = locationState?.company;
  const company = stateCompany && stateCompany.id === companyId ? stateCompany : null;

  return (
    <Page>
      <PageHeader
        title="契約詳細"
        leading={
          <button
            type="button"
            className="ui-page-header__back"
            aria-label="戻る"
            onClick={() => navigate("/contracts")}
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

      {!company ? (
        <p role="alert">
          企業情報を取得できませんでした。契約一覧から開き直してください。
        </p>
      ) : (
        <DetailPageBody
          main={<div />}
          aside={
            <section className="ui-detail-section">
              <h2 className="ui-detail-section__title">企業情報</h2>
              <dl className="ui-detail-definition-list">
                <dt>企業名</dt>
                <dd>{company.name || "-"}</dd>

                <dt>代表者</dt>
                <dd>{company.representativeName || "-"}</dd>

                <dt>登録日時</dt>
                <dd>{formatDateTime(company.createdAt)}</dd>

                <dt>最終更新日</dt>
                <dd>{formatDateTime(company.updatedAt)}</dd>
              </dl>
            </section>
          }
        />
      )}
    </Page>
  );
}