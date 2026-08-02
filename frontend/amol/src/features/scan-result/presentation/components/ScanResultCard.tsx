// frontend/amol/src/features/scan-result/presentation/components/ScanResultCard.tsx

import { useMemo } from "react";

import Button from "../../../../components/ui/Button";
import SectionCard from "../../../../components/ui/SectionCard";
import TextState from "../../../../components/ui/TextState";
import { rgbToCssColor } from "../../../../components/utils/color";
import { createScanAlcoholInfo } from "../../application/scanAlcoholInfoFactory";
import type {
  MallOwnerInfo,
  ScanResultPageState,
} from "../../../shared/types/scanResult";
import { isRecord } from "../../../../components/utils/typeGuards";
import {
  getNumber,
  getString,
  getStringArray,
} from "../../utils/guards";
import { createProductBlueprintRows } from "../../utils/productBlueprint";
import ScanResultProductSection from "./ScanResultProductSection";
import ScanResultReviewForm from "./ScanResultReviewForm";
import ScanResultReviewList from "./ScanResultReviewList";
import ScanResultTokenSection from "./ScanResultTokenSection";

type ScanResultCardProps = {
  state: ScanResultPageState;
  onRefresh: () => void;
  onPrevReviewsPage: () => void;
  onNextReviewsPage: () => void;
  onOpenTokenContents: (
    mintAddress: string,
  ) => void | Promise<void>;

  reviewBody: string;
  reviewRating: number;
  onReviewBodyChange: (value: string) => void;
  onReviewRatingChange: (rating: number) => void;
  onSubmitReviewForm: () => void | Promise<void>;
  hideReviewForm?: boolean;
};

