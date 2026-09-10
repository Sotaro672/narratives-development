// frontend/admin/shell/src/pages/MintDetailPage.tsx

import { useNavigate, useParams } from "react-router-dom";

import Page, { PageHeader } from "../shared/ui/Page/Page";

export default function MintDetailPage() {
  const navigate = useNavigate();
  const { mintId = "" } = useParams<{ mintId: string }>();

  return (
    <Page>
      <PageHeader
        title="Mint詳細"
        meta={mintId || undefined}
        leading={
          <button
            type="button"
            className="ui-page-header__back"
            aria-label="戻る"
            onClick={() => navigate("/gas")}
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

      <section>
        <p>Mint ID: {mintId}</p>
      </section>
    </Page>
  );
}