// frontend/admin/shell/src/pages/TradePage.tsx

import { useNavigate, useParams } from "react-router-dom";

import ResaleDetailAside from "../features/resale/presentation/components/ResaleDetailAside";
import { useResaleDetail } from "../features/resale/presentation/hooks/useResaleDetail";
import TradeMessageTable from "../features/trade/presentation/components/TradeMessageTable";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";

export default function TradePage() {
  const navigate = useNavigate();
  const { avatarId = "", resaleId = "", tradeId = "" } = useParams<{
    avatarId?: string;
    resaleId?: string;
    tradeId?: string;
  }>();

  const { resale, loading, error, reload } = useResaleDetail(
    avatarId,
    resaleId,
  );

  return (
    <Page>
      <PageHeader
        title="取引詳細"
        leading={
          <button
            type="button"
            className="ui-page-header__back"
            aria-label="戻る"
            onClick={() =>
              navigate(
                `/avatars/${encodeURIComponent(avatarId)}/resales/${encodeURIComponent(resaleId)}`,
              )
            }
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
        main={<TradeMessageTable tradeId={tradeId} />}
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