// frontend/mall/src/features/scan-result/presentation/components/ScanResultCard.tsx

import Button from "../../../../components/ui/Button";
import RatingSelect from "../../../../components/ui/RatingSelect";
import SectionCard from "../../../../components/ui/SectionCard";
import SectionHeader from "../../../../components/ui/SectionHeader";
import Textbox from "../../../../components/ui/Textbox";
import TextState from "../../../../components/ui/TextState";

import type { ScanResultPageViewModel } from "../../application/scanPageViewModelFactory";
import ProductReviewSection from "../../../shared/presentation/components/ProductReviewSection";
import TokenSummaryCard from "../../../shared/presentation/components/TokenSummaryCard";
import type { ScanResultPageState } from "../../../shared/types/scanResult";

import ScanResultProductSection from "./ScanResultProductSection";

type ScanResultCardProps = {
  state: ScanResultPageState;
  viewModel: ScanResultPageViewModel | null;
  currentAvatarId: string;
  onRefresh: () => void;
  onAvatarClick: (avatarId: string) => void;
  onOpenTokenContents: (assetId: string) => void | Promise<void>;
  tokenContentsDisabled?: boolean;
  reviewBody: string;
  reviewRating: number;
  reviewSubmitting: boolean;
  reviewError?: string | null;
  reviewCanSubmit: boolean;
  reviewDisabled: boolean;
  onReviewBodyChange: (value: string) => void;
  onReviewRatingChange: (value: number) => void;
  onSubmitReview: () => void | Promise<void>;
};

export default function ScanResultCard(props: ScanResultCardProps) {
  const {
    state,
    viewModel,
    currentAvatarId,
    onRefresh,
    onAvatarClick,
    onOpenTokenContents,
    tokenContentsDisabled = false,
    reviewBody,
    reviewRating,
    reviewSubmitting,
    reviewError,
    reviewCanSubmit,
    reviewDisabled,
    onReviewBodyChange,
    onReviewRatingChange,
    onSubmitReview,
  } = props;

  if (state.loading) {
    return (
      <SectionCard>
        <TextState variant="loading">プレビューを取得しています...</TextState>
      </SectionCard>
    );
  }

  if (state.error) {
    return (
      <SectionCard>
        <h1>Scan Result</h1>
        <TextState variant="error">{state.error}</TextState>
        <Button type="button" onClick={onRefresh}>
          再読み込み
        </Button>
      </SectionCard>
    );
  }

  if (!viewModel) {
    return (
      <SectionCard>
        <h1>Scan Result</h1>
        <TextState>プレビューが空です。</TextState>
      </SectionCard>
    );
  }

  const { product, token } = viewModel;
  const owned = state.ownedByWallet;
  const ownedError = state.ownedByWalletError ?? "";
  const productBlueprintId = state.previewState?.raw.productBlueprintId?.trim() ?? "";
  const tokenCardDisabled = tokenContentsDisabled || !token?.canOpenTokenContents;

  const handleReviewSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (!reviewCanSubmit) {
      return;
    }

    void onSubmitReview();
  };

  return (
    <div className="scan-result-desktop-grid">
      <div className="scan-result-desktop-main">
        {state.transferError ? (
          <SectionCard>
            <TextState variant="error">
              NFT受取処理に失敗しました: {state.transferError}
            </TextState>
          </SectionCard>
        ) : null}

        <ScanResultProductSection
          title={product.title}
          owned={owned}
          ownedError={ownedError}
          ownerLabel={product.ownerLabel}
          brandId={product.brandId}
          brandName={product.brandName}
          hasBrandInfo={product.hasBrandInfo}
          productBlueprintRows={product.productBlueprintRows}
          qualityAssuranceTabs={product.qualityAssuranceTabs}
          modelNumber={product.modelNumber}
          size={product.size}
          color={product.color}
          swatch={product.swatch}
          measurementEntries={product.measurementEntries}
          alcoholInfo={product.alcoholInfo}
        />

        {token ? (
          <div className="scan-result-token-card">
            <TokenSummaryCard
              brandName={token.brandName}
              tokenName={token.tokenName}
              tokenIcon={token.tokenIcon}
              symbol={token.symbol}
              onClick={
                tokenCardDisabled
                  ? undefined
                  : () => void onOpenTokenContents(token.assetId)
              }
              disabled={tokenCardDisabled}
            />
          </div>
        ) : null}
      </div>

      <aside className="scan-result-desktop-side">
        <section className="scan-result-desktop-review-form">
          <SectionHeader
            title="レビューを投稿"
            titleAs="h2"
            titleSize="sm"
          />

          <form
            className="scan-result-desktop-review-form__body"
            onSubmit={handleReviewSubmit}
          >
            <div className="scan-result-desktop-review-form__rating">
              <span className="scan-result-desktop-review-form__label">
                評価
              </span>

              <RatingSelect
                value={reviewRating}
                onChange={onReviewRatingChange}
                disabled={reviewSubmitting || reviewDisabled}
                ariaLabel="商品評価"
              />
            </div>

            <Textbox
              value={reviewBody}
              rows={4}
              placeholder="レビューを書く…"
              aria-label="レビュー本文"
              disabled={reviewSubmitting || reviewDisabled}
              onChange={(event) => {
                onReviewBodyChange(event.currentTarget.value);
              }}
            />

            {reviewError ? (
              <TextState variant="error">{reviewError}</TextState>
            ) : null}

            <div className="scan-result-desktop-review-form__actions">
              <Button
                type="submit"
                disabled={!reviewCanSubmit}
              >
                {reviewSubmitting ? "投稿中..." : "投稿"}
              </Button>
            </div>
          </form>
        </section>

        <div className="scan-result-review-scroll">
          <ProductReviewSection
            items={state.reviews?.items ?? []}
            productBlueprintId={productBlueprintId}
            currentAvatarId={currentAvatarId}
            totalCount={state.reviews?.total}
            loading={state.busyReviews}
            errorMessage={state.reviewsError}
            showHelpfulVotes
            onAvatarClick={onAvatarClick}
          />
        </div>
      </aside>
    </div>
  );
}