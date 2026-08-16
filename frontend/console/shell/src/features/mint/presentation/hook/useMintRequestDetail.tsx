// frontend/console/shell/src/features/mint/presentation/hook/useMintRequestDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import type { TokenBlueprintSummary } from "../../infrastructure/dto/MintRequestRepository";
import { buildInspectionResultCardData } from "../../application/mapper/buildInspectionResultCardData";
import { completeMintInspection } from "../../application/usecase/completeMintInspection";
import { submitMintRequest } from "../../application/usecase/submitMintRequest";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";
import { useBrandSelection } from "../../../brand/presentation/hook/useBrandSelection";
import type { MintRequestManagementRowDTO } from "../../infrastructure/dto/mintRequestManagementRow";
import type { MintProductBlueprintDTO, MintRequestDetailDTO } from "../../infrastructure/dto/mintRequestLocal.dto";
import { completeInspectionHTTP, fetchMintRequestDetailHTTP } from "../../infrastructure/repository/inspections";
import {
  fetchMintRequestRowByProductionIdHTTP,
  postMintRequestHTTP,
} from "../../infrastructure/repository/mintRequests";
import { fetchMintProductBlueprintHTTP } from "../../infrastructure/repository/mintProductBlueprint";
import { fetchTokenBlueprintsByBrandHTTP } from "../../infrastructure/repository/tokenBlueprints";
import { buildProductBlueprintCardView, buildTokenBlueprintCardVm } from "../viewModel/mintRequestDetailViewModel";
import { useMintAutoSelection } from "./useMintRequestDetail.useMintAutoSelection";
import { useMintFundingEstimate } from "./useMintFundingEstimate";

function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}

const MINT_REQUEST_MANAGEMENT_PATH = "/mint";

