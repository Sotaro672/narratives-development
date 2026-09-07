// frontend/amol/src/pages/PublicWalletPage.tsx

import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/wallet-page.css";
import "../styles/wallet-page/resale-panel.css";
import "../styles/avatar-review-page.css";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";
import { fetchAvatarReviews, type AvatarReviewPageResponse } from "../features/avatar-review/api/avatarReviewApi";
import ReportModal from "../features/report/components/ReportModal";
import { useReport } from "../features/report/hooks/useReport";
import WalletProfile from "../features/wallet/components/WalletProfile";
import WalletResalePanel from "../features/wallet/components/WalletResalePanel";
import { useWalletPage } from "../features/wallet/hooks/useWalletPage";

export default function PublicWalletPage() {
  const navigate = useNavigate();

  const {
    avatarId,
    viewedAvatarId,
    avatarName,
    avatarIcon,
    profile,
    isOwnAvatar,
    loading,
    error,
    pageTitle,
  } = useWalletPage();

  const {
    target: reportTarget,
    isOpen: reportOpen,
    reason: reportReason,
    detail: reportDetail,
    submitting: reportSubmitting,
    error: reportError,
    result: reportResult,
    canSubmit: canSubmitReport,
    openAvatarReport,
    close: closeReport,
    setReason: setReportReason,
    setDetail: setReportDetail,
    submit: submitReport,
  } = useReport();

  const targetAvatarId = viewedAvatarId || avatarId;
  const normalizedTargetAvatarId = targetAvatarId.trim();
  const canReportAvatar = Boolean(normalizedTargetAvatarId) && !isOwnAvatar;

  const [reviewSummary, setReviewSummary] =
    useState<AvatarReviewPageResponse | null>(null);
  const [reviewLoading, setReviewLoading] = useState(false);

  useEffect(() => {
    const id = targetAvatarId.trim();

    if (!id || loading || error) {
      setReviewSummary(null);
      return;
    }

    let active = true;

    const loadReviewSummary = async () => {
      setReviewLoading(true);

      try {
        const result = await fetchAvatarReviews({
          avatarId: id,
          page: 1,
          perPage: 1,
        });

        if (active) {
          setReviewSummary(result);
        }
      } catch {
        if (active) {
          setReviewSummary(null);
        }
      } finally {
        if (active) {
          setReviewLoading(false);
        }
      }
    };

    void loadReviewSummary();

    return () => {
      active = false;
    };
  }, [error, loading, targetAvatarId]);

  const handleOpenMarketDetail = (resaleId: string) => {
    const id = resaleId.trim();

    if (!id) {
      return;
    }

    navigate(`/market/${encodeURIComponent(id)}`);
  };

  const handleOpenAvatarReviews = () => {
    const id = targetAvatarId.trim();

    if (!id) {
      return;
    }

    navigate(`/avatars/${encodeURIComponent(id)}/reviews`);
  };

  const handleOpenAvatarReport = () => {
    if (!canReportAvatar) {
      return;
    }

    openAvatarReport({
      avatarId: normalizedTargetAvatarId,
    });
  };

  return (
    <>
      <Layout
        title={pageTitle || "AMOL"}
        mode="mypage"
        showBackButton
        onBackButtonClick={() => navigate(-1)}
      >
        <section className="content-page-section wallet-page">
          <div className="wallet-page-layout">
            <aside className="wallet-page-layout__profile">
              {loading ? (
                <p className="wallet-page__message">
                  読み込み中です...
                </p>
              ) : null}

              {!loading && error ? (
                <div role="alert" className="wallet-page__message">
                  <p>{error}</p>
                </div>
              ) : null}

              {!loading && !error ? (
                <>
                  <WalletProfile
                    avatarName={avatarName}
                    avatarIcon={avatarIcon}
                    profile={profile}
                    isOwnAvatar={isOwnAvatar}
                  />

                  <button
                    type="button"
                    className="avatar-review-summary"
                    onClick={handleOpenAvatarReviews}
                    disabled={!targetAvatarId || reviewLoading}
                  >
                    <span className="avatar-review-summary__item">
                      <span className="avatar-review-summary__label">
                        良かった
                      </span>
                      <strong className="avatar-review-summary__count">
                        {reviewLoading ? "-" : reviewSummary?.goodCount ?? 0}
                      </strong>
                    </span>

                    <span className="avatar-review-summary__divider" />

                    <span className="avatar-review-summary__item">
                      <span className="avatar-review-summary__label">
                        残念だった
                      </span>
                      <strong className="avatar-review-summary__count">
                        {reviewLoading
                          ? "-"
                          : reviewSummary?.disappointedCount ?? 0}
                      </strong>
                    </span>

                    <span
                      className="avatar-review-summary__arrow"
                      aria-hidden="true"
                    >
                      ›
                    </span>
                  </button>

                  {canReportAvatar ? (
                    <div className="wallet-page-profile-actions-bar">
                      <div className="wallet-page-profile-actions-bar__inner">
                        <Button
                          type="button"
                          variant="secondary"
                          size="sm"
                          fullWidth
                          disabled={reportSubmitting}
                          aria-label={`${avatarName || "アバター"}を通報`}
                          onClick={handleOpenAvatarReport}
                        >
                          通報
                        </Button>
                      </div>
                    </div>
                  ) : null}
                </>
              ) : null}
            </aside>

            <div className="wallet-page-layout__main">
              {!loading && !error ? (
                <WalletResalePanel
                  avatarId={targetAvatarId}
                  onItemClick={handleOpenMarketDetail}
                />
              ) : null}
            </div>
          </div>
        </section>
      </Layout>

      <ReportModal
        open={reportOpen}
        targetType={reportTarget?.type}
        reason={reportReason}
        detail={reportDetail}
        submitting={reportSubmitting}
        error={reportError}
        result={reportResult}
        canSubmit={canSubmitReport}
        onReasonChange={setReportReason}
        onDetailChange={setReportDetail}
        onSubmit={submitReport}
        onClose={closeReport}
      />
    </>
  );
}