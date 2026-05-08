// frontend/amol/src/features/scan-result/components/ScanResultCard.tsx
import { useMemo } from "react";

import Button from "../../../components/ui/Button";
import SectionCard from "../../../components/ui/SectionCard";
import TextState from "../../../components/ui/TextState";
import { rgbToCssColor } from "../../../components/utils/color";
import type {
  MallOwnerInfo,
  MallPreviewTransferInfo,
  ScanResultPageState,
} from "../types";
import {
  getNumber,
  getRecord,
  getString,
  getStringArray,
  isRecord,
} from "../utils/guards";
import { createProductBlueprintRows } from "../utils/productBlueprint";
import ScanResultProductSection from "./ScanResultProductSection";
import ScanResultReviewForm from "./ScanResultReviewForm";
import ScanResultReviewList from "./ScanResultReviewList";
import ScanResultTokenSection from "./ScanResultTokenSection";
import ScanResultTransferHistory from "./ScanResultTransferHistory";

type ScanResultCardProps = {
  state: ScanResultPageState;
  transfers: MallPreviewTransferInfo[];
  onRefresh: () => void;
  onPrevReviewsPage: () => void;
  onNextReviewsPage: () => void;
  onOpenTokenContents: (mintAddress: string) => void | Promise<void>;

  reviewBody: string;
  reviewRating: number;
  onReviewBodyChange: (value: string) => void;
  onReviewRatingChange: (rating: number) => void;
  onSubmitReviewForm: () => void | Promise<void>;
  hideReviewForm?: boolean;
};

function mallOwnerInfoFromRecord(
  value: Record<string, unknown> | null
): MallOwnerInfo | null {
  if (!value) {
    return null;
  }

  return {
    brandId: getString(value, "brandId"),
    avatarId: getString(value, "avatarId"),
    brandName: getString(value, "brandName"),
    avatarName: getString(value, "avatarName"),
  };
}

export default function ScanResultCard(props: ScanResultCardProps) {
  const {
    state,
    transfers,
    onRefresh,
    onPrevReviewsPage,
    onNextReviewsPage,
    onOpenTokenContents,
    reviewBody,
    reviewRating,
    onReviewBodyChange,
    onReviewRatingChange,
    onSubmitReviewForm,
    hideReviewForm = false,
  } = props;

  const previewStateRecord = isRecord(state.previewState)
    ? state.previewState
    : null;

  const rawPreview = state.previewState?.raw ?? null;
  const preview = getRecord(rawPreview, "data") ?? rawPreview;

  const productBlueprintPatch = getRecord(preview, "productBlueprintPatch");
  const token = getRecord(preview, "token");

  const tokenBlueprintPatch =
    getRecord(preview, "tokenBlueprintPatch") ??
    getRecord(previewStateRecord, "tokenBlueprintPatch");

  const brandName = getString(preview, "brandName");
  const brandIcon = getString(preview, "brandIcon");

  const productName = getString(productBlueprintPatch, "productName");
  const tokenName = getString(tokenBlueprintPatch, "tokenName");

  const tokenIconUrl =
    getString(tokenBlueprintPatch, "tokenIcon") ||
    getString(previewStateRecord, "tokenIconUrlEncoded");

  const tokenBrandName = getString(tokenBlueprintPatch, "brandName");
  const tokenCompanyName = getString(tokenBlueprintPatch, "companyName");
  const tokenDescription = getString(tokenBlueprintPatch, "description");
  const mintAddress = getString(token, "mintAddress");

  const qualityAssuranceTabs = useMemo(
    () => getStringArray(productBlueprintPatch, "qualityAssurance"),
    [productBlueprintPatch]
  );

  const productBlueprintRows = useMemo(
    () => createProductBlueprintRows(productBlueprintPatch),
    [productBlueprintPatch]
  );

  const measurementEntries = useMemo(() => {
    const measurements = getRecord(preview, "measurements");

    return Object.entries(measurements ?? {})
      .filter(([key]) => key.trim())
      .sort(([a], [b]) => a.localeCompare(b));
  }, [preview]);

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

  if (!isRecord(preview)) {
    return (
      <SectionCard>
        <h1>Scan Result</h1>
        <TextState>プレビューが空です。</TextState>
      </SectionCard>
    );
  }

  const owned = state.ownedByWallet;
  const ownedError = state.ownedByWalletError?.trim() || "";
  const rgb = getNumber(preview, "rgb") ?? 0;
  const swatch = rgbToCssColor(rgb);
  const modelNumber = getString(preview, "modelNumber");
  const productId = getString(preview, "productId");
  const size = getString(preview, "size");
  const color = getString(preview, "color");
  const owner = mallOwnerInfoFromRecord(getRecord(preview, "owner"));

  const title = productName || modelNumber || productId || "Scan Result";

  const canOpenTokenContents =
    owned === true && Boolean(tokenName.trim()) && Boolean(mintAddress.trim());

  const hasTokenInfo =
    Boolean(tokenName.trim()) ||
    Boolean(tokenIconUrl.trim()) ||
    Boolean(tokenBrandName.trim()) ||
    Boolean(tokenCompanyName.trim()) ||
    Boolean(tokenDescription.trim());

  const hasBrandInfo = Boolean(brandName.trim()) || Boolean(brandIcon.trim());

  return (
    <div className="scan-result-desktop-grid">
      <div className="scan-result-desktop-main">
        <ScanResultProductSection
          title={title}
          owned={owned}
          ownedError={ownedError}
          owner={owner}
          brandName={brandName}
          brandIcon={brandIcon}
          hasBrandInfo={hasBrandInfo}
          productBlueprintRows={productBlueprintRows}
          qualityAssuranceTabs={qualityAssuranceTabs}
          modelNumber={modelNumber}
          size={size}
          color={color}
          swatch={swatch}
          measurementEntries={measurementEntries}
        />

        {hasTokenInfo ? (
          <ScanResultTokenSection
            tokenName={tokenName}
            tokenIconUrl={tokenIconUrl}
            tokenBrandName={tokenBrandName}
            tokenCompanyName={tokenCompanyName}
            tokenDescription={tokenDescription}
            mintAddress={mintAddress}
            canOpenTokenContents={canOpenTokenContents}
            onOpenTokenContents={onOpenTokenContents}
          />
        ) : null}

        <ScanResultTransferHistory transfers={transfers} />
      </div>

      <aside className="scan-result-desktop-side">
        {owned === true && !hideReviewForm ? (
          <ScanResultReviewForm
            reviewBody={reviewBody}
            reviewRating={reviewRating}
            postingReview={state.postingReview}
            postReviewError={state.postReviewError}
            onReviewBodyChange={onReviewBodyChange}
            onReviewRatingChange={onReviewRatingChange}
            onSubmit={onSubmitReviewForm}
          />
        ) : null}

        <ScanResultReviewList
          reviews={state.reviews}
          reviewsError={state.reviewsError}
          busyReviews={state.busyReviews}
          reviewPage={state.reviewPage}
          onPrevReviewsPage={onPrevReviewsPage}
          onNextReviewsPage={onNextReviewsPage}
        />
      </aside>
    </div>
  );
}