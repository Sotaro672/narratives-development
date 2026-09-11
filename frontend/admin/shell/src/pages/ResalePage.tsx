// frontend/admin/shell/src/pages/ResalePage.tsx

import { useNavigate, useParams } from "react-router-dom";

import ResaleDetailAside from "../features/resale/presentation/components/ResaleDetailAside";
import ResaleMediaSection from "../features/resale/presentation/components/ResaleMediaSection";
import ResaleStatusTab from "../features/resale/presentation/components/ResaleStatusTab";
import { useResaleDetail } from "../features/resale/presentation/hooks/useResaleDetail";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";

export default function ResalePage() {
  const navigate = useNavigate();
  const { avatarId = "", resaleId = "" } = useParams<{
    avatarId?: string;
    resaleId?: string;
  }>();

  const { resale, loading, error, reload } = useResaleDetail(avatarId, resaleId);

  return (
    <Page>
      <PageHeader
        title={resale?.productName || "Resale詳細"}
        meta={
          resale?.status ? (
            <ResaleStatusTab status={resale.status} />
          ) : undefined
        }
        leading={
          <button
            type="button"
            className="ui-page-header__back"
            aria-label="戻る"
            onClick={() => navigate(`/avatars/${encodeURIComponent(avatarId)}`)}
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

      <DetailPageBody
        main={<ResaleMediaSection resale={resale} />}
        aside={
          <ResaleDetailAside
            resale={resale}
            loading={loading}
            error={error}
            onReload={reload}
          />
        }
      />
    </Page>
  );
}