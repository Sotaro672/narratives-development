// frontend/mall/src/features/scan-result/presentation/components/ScanResultCard.tsx

import Button from "../../../../components/ui/Button";
import SectionCard from "../../../../components/ui/SectionCard";
import TextState from "../../../../components/ui/TextState";

import type { ScanResultPageViewModel } from "../../application/scanPageViewModelFactory";
import type { ScanResultPageState } from "../../../shared/types/scanResult";
import ProductReviewSection from "../../../shared/presentation/components/ProductReviewSection";
import TokenSummaryCard from "../../../shared/presentation/components/TokenSummaryCard";

import ScanResultProductSection from "./ScanResultProductSection";

type ScanResultCardProps = {
  state: ScanResultPageState;
  viewModel: ScanResultPageViewModel | null;
  currentAvatarId: string;
  onRefresh: () => void;
  onAvatarClick: (avatarId: string) => void;
  onOpenTokenContents: (assetId: string) => void | Promise<void>;
  onOpenReviewModal: () => void;
  hideReviewAction?: boolean;
};

export default function ScanResultCard(props: ScanResultCardProps) {
  const {
    state,
    viewModel,
    currentAvatarId,
    onRefresh,
    onAvatarClick,
    onOpenTokenContents,
    onOpenReviewModal,
    hideReviewAction = false,
  } = props;

  if (state.loading) {
    return (
      <SectionCard>
        <TextState variant="loading">
          プレビューを取得しています...
        </TextState>
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
  const productBlueprintId =
    state.previewState?.raw.productBlueprintId?.trim() ?? "";

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
          <TokenSummaryCard
            brandName={token.brandName}
            tokenName={token.tokenName}
            tokenIcon={token.tokenIcon}
            symbol={token.symbol}
            description={token.description}
            onClick={() => void onOpenTokenContents(token.assetId)}
            disabled={!token.canOpenTokenContents}
          />
        ) : null}
      </div>

      <aside className="scan-result-desktop-side">
        {state.authAvailable === true && !hideReviewAction ? (
          <div className="scan-result-review-action">
            <Button
              type="button"
              disabled={state.postingReview}
              onClick={onOpenReviewModal}
            >
              レビューを書く
            </Button>
          </div>
        ) : null}

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
      </aside>
    </div>
  );
}