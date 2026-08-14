// frontend/console/shell/src/features/mint/presentation/hook/useMintRequestDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import type {
  BrandSummary,
  MintFundingEstimate,
  TokenBlueprintSummary,
} from "../../application/port/MintRequestRepository";

import { completeMintInspection } from "../../application/usecase/completeMintInspection";
import { getMintRequestDetail } from "../../application/usecase/getMintRequestDetail";
import { getMintProductBlueprint } from "../../application/usecase/getMintProductBlueprint";
import { submitMintRequest } from "../../application/usecase/submitMintRequest";

import type { InspectionBatchDTO } from "../../../../shared/types/inspections";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";

import type { MintRequestManagementRowDTO } from "../../infrastructure/dto/mintRequestManagementRow";
import type {
  MintProductBlueprintDTO,
  MintRequestDetailDTO,
} from "../../infrastructure/dto/mintRequestLocal.dto";

import { HttpMintRequestRepository } from "../../infrastructure/repository/HttpMintRequestRepository";

import {
  buildProductBlueprintCardView,
  buildTokenBlueprintCardVm,
} from "../viewModel/mintRequestDetailViewModel";

import { useInspectionResultCard } from "./useInspectionResultCard";
import { useMintAutoSelection } from "./useMintRequestDetail.useMintAutoSelection";

function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}

/**
 * GET /mint/inspections/{productionId} のBFF responseを、
 * Presentationで使用するInspectionBatchDTOへ変換する。
 *
 * model表示情報はdetail.modelMetaをそのまま使用し、
 * Frontend側でModel Variation APIなどによる補完は行わない。
 */
function buildInspectionBatch(
  detail: MintRequestDetailDTO | null,
): InspectionBatchDTO | null {
  if (!detail?.inspection) return null;

  return {
    ...detail.inspection,
    productBlueprintId: detail.productBlueprintId ?? "",
    productName: detail.productName,
    modelMeta: detail.modelMeta ?? {},
  };
}

const MINT_REQUEST_MANAGEMENT_PATH = "/mint";

