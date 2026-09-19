// frontend/console/shell/src/pages/mintDetail.tsx

import PageStyle from "../layout/PageStyle/PageStyle";
import { Card, CardContent } from "../shared/ui/card";
import { ErrorMessage } from "../shared/ui/error";
import Text from "../shared/ui/text";

import MintBrandSelectorCard from "../features/mint/presentation/components/mintBrandSelectorCard";
import MintFundingEstimateCard from "../features/mint/presentation/components/mintFundingEstimateCard";
import MintInfoCard from "../features/mint/presentation/components/mintInfoCard";
import MintInspectionCompleteCard from "../features/mint/presentation/components/mintInspectionCompleteCard";
import InspectionResultCard from "../features/mint/presentation/components/inspectionResultCard";
import MintProgressCard from "../features/mint/presentation/components/mintProgressCard";
import MintTokenBlueprintSelectorCard from "../features/mint/presentation/components/mintTokenBlueprintSelectorCard";
import { useMintRequestDetail } from "../features/mint/presentation/hook/useMintRequestDetail";
import ProductBlueprintCard from "../features/productBlueprint/presentation/components/productBlueprintForm";
import TokenBlueprintCard from "../features/tokenBlueprint/presentation/components/tokenBlueprintCard";

import "../styles/mintRequest.css";

export default function MintRequestDetail() {
  const {
    title,
    loading,
    error,
    inspectionCardData,
    mintRequestRow,
    mintStatus,
    mintProgress,
    onBack,
    handleMint,
    isMinting,
    canSubmitMint,
    hasMint,
    productBlueprintCardView,
    productBlueprintLoading,
    productBlueprintError,
    brandOptions,
    selectedBrandId,
    selectedBrandName,
    handleSelectBrand,
    tokenBlueprintOptions,
    selectedTokenBlueprintId,
    handleSelectTokenBlueprint,
    showMintButton,
    showBrandSelectorCard,
    showTokenSelectorCard,
    showCompleteInspectionButton,
    isCompletingInspection,
    handleCompleteInspection,
    tokenBlueprintCardVm,
    mintMintedAtLabel,
    mintFundingEstimate,
    mintFundingEstimateLoading,
    mintFundingEstimateError,
  } = useMintRequestDetail();

  return (
    <PageStyle layout="grid-2" title={title} onBack={onBack}>
      <div className="page-column page-column--offset-top">
        {productBlueprintLoading ? (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body">
              <Text tone="muted">プロダクト基本情報を読み込み中です…</Text>
            </CardContent>
          </Card>
        ) : productBlueprintError ? (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body">
              <ErrorMessage>
                {productBlueprintError}
              </ErrorMessage>
            </CardContent>
          </Card>
        ) : productBlueprintCardView ? (
          <ProductBlueprintCard
            mode="view"
            productName={productBlueprintCardView.productName}
            brandName={productBlueprintCardView.brandName}
            productBlueprintCategoryPath={
              productBlueprintCardView.productBlueprintCategoryPath ?? null
            }
          />
        ) : (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body">
              <Text tone="muted">プロダクト基本情報を読み込み中です…</Text>
            </CardContent>
          </Card>
        )}

        {loading ? (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body">
              <Text tone="muted">検査結果を読み込み中です…</Text>
            </CardContent>
          </Card>
        ) : error ? (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body">
              <ErrorMessage>
                {error}
              </ErrorMessage>
            </CardContent>
          </Card>
        ) : (
          <>
            <InspectionResultCard data={inspectionCardData} />

            {showCompleteInspectionButton && (
              <MintInspectionCompleteCard
                completing={isCompletingInspection}
                disabled={isMinting}
                onComplete={handleCompleteInspection}
              />
            )}
          </>
        )}

        {tokenBlueprintCardVm && (
          <TokenBlueprintCard vm={tokenBlueprintCardVm} />
        )}

        {showMintButton && (
          <MintFundingEstimateCard
            selectedTokenBlueprintId={selectedTokenBlueprintId}
            estimate={mintFundingEstimate}
            loading={mintFundingEstimateLoading}
            error={mintFundingEstimateError}
            canSubmit={canSubmitMint}
            onMint={handleMint}
          />
        )}
      </div>

      <div className="page-column page-column--offset-top">
        {hasMint && mintRequestRow && (
          <MintInfoCard
            mintRequestRow={mintRequestRow}
            mintedAtLabel={mintMintedAtLabel}
          />
        )}

        {hasMint && mintStatus !== "MINTED" && mintProgress && (
          <MintProgressCard progress={mintProgress} />
        )}

        {showBrandSelectorCard && (
          <MintBrandSelectorCard
            brandOptions={brandOptions}
            selectedBrandId={selectedBrandId}
            selectedBrandName={selectedBrandName}
            disabled={isMinting}
            onSelectBrand={handleSelectBrand}
          />
        )}

        {showTokenSelectorCard && (
          <MintTokenBlueprintSelectorCard
            selectedBrandId={selectedBrandId}
            tokenBlueprintOptions={tokenBlueprintOptions}
            selectedTokenBlueprintId={selectedTokenBlueprintId}
            disabled={isMinting}
            onSelectTokenBlueprint={handleSelectTokenBlueprint}
          />
        )}
      </div>
    </PageStyle>
  );
}