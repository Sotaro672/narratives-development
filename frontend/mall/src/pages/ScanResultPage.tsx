// frontend/mall/src/pages/ScanResultPage.tsx

import { useCallback, useEffect, useRef, useState } from "react";
import { ChevronDown } from "lucide-react";
import { useLocation, useNavigate } from "react-router-dom";

import { useMobilePortrait } from "../components/hooks/useMobilePortrait";
import Layout from "../components/layout/Layout";
import MobileComposerFooter from "../components/layout/MobileComposerFooter";
import MobileSwipeDismissPage, {
  type MobileSwipeDismissPageHandle,
} from "../components/layout/MobileSwipeDismissPage";
import Button from "../components/ui/Button";
import IconButton from "../components/ui/IconButton";
import RatingSelect from "../components/ui/RatingSelect";

import ScanResultCard from "../features/scan-result/presentation/components/ScanResultCard";
import ScanTransferConfirmModal from "../features/scan-result/presentation/components/ScanTransferConfirmModal";
import ScanTransferSuccessModal from "../features/scan-result/presentation/components/ScanTransferSuccessModal";
import { useScanResultPage } from "../features/scan-result/presentation/hooks/useScanResultPage";

import "../styles/page-layout.css";
import "../styles/scan-result-page.css";

export default function ScanResultPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const isMobilePortrait = useMobilePortrait();
  const swipeDismissRef = useRef<MobileSwipeDismissPageHandle | null>(null);

  const [reviewBody, setReviewBody] = useState("");
  const [reviewRating, setReviewRating] = useState(5);

  const {
    state,
    viewModel,
    currentAvatarId,
    hasMultipleTransfers,
    canOpenTransferContents,
    load,
    submitReview,
    voteHelpfulReview,
    deleteReview,
    openContentsAfterResolve,
    openTokenContentsByAssetId,
    transferConfirmModalOpen,
    transferModalOpen,
    transferModalError,
    confirmTransfer,
    closeTransferConfirmModal,
    closeTransferModal,
  } = useScanResultPage();

  const isLoggedIn = state.authAvailable === true;
  const isLoggedInMobile = isLoggedIn && isMobilePortrait;

  const isWalletOverlay =
    isLoggedInMobile &&
    (location.pathname === "/wallet/scan-result" ||
      location.pathname.startsWith("/wallet/scan-result/"));

  const productBlueprintId =
    state.previewState?.raw.productBlueprintId?.trim() ?? "";

  const canSubmitReview =
    isLoggedIn &&
    !state.loading &&
    !state.postingReview &&
    Boolean(productBlueprintId) &&
    Boolean(reviewBody.trim()) &&
    Number.isInteger(reviewRating) &&
    reviewRating >= 1 &&
    reviewRating <= 5;

  const swipeDismissEnabled =
    isWalletOverlay &&
    !transferConfirmModalOpen &&
    !transferModalOpen;

  useEffect(() => {
    if (!isLoggedInMobile || isWalletOverlay) {
      return;
    }

    const productId = state.productId.trim();

    if (!productId) {
      return;
    }

    const searchParams = new URLSearchParams(location.search);
    searchParams.set("productId", productId);

    navigate(`/wallet/scan-result?${searchParams.toString()}`, {
      replace: true,
    });
  }, [
    isLoggedInMobile,
    isWalletOverlay,
    location.search,
    navigate,
    state.productId,
  ]);

  const handleSubmitReview = useCallback(async () => {
    if (!canSubmitReview) {
      return;
    }

    const ok = await submitReview(reviewBody, reviewRating);

    if (!ok) {
      return;
    }

    setReviewBody("");
    setReviewRating(5);
  }, [
    canSubmitReview,
    reviewBody,
    reviewRating,
    submitReview,
  ]);

  const handleOpenInquiryPage = useCallback(() => {
    const productId = state.productId.trim();

    if (!productId) {
      return;
    }

    const searchParams = new URLSearchParams({ productId });
    navigate(`/inquiries/new?${searchParams.toString()}`);
  }, [navigate, state.productId]);

  const handleAvatarClick = useCallback(
    (avatarId: string) => {
      const normalizedAvatarId = avatarId.trim();
      const normalizedCurrentAvatarId = currentAvatarId.trim();

      if (!normalizedAvatarId) {
        return;
      }

      if (
        normalizedCurrentAvatarId &&
        normalizedAvatarId === normalizedCurrentAvatarId
      ) {
        navigate("/wallet");
        return;
      }

      navigate(`/avatars/${encodeURIComponent(normalizedAvatarId)}`);
    },
    [currentAvatarId, navigate],
  );

  const handleDismiss = useCallback(() => {
    navigate("/wallet", {
      replace: true,
    });
  }, [navigate]);

  const handleDismissButtonClick = useCallback(() => {
    swipeDismissRef.current?.dismiss();
  }, []);

  const isResalePurchase =
    state.transferResult?.matchedItemType === "resale" ||
    hasMultipleTransfers;

  const canOpenInquiryPage =
    isLoggedIn &&
    !state.loading &&
    !state.busyTransfer &&
    Boolean(state.productId.trim()) &&
    !isResalePurchase;

  const content = (
    <>
      <Layout
        title="AMOL"
        mode={isLoggedIn ? "mypage" : "landing"}
        showHeader={!isLoggedInMobile}
        showFooter={isLoggedIn && !isLoggedInMobile}
        hideSettingsButton={!isLoggedIn}
        hideAnnouncementButton={!isLoggedIn}
        actionButtonLabel={canOpenInquiryPage ? "問い合わせ" : undefined}
        onActionButtonClick={
          canOpenInquiryPage ? handleOpenInquiryPage : undefined
        }
        disableFooterPaddingOnDesktop
      >
        {isWalletOverlay ? (
          <div
            className="scan-result-page__dismiss-control"
            data-mobile-swipe-dismiss-ignore="true"
          >
            <IconButton
              type="button"
              variant="floating"
              size="md"
              aria-label="スキャン結果を閉じる"
              onClick={handleDismissButtonClick}
            >
              <ChevronDown
                size={24}
                strokeWidth={2.25}
                aria-hidden="true"
              />
            </IconButton>

            {canOpenInquiryPage ? (
              <Button
                type="button"
                variant="secondary"
                size="md"
                className="scan-result-page__inquiry-button"
                onClick={handleOpenInquiryPage}
              >
                お問い合わせ
              </Button>
            ) : null}
          </div>
        ) : null}

        <section
          className={[
            "product-detail-page-layout",
            "scan-result-page-layout",
            isWalletOverlay
              ? "scan-result-page-layout--with-dismiss-control"
              : "",
            isLoggedInMobile
              ? "scan-result-page-layout--with-review-composer"
              : "",
          ]
            .filter(Boolean)
            .join(" ")}
        >
          <ScanResultCard
            state={state}
            viewModel={viewModel}
            currentAvatarId={currentAvatarId}
            onRefresh={load}
            onAvatarClick={handleAvatarClick}
            onOpenTokenContents={openTokenContentsByAssetId}
            tokenContentsDisabled={!isLoggedIn}
            reviewBody={reviewBody}
            reviewRating={reviewRating}
            reviewSubmitting={state.postingReview}
            reviewError={state.postReviewError}
            reviewCanSubmit={canSubmitReview}
            reviewDisabled={state.loading || !productBlueprintId}
            onReviewBodyChange={setReviewBody}
            onReviewRatingChange={setReviewRating}
            onSubmitReview={handleSubmitReview}
            onHelpfulVote={voteHelpfulReview}
            onDeleteReview={deleteReview}
          />
        </section>

        <ScanTransferConfirmModal
          open={transferConfirmModalOpen}
          loading={state.busyTransfer}
          error={transferModalError}
          onCancel={closeTransferConfirmModal}
          onConfirm={confirmTransfer}
        />

        <ScanTransferSuccessModal
          open={transferModalOpen}
          loading={state.busyTransfer}
          error={transferModalError}
          canOpenContents={canOpenTransferContents}
          onClose={closeTransferModal}
          onOpenContents={openContentsAfterResolve}
        />
      </Layout>

      {isLoggedInMobile ? (
        <MobileComposerFooter
          content={reviewBody}
          placeholder="レビューを書く…"
          error={state.postReviewError}
          submitting={state.postingReview}
          canSubmit={canSubmitReview}
          disabled={state.loading || !productBlueprintId}
          submitLabel="投稿"
          submittingLabel="投稿中..."
          onContentChange={setReviewBody}
          onSubmit={handleSubmitReview}
          beforeInput={
            <div className="scan-result-mobile-review-rating">
              <span className="scan-result-mobile-review-rating__label">
                評価
              </span>

              <RatingSelect
                value={reviewRating}
                onChange={setReviewRating}
                disabled={state.postingReview}
                ariaLabel="商品評価"
              />
            </div>
          }
        />
      ) : null}
    </>
  );

  if (!isWalletOverlay) {
    return content;
  }

  return (
    <MobileSwipeDismissPage
      ref={swipeDismissRef}
      enabled={swipeDismissEnabled}
      onDismiss={handleDismiss}
    >
      {content}
    </MobileSwipeDismissPage>
  );
}