export function useMintRequestDetail() {
  const navigate = useNavigate();

  /**
   * route名はrequestIdのままでも、実体はproductionIdとして扱う。
   */
  const { requestId } = useParams<{ requestId: string }>();
  const productionId = React.useMemo(
    () => String(requestId ?? "").trim(),
    [requestId],
  );

  const mintRequestRepo = React.useMemo(
    () => new HttpMintRequestRepository(),
    [],
  );

  const [inspectionBatch, setInspectionBatch] =
    React.useState<InspectionBatchDTO | null>(null);

  const [mintRequestRow, setMintRequestRow] =
    React.useState<MintRequestManagementRowDTO | null>(null);

  const [productBlueprintId, setProductBlueprintId] = React.useState("");
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const [productBlueprint, setProductBlueprint] =
    React.useState<MintProductBlueprintDTO | null>(null);

  const [productBlueprintLoading, setProductBlueprintLoading] =
    React.useState(false);

  const [productBlueprintError, setProductBlueprintError] =
    React.useState<string | null>(null);

  const [brandOptions, setBrandOptions] =
    React.useState<BrandSummary[]>([]);

  const [selectedBrandId, setSelectedBrandId] = React.useState("");

  const [tokenBlueprintOptions, setTokenBlueprintOptions] =
    React.useState<TokenBlueprintSummary[]>([]);

  const [selectedTokenBlueprintId, setSelectedTokenBlueprintId] =
    React.useState("");

  const [mintFundingEstimate, setMintFundingEstimate] =
    React.useState<MintFundingEstimate | null>(null);

  const [mintFundingEstimateLoading, setMintFundingEstimateLoading] =
    React.useState(false);

  const [mintFundingEstimateError, setMintFundingEstimateError] =
    React.useState<string | null>(null);

  const [isSubmittingMintRequest, setIsSubmittingMintRequest] =
    React.useState(false);

  const [isCompletingInspection, setIsCompletingInspection] =
    React.useState(false);

  const title = "ミント申請詳細";

  const selectedBrandName = React.useMemo(() => {
    if (!selectedBrandId) return "";

    return (
      brandOptions.find(
        (brand) => brand.id === selectedBrandId,
      )?.name ?? ""
    );
  }, [brandOptions, selectedBrandId]);

  /**
   * inspection detailとMint status rowをまとめて再取得する。
   *
   * inspection detail:
   * GET /mint/inspections/{productionId}
   *
   * Mint status:
   * GET /mint/requests?productionIds={productionId}
   */
  const reloadDetail = React.useCallback(async () => {
    if (!productionId) return;

    const [detail, row] = await Promise.all([
      getMintRequestDetail(
        mintRequestRepo,
        productionId,
      ),
      mintRequestRepo.fetchMintRequestRowByProductionId(
        productionId,
      ),
    ]);

    setInspectionBatch(buildInspectionBatch(detail));
    setMintRequestRow(row);
    setProductBlueprintId(detail?.productBlueprintId ?? "");
  }, [mintRequestRepo, productionId]);

  /**
   * Mint状態だけを再取得する。
   * ミント中の周期処理ではinspection detailを再取得しない。
   */
  const reloadMintStatus = React.useCallback(async () => {
    if (!productionId) return;

    const row =
      await mintRequestRepo.fetchMintRequestRowByProductionId(
        productionId,
      );

    setMintRequestRow(row);
  }, [mintRequestRepo, productionId]);

  React.useEffect(() => {
    if (!productionId) {
      setInspectionBatch(null);
      setMintRequestRow(null);
      setProductBlueprintId("");
      return;
    }

    let cancelled = false;

    const run = async () => {
      setLoading(true);
      setError(null);

      try {
        const [detail, row] = await Promise.all([
          getMintRequestDetail(
            mintRequestRepo,
            productionId,
          ),
          mintRequestRepo.fetchMintRequestRowByProductionId(
            productionId,
          ),
        ]);

        if (cancelled) return;

        setInspectionBatch(buildInspectionBatch(detail));
        setMintRequestRow(row);
        setProductBlueprintId(detail?.productBlueprintId ?? "");
      } catch (error: unknown) {
        if (!cancelled) {
          setError(
            getErrorMessage(
              error,
              "検査結果の取得に失敗しました",
            ),
          );
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    void run();

    return () => {
      cancelled = true;
    };
  }, [mintRequestRepo, productionId]);

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
        const result = await getMintProductBlueprint(
          mintRequestRepo,
          productBlueprintId,
        );

        if (!cancelled) setProductBlueprint(result);
      } catch (error: unknown) {
        if (!cancelled) {
          setProductBlueprintError(
            getErrorMessage(
              error,
              "プロダクト基本情報の取得に失敗しました",
            ),
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
  }, [mintRequestRepo, productBlueprintId]);

  const batchForInspectionCard = React.useMemo(() => {
    if (!inspectionBatch) return undefined;

    /**
     * modelMetaはGET /mint/inspections/{productionId} の
     * Backend responseをそのまま正として使用する。
     */
    return {
      ...inspectionBatch,
      productBlueprint,
    };
  }, [inspectionBatch, productBlueprint]);

  /**
   * Model情報はBackend responseのmodelMetaを正とする。
   * FrontendからGET /models/{modelId}による個別補完は行わない。
   */
  const inspectionCardData = useInspectionResultCard({
    batch: batchForInspectionCard,
  });

  /**
   * BackendのGET /mint/requests responseのmintQuantityを正とする。
   * row未取得時のみ画面初期値として0を使用する。
   */
  const totalMintQuantity =
    mintRequestRow?.mintQuantity ?? 0;

  const productBlueprintCardView = React.useMemo(
    () => buildProductBlueprintCardView(productBlueprint),
    [productBlueprint],
  );

  const onBack = React.useCallback(() => {
    navigate(MINT_REQUEST_MANAGEMENT_PATH);
  }, [navigate]);

  React.useEffect(() => {
    let cancelled = false;

    const run = async () => {
      try {
        const brands =
          await mintRequestRepo.fetchBrandsForMint();

        if (!cancelled) setBrandOptions(brands);
      } catch {
        if (!cancelled) setBrandOptions([]);
      }
    };

    void run();

    return () => {
      cancelled = true;
    };
  }, [mintRequestRepo]);

  const handleSelectBrand = React.useCallback(
    async (brandId: string) => {
      setSelectedBrandId(brandId);

      if (!brandId) {
        setTokenBlueprintOptions([]);
        setSelectedTokenBlueprintId("");
        return;
      }

      try {
        const options =
          await mintRequestRepo.fetchTokenBlueprintsByBrand(
            brandId,
          );

        setTokenBlueprintOptions(options);
        setSelectedTokenBlueprintId("");
      } catch {
        setTokenBlueprintOptions([]);
        setSelectedTokenBlueprintId("");
      }
    },
    [mintRequestRepo],
  );

  /**
   * GET /mint/requests responseのmintStatusをMint状態の唯一の正とする。
   * Frontendではtrim / uppercaseなどの再正規化を行わない。
   */
  const mintStatus =
    mintRequestRow?.mintStatus ?? null;

  /**
   * BackendはMintが存在しないProductionについてもrowを返し得るため、
   * rowの存在そのものではhasMintを判定せず、mintStatusの有無を正とする。
   */
  const hasMint = Boolean(mintStatus);

  /**
   * 非同期Mint処理中としてポーリングする状態。
   */
  const isMintProcessing =
    mintStatus === "QUEUED" ||
    mintStatus === "MINTING" ||
    mintStatus === "PARTIALLY_MINTED";

  const isMintCompleted =
    mintStatus === "MINTED";

  /**
   * Backendが解決済みの表示名をそのまま使用する。
   * memberIdへのfallbackは行わない。
   */
  const createdByName =
    mintRequestRow?.createdByName ?? null;

  const requestedByName =
    mintRequestRow?.requestedByName ?? null;

  const mintRequestedTokenBlueprintId =
    mintRequestRow?.tokenBlueprintId ?? "";

  /**
   * 現行GET /mint/requests responseにはbrandIdがないため、
   * ProductBlueprintのbrandIdを使用する。
   */
  const mintRequestedBrandId =
    productBlueprint?.brandId ?? "";

  /**
   * ミント申請の送信中、またはBackend上で
   * 非同期Mint処理中の状態を「ミント中」として扱う。
   */
  const isMinting =
    isSubmittingMintRequest ||
    isMintProcessing;

  /**
   * Mint状態だけを3秒ごとに再取得する。
   * GET /mint/requestsのみを呼び、inspection情報は周期的に再取得しない。
   */
  React.useEffect(() => {
    if (!productionId || !isMintProcessing) return;

    let cancelled = false;

    const timer = window.setInterval(() => {
      if (cancelled) return;

      void reloadMintStatus().catch(() => {
        // Mint状態の再取得失敗では画面全体をエラーにしない。
      });
    }, 3000);

    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [
    productionId,
    isMintProcessing,
    reloadMintStatus,
  ]);

  const inspectionStatus =
    inspectionBatch?.status ?? "";

  const isInspectionCompleted =
    inspectionStatus === "completed";

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

  /**
   * Mint申請に関する入力UIは、
   * 検品完了後かつMint処理開始前だけ表示する。
   */
  const showMintControls =
    isInspectionCompleted &&
    !isMinting &&
    !isMintCompleted;

  useMintAutoSelection({
    hasMint,
    mintRequestedBrandId,
    selectedBrandId,
    handleSelectBrand,
    mintRequestedTokenBlueprintId,
    selectedTokenBlueprintId,
    setSelectedTokenBlueprintId,
  });

  /**
   * TokenBlueprint選択後にBubblegum V2 Mintの
   * Reserve / Fee Payer残高とSOL費用見積を取得する。
   * metadataUriはFrontendから送信しない。
   */
  React.useEffect(() => {
    if (
      !showMintControls ||
      !productionId ||
      !selectedTokenBlueprintId
    ) {
      setMintFundingEstimate(null);
      setMintFundingEstimateError(null);
      setMintFundingEstimateLoading(false);
      return;
    }

    let cancelled = false;

    const run = async () => {
      setMintFundingEstimate(null);
      setMintFundingEstimateError(null);
      setMintFundingEstimateLoading(true);

      try {
        const estimate =
          await mintRequestRepo.fetchMintFundingEstimate(
            productionId,
            selectedTokenBlueprintId,
          );

        if (!cancelled) setMintFundingEstimate(estimate);
      } catch (error: unknown) {
        if (!cancelled) {
          setMintFundingEstimate(null);
          setMintFundingEstimateError(
            getErrorMessage(
              error,
              "SOL見積の取得に失敗しました",
            ),
          );
        }
      } finally {
        if (!cancelled) setMintFundingEstimateLoading(false);
      }
    };

    void run();

    return () => {
      cancelled = true;
    };
  }, [
    mintRequestRepo,
    productionId,
    selectedTokenBlueprintId,
    showMintControls,
  ]);

  const displayTokenBlueprintId = React.useMemo(
    () =>
      selectedTokenBlueprintId ||
      mintRequestedTokenBlueprintId,
    [
      selectedTokenBlueprintId,
      mintRequestedTokenBlueprintId,
    ],
  );

  const handleCompleteInspection = React.useCallback(
    async () => {
      if (
        isCompletingInspection ||
        isMinting ||
        isMintCompleted
      ) {
        return;
      }

      const confirmed = window.confirm(
        "検品を完了します。未入力の検品結果は合格として確定されます。よろしいですか？",
      );

      if (!confirmed) return;

      setIsCompletingInspection(true);

      try {
        const result = await completeMintInspection(
          mintRequestRepo,
          {
            inspectionBatch,
            productionId,
          },
        );

        if (!result.ok) {
          alert(result.message);
          return;
        }

        if (result.inspectionBatch) {
          setInspectionBatch(result.inspectionBatch);
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
    },
    [
      inspectionBatch,
      isCompletingInspection,
      isMinting,
      isMintCompleted,
      mintRequestRepo,
      productionId,
      reloadDetail,
    ],
  );

  const handleMint = React.useCallback(
    async () => {
      if (
        !isInspectionCompleted ||
        isMinting ||
        isMintCompleted
      ) {
        return;
      }

      if (mintFundingEstimateLoading) {
        alert(
          "SOL見積を取得中です。完了後にミント申請を実行してください。",
        );
        return;
      }

      if (!mintFundingEstimate) {
        alert(
          mintFundingEstimateError ||
            "SOL見積を取得できていません。",
        );
        return;
      }

      if (!mintFundingEstimate.estimate.sufficient) {
        alert(
          "Reserve WalletのSOL残高が不足しているため、ミント申請を実行できません。",
        );
        return;
      }

      setIsSubmittingMintRequest(true);
      setError(null);

      try {
        const result = await submitMintRequest(
          mintRequestRepo,
          {
            inspectionBatch,
            selectedTokenBlueprintId,
            productionId,
          },
        );

        if (!result.ok) {
          if (result.reason === "validation") {
            alert(result.message);
            return;
          }

          setError(result.message);

          alert(
            `ミント申請に失敗しました: ${result.message}`,
          );

          try {
            await reloadDetail();
          } catch {
            // エラー表示を優先するため、再取得失敗は握りつぶす。
          }

          return;
        }

        const { queuedResponse } = result;

        /**
         * Mint申請直後だけは画面全体を一度再取得する。
         * 以降の周期処理はreloadMintStatusのみを使用する。
         */
        await reloadDetail();

        alert(
          `ミント申請を受け付けました（生産ID: ${queuedResponse.productionId} / ミント数: ${totalMintQuantity}）。順次ミント処理を実行します。`,
        );
      } catch (error: unknown) {
        const message = getErrorMessage(
          error,
          "不明なエラーが発生しました",
        );

        setError(message);

        alert(
          `ミント申請に失敗しました: ${message}`,
        );

        try {
          await reloadDetail();
        } catch {
          // エラー表示を優先するため、再取得失敗は握りつぶす。
        }
      } finally {
        setIsSubmittingMintRequest(false);
      }
    },
    [
      inspectionBatch,
      isInspectionCompleted,
      isMinting,
      isMintCompleted,
      mintFundingEstimate,
      mintFundingEstimateError,
      mintFundingEstimateLoading,
      mintRequestRepo,
      productionId,
      reloadDetail,
      selectedTokenBlueprintId,
      totalMintQuantity,
    ],
  );

  const handleSelectTokenBlueprint = React.useCallback(
    (tokenBlueprintId: string) => {
      setSelectedTokenBlueprintId(tokenBlueprintId);
    },
    [],
  );

  const selectedTokenBlueprint = React.useMemo(
    () =>
      tokenBlueprintOptions.find(
        (tokenBlueprint) =>
          tokenBlueprint.id ===
          selectedTokenBlueprintId,
      ) ?? null,
    [
      tokenBlueprintOptions,
      selectedTokenBlueprintId,
    ],
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

  /**
   * 現行GET /mint/requests responseにcreatedAtは含まれていない。
   */
  const mintCreatedAtLabel = "（未登録）";

  const mintCreatedByLabel =
    createdByName ?? "（不明）";

  const mintMintedAtLabel =
    safeDateTimeLabelJa(
      mintRequestRow?.mintedAt ?? null,
      "（未完了）",
    );

  /**
   * 現行GET /mint/requests responseにonChainTxSignatureは含まれていない。
   */
  const onChainTxSignature = "";

  return {
    title,
    loading,
    error,
    inspectionCardData,

    /**
     * 右カラムでGET /mint/requestsのrowを直接参照できるよう返す。
     */
    mintRequestRow,
    mintStatus,
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