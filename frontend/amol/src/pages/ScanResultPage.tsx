// frontend/amol/src/pages/ScanResultPage.tsx
import { useCallback, useState } from "react";
import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import ScanResultCard from "../features/scan-result/presentation/components/ScanResultCard";
import ScanTransferSuccessModal from "../features/scan-result/presentation/components/ScanTransferSuccessModal";
import { useMobilePortrait } from "../features/catalog/presentation/hooks/useMobilePortrait";
import { useScanResultPage } from "../features/scan-result/presentation/hooks/useScanResultPage";

import "../styles/scan-result-page.css";

export default function ScanResultPage() {
  const navigate = useNavigate();
  const isMobilePortrait = useMobilePortrait();

  const [reviewBody, setReviewBody] = useState("");
  const [reviewRating, setReviewRating] = useState(5);

  const {
    state,
    hasMultipleTransfers,
    load,
    submitReview,
    nextReviewsPage,
    prevReviewsPage,
    openContentsAfterResolve,
    openTokenContentsByMintAddress,
    transferModalOpen,
    transferModalError,
    closeTransferModal,
    transferSuccessModalViewModel,
  } = useScanResultPage();

  const handleSubmitReview = useCallback(async () => {
    const ok = await submitReview(reviewBody, reviewRating);

    if (ok) {
      setReviewBody("");
      setReviewRating(5);
    }
  }, [reviewBody, reviewRating, submitReview]);

  const handleOpenInquiryPage = useCallback(() => {
    const productId = state.productId.trim();

    if (!productId) {
      return;
    }

    navigate(`/inquiries/new?productId=${encodeURIComponent(productId)}`);
  }, [navigate, state.productId]);

  const isLoggedIn = state.authAvailable === true;

  const canPostReview =
    state.ownedByWallet === true &&
    !state.loading &&
    !state.postingReview &&
    Boolean(reviewBody.trim());

  const canOpenInquiryPage =
    isLoggedIn &&
    Boolean(state.productId.trim()) &&
    !hasMultipleTransfers;

  return (
    <Layout
      title="AMOL"
      mode={isLoggedIn ? "mypage" : "landing"}
      showHeader
      showBackButton={isLoggedIn && isMobilePortrait}
      showFooter={isLoggedIn}
      backTo="/wallet"
      hideHamburgerMenu={false}
      hideSettingsButton={!isLoggedIn}
      mainClassName="scan-result-page"
      secondaryActionButtonLabel={canOpenInquiryPage ? "問い合わせ" : undefined}
      onSecondaryActionButtonClick={
        canOpenInquiryPage ? handleOpenInquiryPage : undefined
      }
      secondaryActionButtonDisabled={!canOpenInquiryPage}
      footerProps={
        isLoggedIn && isMobilePortrait && state.ownedByWallet === true
          ? {
              variant: "reviewAction",
              value: reviewBody,
              rating: reviewRating,
              placeholder: "口コミを入力",
              buttonLabel: state.postingReview ? "投稿中" : "投稿",
              disabled: !canPostReview,
              posting: state.postingReview,
              onChange: setReviewBody,
              onRatingChange: setReviewRating,
              onSubmit: handleSubmitReview,
            }
          : { variant: "default" }
      }
    >
      <ScanResultCard
        state={state}
        onRefresh={load}
        onPrevReviewsPage={prevReviewsPage}
        onNextReviewsPage={nextReviewsPage}
        onOpenTokenContents={openTokenContentsByMintAddress}
        reviewBody={reviewBody}
        reviewRating={reviewRating}
        onReviewBodyChange={setReviewBody}
        onReviewRatingChange={setReviewRating}
        onSubmitReviewForm={handleSubmitReview}
        hideReviewForm={isMobilePortrait}
      />

      <ScanTransferSuccessModal
        open={transferModalOpen}
        loading={state.busyTransfer || state.resolvingTransferredToken}
        error={transferModalError}
        viewModel={transferSuccessModalViewModel}
        resolvedContentsReady={Boolean(state.resolvedTransferredToken)}
        onClose={closeTransferModal}
        onOpenContents={openContentsAfterResolve}
      />
    </Layout>
  );
}