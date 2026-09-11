// frontend/admin/shell/src/pages/ResalePage.tsx

import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import ResaleDetailAside from "../features/resale/presentation/components/ResaleDetailAside";
import ResaleMediaSection from "../features/resale/presentation/components/ResaleMediaSection";
import ResaleReviewTable from "../features/resale/presentation/components/ResaleReviewTable";
import ResaleStatusTab from "../features/resale/presentation/components/ResaleStatusTab";
import { useResaleDetail } from "../features/resale/presentation/hooks/useResaleDetail";
import TradeTable from "../features/trade/presentation/components/TradeTable";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";

import "./ResalePage.css";

type ResalePageTab = "reviews" | "trades";

export default function ResalePage() {
  const navigate = useNavigate();
  const { avatarId = "", resaleId = "" } = useParams<{
    avatarId?: string;
    resaleId?: string;
  }>();

  const { resale, loading, error, reload } = useResaleDetail(avatarId, resaleId);
  const [activeTab, setActiveTab] = useState<ResalePageTab>("reviews");

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
        main={
          <>
            <ResaleMediaSection resale={resale} />

            <div className="resale-page__related">
              <div
                className="resale-page__tabs"
                role="tablist"
                aria-label="再販関連情報"
              >
                <button
                  id="resale-page-tab-reviews"
                  type="button"
                  role="tab"
                  className={[
                    "resale-page__tab",
                    activeTab === "reviews"
                      ? "resale-page__tab--active"
                      : "",
                  ].filter(Boolean).join(" ")}
                  aria-selected={activeTab === "reviews"}
                  aria-controls="resale-page-panel-reviews"
                  onClick={() => setActiveTab("reviews")}
                >
                  レビュー
                </button>

                <button
                  id="resale-page-tab-trades"
                  type="button"
                  role="tab"
                  className={[
                    "resale-page__tab",
                    activeTab === "trades"
                      ? "resale-page__tab--active"
                      : "",
                  ].filter(Boolean).join(" ")}
                  aria-selected={activeTab === "trades"}
                  aria-controls="resale-page-panel-trades"
                  onClick={() => setActiveTab("trades")}
                >
                  取引
                </button>
              </div>

              {activeTab === "reviews" && (
                <div
                  id="resale-page-panel-reviews"
                  className="resale-page__panel"
                  role="tabpanel"
                  aria-labelledby="resale-page-tab-reviews"
                >
                  <ResaleReviewTable
                    avatarId={avatarId}
                    resaleId={resaleId}
                    perPage={20}
                  />
                </div>
              )}

              {activeTab === "trades" && (
                <div
                  id="resale-page-panel-trades"
                  className="resale-page__panel"
                  role="tabpanel"
                  aria-labelledby="resale-page-tab-trades"
                >
                  <TradeTable
                    avatarId={avatarId}
                    resaleId={resaleId}
                  />
                </div>
              )}
            </div>
          </>
        }
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