export function useMintRequestDetail() {
  const navigate = useNavigate();
  const { requestId } = useParams<{ requestId: string }>();
  const productionId = React.useMemo(() => String(requestId ?? "").trim(), [requestId]);

  const [mintRequestDetail, setMintRequestDetail] = React.useState<MintRequestDetailDTO | null>(null);
  const [mintRequestRow, setMintRequestRow] = React.useState<MintRequestManagementRowDTO | null>(null);
  const inspectionBatch = mintRequestDetail?.inspection ?? null;
  const productBlueprintId = mintRequestDetail?.productBlueprintId ?? "";

  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [productBlueprint, setProductBlueprint] = React.useState<MintProductBlueprintDTO | null>(null);
  const [productBlueprintLoading, setProductBlueprintLoading] = React.useState(false);
  const [productBlueprintError, setProductBlueprintError] = React.useState<string | null>(null);

  const {
    brandId: selectedBrandId,
    brandName: selectedBrandName,
    brandOptions,
    loadingBrands,
    brandError,
    selectBrand,
  } = useBrandSelection();

  const [tokenBlueprintOptions, setTokenBlueprintOptions] = React.useState<TokenBlueprintSummary[]>([]);
  const [selectedTokenBlueprintId, setSelectedTokenBlueprintId] = React.useState("");
  const [isSubmittingMintRequest, setIsSubmittingMintRequest] = React.useState(false);
  const [isCompletingInspection, setIsCompletingInspection] = React.useState(false);

  const title = "ミント申請詳細";

  const reloadDetail = React.useCallback(async () => {
    if (!productionId) return;

    const [detail, row] = await Promise.all([
      fetchMintRequestDetailHTTP(productionId),
      fetchMintRequestRowByProductionIdHTTP(productionId),
    ]);

    setMintRequestDetail(detail);
    setMintRequestRow(row);
  }, [productionId]);

  const reloadMintStatus = React.useCallback(async () => {
    if (!productionId) return;

    const row = await fetchMintRequestRowByProductionIdHTTP(productionId);
    setMintRequestRow(row);
  }, [productionId]);

  React.useEffect(() => {
    if (!productionId) {
      setMintRequestDetail(null);
      setMintRequestRow(null);
      return;
    }

    let cancelled = false;

    const run = async () => {
      setLoading(true);
      setError(null);

      try {
        const [detail, row] = await Promise.all([
          fetchMintRequestDetailHTTP(productionId),
          fetchMintRequestRowByProductionIdHTTP(productionId),
        ]);

        if (cancelled) return;

        setMintRequestDetail(detail);
        setMintRequestRow(row);
      } catch (error: unknown) {
        if (!cancelled) {
          setError(getErrorMessage(error, "検査結果の取得に失敗しました"));
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    void run();

    return () => {
      cancelled = true;
    };
  }, [productionId]);

  React.useEffect(() => {
    if (!productBlueprintId) {
      setProductBlueprint(null);
      setProductBlueprintError(null);
      return;
    }

    let cancelled = false;

    const run = async () => {
      setProductBlueprintLoading(true);
      setProductBlueprintError(null);

      try {
        const result = await fetchMintProductBlueprintHTTP(productBlueprintId);
        if (!cancelled) setProductBlueprint(result);
      } catch (error: unknown) {
        if (!cancelled) {
          setProductBlueprintError(
            getErrorMessage(error, "プロダクト基本情報の取得に失敗しました"),
          );
        }
      } finally {
        if (!cancelled) setProductBlueprintLoading(false);
      }
    };

    void run();

    return () => {
      cancelled = true;
    };
  }, [productBlueprintId]);

  const inspectionCardData = React.useMemo(
    () =>
      buildInspectionResultCardData({
        inspection: inspectionBatch,
        productName: mintRequestDetail?.productName ?? "",
        modelMeta: mintRequestDetail?.modelMeta ?? {},
        productBlueprint,
      }),
    [inspectionBatch, mintRequestDetail, productBlueprint],
  );

  const totalMintQuantity = mintRequestRow?.mintQuantity ?? 0;

  const productBlueprintCardView = React.useMemo(
    () => buildProductBlueprintCardView(productBlueprint),
    [productBlueprint],
  );

  const onBack = React.useCallback(() => {
    navigate(MINT_REQUEST_MANAGEMENT_PATH);
  }, [navigate]);

  const handleSelectBrand = React.useCallback(
    async (brandId: string) => {
      selectBrand(brandId);

      if (!brandId) {
        setTokenBlueprintOptions([]);
        setSelectedTokenBlueprintId("");
        return;
      }

      try {
        const options = await fetchTokenBlueprintsByBrandHTTP(brandId);
        setTokenBlueprintOptions(options);
        setSelectedTokenBlueprintId("");
      } catch {
        setTokenBlueprintOptions([]);
        setSelectedTokenBlueprintId("");
      }
    },
    [selectBrand],
  );

  const mintStatus = mintRequestRow?.mintStatus ?? null;
  const mintProgress = mintRequestRow?.mintProgress ?? null;
  const hasMint = Boolean(mintStatus);
  const isMintProcessing =
    mintStatus === "QUEUED" ||
    mintStatus === "MINTING" ||
    mintStatus === "PARTIALLY_MINTED" ||
    mintStatus === "FAILED_RETRYABLE";
  const isMintCompleted = mintStatus === "MINTED";
  const createdByName = mintRequestRow?.createdByName ?? null;
  const requestedByName = mintRequestRow?.requestedByName ?? null;
  const mintRequestedTokenBlueprintId = mintRequestRow?.tokenBlueprintId ?? "";
  const mintRequestedBrandId = productBlueprint?.brandId ?? "";
  const isMinting = isSubmittingMintRequest || isMintProcessing;

  React.useEffect(() => {
    if (!productionId || !isMintProcessing) return;

    let cancelled = false;

    const timer = window.setInterval(() => {
      if (cancelled) return;
      void reloadMintStatus().catch(() => {});
    }, 3000);

    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [productionId, isMintProcessing, reloadMintStatus]);

  const inspectionStatus = inspectionBatch?.status ?? "";
  const isInspectionCompleted = inspectionStatus === "completed";

  const showCompleteInspectionButton = React.useMemo(
    () =>
      Boolean(
        inspectionBatch &&
          !loading &&
          !error &&
          !isMinting &&
          !isMintCompleted &&
          !isInspectionCompleted,
      ),
    [
      inspectionBatch,
      loading,
      error,
      isMinting,
      isMintCompleted,
      isInspectionCompleted,
    ],
  );

  const showMintControls = isInspectionCompleted && !isMinting && !isMintCompleted;

  useMintAutoSelection({
    hasMint,
    mintRequestedBrandId,
    selectedBrandId,
    handleSelectBrand,
    mintRequestedTokenBlueprintId,
    selectedTokenBlueprintId,
    setSelectedTokenBlueprintId,
  });

  const {
    estimate: mintFundingEstimate,
    loading: mintFundingEstimateLoading,
    error: mintFundingEstimateError,
  } = useMintFundingEstimate({
    productionId,
    tokenBlueprintId: selectedTokenBlueprintId,
    enabled: showMintControls,
  });

  const displayTokenBlueprintId = React.useMemo(
    () => selectedTokenBlueprintId || mintRequestedTokenBlueprintId,
    [selectedTokenBlueprintId, mintRequestedTokenBlueprintId],
  );

  const handleCompleteInspection = React.useCallback(async () => {
    if (isCompletingInspection || isMinting || isMintCompleted) return;

    const confirmed = window.confirm(
      "検品を完了します。未入力の検品結果は合格として確定されます。よろしいですか？",
    );
    if (!confirmed) return;

    setIsCompletingInspection(true);

    try {
      const result = await completeMintInspection(
        { completeInspection: completeInspectionHTTP },
        { inspectionBatch },
      );

      if (!result.ok) {
        alert(result.message);
        return;
      }

      await reloadDetail();
      alert("検品を完了しました。");
    } catch (error: unknown) {
      alert(
        `検品完了に失敗しました: ${getErrorMessage(
          error,
          "不明なエラーが発生しました",
        )}`,
      );
    } finally {
      setIsCompletingInspection(false);
    }
  }, [
    inspectionBatch,
    isCompletingInspection,
    isMinting,
    isMintCompleted,
    reloadDetail,
  ]);

  const handleMint = React.useCallback(async () => {
    if (!isInspectionCompleted || isMinting || isMintCompleted) return;

    if (mintFundingEstimateLoading) {
      alert("SOL見積を取得中です。完了後にミント申請を実行してください。");
      return;
    }

    if (!mintFundingEstimate) {
      alert(mintFundingEstimateError || "SOL見積を取得できていません。");
      return;
    }

    if (!mintFundingEstimate.estimate.sufficient) {
      alert("Reserve WalletのSOL残高が不足しているため、ミント申請を実行できません。");
      return;
    }

    setIsSubmittingMintRequest(true);
    setError(null);

    try {
      const result = await submitMintRequest(
        { postMintRequest: postMintRequestHTTP },
        {
          inspectionBatch,
          selectedTokenBlueprintId,
        },
      );

      if (!result.ok) {
        if (result.reason === "validation") {
          alert(result.message);
          return;
        }

        setError(result.message);
        alert(`ミント申請に失敗しました: ${result.message}`);

        try {
          await reloadDetail();
        } catch {}

        return;
      }

      const { queuedResponse } = result;

      await reloadDetail();

      alert(
        `ミント申請を受け付けました（生産ID: ${queuedResponse.productionId} / ミント数: ${totalMintQuantity}）。順次ミント処理を実行します。`,
      );
    } catch (error: unknown) {
      const message = getErrorMessage(error, "不明なエラーが発生しました");
      setError(message);
      alert(`ミント申請に失敗しました: ${message}`);

      try {
        await reloadDetail();
      } catch {}
    } finally {
      setIsSubmittingMintRequest(false);
    }
  }, [
    inspectionBatch,
    isInspectionCompleted,
    isMinting,
    isMintCompleted,
    mintFundingEstimate,
    mintFundingEstimateError,
    mintFundingEstimateLoading,
    reloadDetail,
    selectedTokenBlueprintId,
    totalMintQuantity,
  ]);

  const handleSelectTokenBlueprint = React.useCallback((tokenBlueprintId: string) => {
    setSelectedTokenBlueprintId(tokenBlueprintId);
  }, []);

  const selectedTokenBlueprint = React.useMemo(
    () =>
      tokenBlueprintOptions.find(
        (tokenBlueprint) => tokenBlueprint.id === selectedTokenBlueprintId,
      ) ?? null,
    [tokenBlueprintOptions, selectedTokenBlueprintId],
  );

  const tokenBlueprintCardVm = React.useMemo(
    () =>
      buildTokenBlueprintCardVm({
        selectedTokenBlueprint,
        displayTokenBlueprintId,
        selectedBrandName,
        productBlueprint,
        brandOptions,
      }),
    [
      selectedTokenBlueprint,
      displayTokenBlueprintId,
      selectedBrandName,
      productBlueprint,
      brandOptions,
    ],
  );

  const mintCreatedAtLabel = "（未登録）";
  const mintCreatedByLabel = createdByName ?? "（不明）";
  const mintMintedAtLabel = safeDateTimeLabelJa(
    mintRequestRow?.mintedAt ?? null,
    "（未完了）",
  );
  const onChainTxSignature = "";

  return {
    title,
    loading,
    error,
    inspectionCardData,
    mintRequestRow,
    mintStatus,
    mintProgress,
    totalMintQuantity,
    onBack,
    handleMint,
    isMinting,
    hasMint,
    isMintCompleted,
    isInspectionCompleted,
    showMintButton: showMintControls,
    showBrandSelectorCard: showMintControls,
    showTokenSelectorCard: showMintControls,
    showCompleteInspectionButton,
    isCompletingInspection,
    handleCompleteInspection,
    requestedByName,
    productBlueprintCardView,
    productBlueprintLoading,
    productBlueprintError,
    brandOptions,
    selectedBrandId,
    selectedBrandName,
    loadingBrands,
    brandError,
    handleSelectBrand,
    tokenBlueprintOptions,
    selectedTokenBlueprintId,
    handleSelectTokenBlueprint,
    tokenBlueprintCardVm,
    mintCreatedAtLabel,
    mintCreatedByLabel,
    mintMintedAtLabel,
    onChainTxSignature,
    mintFundingEstimate,
    mintFundingEstimateLoading,
    mintFundingEstimateError,
  };
}