function mallOwnerInfoFromRecord(
  value: Record<string, unknown> | null,
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

export default function ScanResultCard(
  props: ScanResultCardProps,
) {
  const {
    state,
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

  const previewStateRecord =
    isRecord(state.previewState) &&
    !Array.isArray(state.previewState)
      ? state.previewState
      : null;

  const preview =
    state.previewState?.raw ?? null;

  const rawProductBlueprintPatch =
    isRecord(preview) &&
    !Array.isArray(preview)
      ? preview.productBlueprintPatch
      : null;

  const productBlueprintPatch =
    isRecord(rawProductBlueprintPatch) &&
    !Array.isArray(rawProductBlueprintPatch)
      ? rawProductBlueprintPatch
      : null;

  const rawCategoryFields =
    productBlueprintPatch?.categoryFields;

  const categoryFields =
    isRecord(rawCategoryFields) &&
    !Array.isArray(rawCategoryFields)
      ? rawCategoryFields
      : null;

  const rawToken =
    isRecord(preview) &&
    !Array.isArray(preview)
      ? preview.token
      : null;

  const token =
    isRecord(rawToken) &&
    !Array.isArray(rawToken)
      ? rawToken
      : null;

  const rawPreviewTokenBlueprintPatch =
    isRecord(preview) &&
    !Array.isArray(preview)
      ? preview.tokenBlueprintPatch
      : null;

  const previewTokenBlueprintPatch =
    isRecord(rawPreviewTokenBlueprintPatch) &&
    !Array.isArray(rawPreviewTokenBlueprintPatch)
      ? rawPreviewTokenBlueprintPatch
      : null;

  const rawStateTokenBlueprintPatch =
    previewStateRecord?.tokenBlueprintPatch;

  const stateTokenBlueprintPatch =
    isRecord(rawStateTokenBlueprintPatch) &&
    !Array.isArray(rawStateTokenBlueprintPatch)
      ? rawStateTokenBlueprintPatch
      : null;

  const tokenBlueprintPatch =
    previewTokenBlueprintPatch ??
    stateTokenBlueprintPatch;

  const brandId =
    getString(preview, "brandId") ||
    getString(
      productBlueprintPatch,
      "brandId",
    ) ||
    getString(token, "brandId");

  const brandName =
    getString(preview, "brandName") ||
    getString(token, "brandName") ||
    getString(
      tokenBlueprintPatch,
      "brandName",
    );

  const productName = getString(
    productBlueprintPatch,
    "productName",
  );

  const tokenName = getString(
    tokenBlueprintPatch,
    "tokenName",
  );

  const tokenIconUrl =
    getString(
      tokenBlueprintPatch,
      "tokenIcon",
    ) ||
    getString(
      previewStateRecord,
      "tokenIconUrlEncoded",
    );

  const tokenBrandName = getString(
    tokenBlueprintPatch,
    "brandName",
  );

  const tokenCompanyName = getString(
    tokenBlueprintPatch,
    "companyName",
  );

  const tokenDescription = getString(
    tokenBlueprintPatch,
    "description",
  );

  const mintAddress = getString(
    token,
    "mintAddress",
  );

  const qualityAssuranceTabs = useMemo(
    () =>
      getStringArray(
        productBlueprintPatch,
        "qualityAssurance",
      ),
    [productBlueprintPatch],
  );

  const productBlueprintRows = useMemo(
    () =>
      createProductBlueprintRows(
        productBlueprintPatch,
      ),
    [productBlueprintPatch],
  );

  const measurementEntries = useMemo(() => {
    const rawMeasurements =
      isRecord(preview) &&
      !Array.isArray(preview)
        ? preview.measurements
        : null;

    const measurements =
      isRecord(rawMeasurements) &&
      !Array.isArray(rawMeasurements)
        ? rawMeasurements
        : null;

    return Object.entries(
      measurements ?? {},
    )
      .filter(([key]) => Boolean(key))
      .sort(([a], [b]) =>
        a.localeCompare(b),
      );
  }, [preview]);

  const alcoholInfo = useMemo(() => {
    const rawProductBlueprintCategory =
      isRecord(preview) &&
      !Array.isArray(preview)
        ? preview.productBlueprintCategory
        : null;

    const productBlueprintCategory =
      isRecord(rawProductBlueprintCategory) &&
      !Array.isArray(
        rawProductBlueprintCategory,
      )
        ? rawProductBlueprintCategory
        : null;

    const rawCategoryInputSchema =
      isRecord(preview) &&
      !Array.isArray(preview)
        ? preview.categoryInputSchema
        : null;

    const categoryInputSchema =
      isRecord(rawCategoryInputSchema) &&
      !Array.isArray(rawCategoryInputSchema)
        ? rawCategoryInputSchema
        : null;

    return createScanAlcoholInfo({
      categoryFields,
      volumeValue: getNumber(
        preview,
        "volumeValue",
      ),
      volumeUnit: getString(
        preview,
        "volumeUnit",
      ),
      modelLabel: getString(
        preview,
        "modelLabel",
      ),
      modelKind: getString(
        preview,
        "modelKind",
      ),
      productBlueprintCategoryKind:
        getString(
          preview,
          "productBlueprintCategoryKind",
        ),
      productBlueprintCategory,
      categoryInputSchema,
    });
  }, [categoryFields, preview]);

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

        <TextState variant="error">
          {state.error}
        </TextState>

        <Button
          type="button"
          onClick={onRefresh}
        >
          再読み込み
        </Button>
      </SectionCard>
    );
  }

  if (
    !isRecord(preview) ||
    Array.isArray(preview)
  ) {
    return (
      <SectionCard>
        <h1>Scan Result</h1>

        <TextState>
          プレビューが空です。
        </TextState>
      </SectionCard>
    );
  }

  const owned = state.ownedByWallet;
  const ownedError =
    state.ownedByWalletError || "";

  const rgb =
    getNumber(preview, "rgb") ?? 0;

  const swatch = rgbToCssColor(rgb);

  const modelNumber = getString(
    preview,
    "modelNumber",
  );

  const productId = getString(
    preview,
    "productId",
  );

  const size = getString(
    preview,
    "size",
  );

  const color = getString(
    preview,
    "color",
  );

  const rawOwner = preview.owner;

  const ownerRecord =
    isRecord(rawOwner) &&
    !Array.isArray(rawOwner)
      ? rawOwner
      : null;

  const owner =
    mallOwnerInfoFromRecord(
      ownerRecord,
    );

  const title =
    productName ||
    modelNumber ||
    productId ||
    "Scan Result";

  const canOpenTokenContents =
    owned === true &&
    Boolean(tokenName) &&
    Boolean(mintAddress);

  const hasTokenInfo =
    Boolean(tokenName) ||
    Boolean(tokenIconUrl) ||
    Boolean(tokenBrandName) ||
    Boolean(tokenCompanyName) ||
    Boolean(tokenDescription);

  const hasBrandInfo = Boolean(
    brandId || brandName,
  );

  return (
    <div className="scan-result-desktop-grid">
      <div className="scan-result-desktop-main">
        <ScanResultProductSection
          title={title}
          owned={owned}
          ownedError={ownedError}
          owner={owner}
          brandId={brandId}
          brandName={brandName}
          hasBrandInfo={hasBrandInfo}
          productBlueprintRows={
            productBlueprintRows
          }
          qualityAssuranceTabs={
            qualityAssuranceTabs
          }
          modelNumber={modelNumber}
          size={size}
          color={color}
          swatch={swatch}
          measurementEntries={
            measurementEntries
          }
          alcoholInfo={alcoholInfo}
        />

        {hasTokenInfo ? (
          <ScanResultTokenSection
            tokenName={tokenName}
            tokenIconUrl={tokenIconUrl}
            tokenBrandName={
              tokenBrandName
            }
            tokenCompanyName={
              tokenCompanyName
            }
            tokenDescription={
              tokenDescription
            }
            mintAddress={mintAddress}
            canOpenTokenContents={
              canOpenTokenContents
            }
            onOpenTokenContents={
              onOpenTokenContents
            }
          />
        ) : null}
      </div>

      <aside className="scan-result-desktop-side">
        {owned === true &&
        !hideReviewForm ? (
          <ScanResultReviewForm
            reviewBody={reviewBody}
            reviewRating={reviewRating}
            postingReview={
              state.postingReview
            }
            postReviewError={
              state.postReviewError
            }
            onReviewBodyChange={
              onReviewBodyChange
            }
            onReviewRatingChange={
              onReviewRatingChange
            }
            onSubmit={
              onSubmitReviewForm
            }
          />
        ) : null}

        <ScanResultReviewList
          reviews={state.reviews}
          reviewsError={
            state.reviewsError
          }
          busyReviews={
            state.busyReviews
          }
          reviewPage={
            state.reviewPage
          }
          onPrevReviewsPage={
            onPrevReviewsPage
          }
          onNextReviewsPage={
            onNextReviewsPage
          }
        />
      </aside>
    </div>
  );
}