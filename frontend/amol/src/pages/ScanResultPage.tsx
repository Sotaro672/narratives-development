// frontend/amol/src/pages/ScanResultPage.tsx
import { useCallback, useState } from "react";

import Layout from "../components/layout/Layout";
import ScanResultCard from "../features/scan-result/presentation/components/ScanResultCard";
import { useMobilePortrait } from "../features/catalog/presentation/hooks/useMobilePortrait";
import { useScanResultPage } from "../features/scan-result/presentation/hooks/useScanResultPage";

import "../styles/scan-result-page.css";

export default function ScanResultPage() {
  const isMobilePortrait = useMobilePortrait();

  const [reviewBody, setReviewBody] = useState("");
  const [reviewRating, setReviewRating] = useState(5);

  const {
    state,
    displayTransfers,
    load,
    submitReview,
    nextReviewsPage,
    prevReviewsPage,
    openTokenContentsByMintAddress,
  } = useScanResultPage();

  const handleSubmitReview = useCallback(async () => {
    const ok = await submitReview(reviewBody, reviewRating);

    if (ok) {
      setReviewBody("");
      setReviewRating(5);
    }
  }, [reviewBody, reviewRating, submitReview]);

  const isLoggedIn = state.authAvailable === true;

  const canPostReview =
    state.ownedByWallet === true &&
    !state.loading &&
    !state.postingReview &&
    Boolean(reviewBody.trim());

  return (
    <Layout
      title="スキャン結果"
      mode={isLoggedIn ? "mypage" : "landing"}
      showHeader
      showBackButton={isLoggedIn && isMobilePortrait}
      showFooter={isLoggedIn}
      backTo="/wallet"
      hideHamburgerMenu={false}
      hideSettingsButton={!isLoggedIn}
      mainClassName="scan-result-page"
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
        transfers={displayTransfers}
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
    </Layout>
  );
}