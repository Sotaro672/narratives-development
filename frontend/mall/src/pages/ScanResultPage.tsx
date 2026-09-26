// frontend/mall/src/pages/ScanResultPage.tsx

import { useCallback, useState } from "react";
import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";

import ScanResultCard from "../features/scan-result/presentation/components/ScanResultCard";
import ScanTransferConfirmModal from "../features/scan-result/presentation/components/ScanTransferConfirmModal";
import ScanTransferSuccessModal from "../features/scan-result/presentation/components/ScanTransferSuccessModal";
import { useScanResultPage } from "../features/scan-result/presentation/hooks/useScanResultPage";
import ProductBlueprintReviewModal from "../features/shared/presentation/components/ProductBlueprintReviewModal";

import "../styles/page-layout.css";
import "../styles/scan-result-page.css";

export default function ScanResultPage() {
  const navigate = useNavigate();

  const [reviewBody, setReviewBody] = useState("");
  const [reviewRating, setReviewRating] = useState(5);
  const [reviewModalOpen, setReviewModalOpen] = useState(false);

  const {
    state,
    viewModel,
    currentAvatarId,
    hasMultipleTransfers,
    canOpenTransferContents,
    load,
    submitReview,
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

  const handleSubmitReview = useCallback(async () => {
    const ok = await submitReview(reviewBody, reviewRating);

    if (ok) {
      setReviewBody("");
      setReviewRating(5);
      setReviewModalOpen(false);
    }
  }, [reviewBody, reviewRating, submitReview]);

  const handleOpenReviewModal = useCallback(() => {
    if (!isLoggedIn || state.postingReview) {
      return;
    }

    setReviewModalOpen(true);
  }, [isLoggedIn, state.postingReview]);

  const handleCloseReviewModal = useCallback(() => {
    if (state.postingReview) {
      return;
    }

    setReviewModalOpen(false);
  }, [state.postingReview]);

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

  const isResalePurchase =
    state.transferResult?.matchedItemType === "resale" ||
    hasMultipleTransfers;

  const canOpenInquiryPage =
    isLoggedIn &&
    !state.loading &&
    !state.busyTransfer &&
    Boolean(state.productId.trim()) &&
    !isResalePurchase;

  return (
    <>
      <Layout
        title="AMOL"
        mode={isLoggedIn ? "mypage" : "landing"}
        showHeader
        hideHamburgerMenu={false}
        hideSettingsButton={!isLoggedIn}
        hideAnnouncementButton={!isLoggedIn}
      >
        <section className="product-detail-page-layout scan-result-page-layout">
          <ScanResultCard
            state={state}
            viewModel={viewModel}
            currentAvatarId={currentAvatarId}
            onRefresh={load}
            onAvatarClick={handleAvatarClick}
            onOpenTokenContents={openTokenContentsByAssetId}
            onOpenReviewModal={handleOpenReviewModal}
            canOpenInquiryPage={canOpenInquiryPage}
            onOpenInquiryPage={handleOpenInquiryPage}
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

      <ProductBlueprintReviewModal
        open={reviewModalOpen}
        body={reviewBody}
        rating={reviewRating}
        submitting={state.postingReview}
        error={state.postReviewError}
        onBodyChange={setReviewBody}
        onRatingChange={setReviewRating}
        onCancel={handleCloseReviewModal}
        onSubmit={handleSubmitReview}
      />
    </>
  );
}