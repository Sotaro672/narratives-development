// frontend/console/shell/src/features/mintRequest/presentation/hook/useMintRequestDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import type {
  BrandSummary,
  MintTaskProgress,
  TokenBlueprintSummary,
} from "../../application/port/MintRequestRepository";

import { completeMintInspection } from "../../application/usecase/completeMintInspection";
import { getMintRequestDetail } from "../../application/usecase/getMintRequestDetail";
import { getMintRequestProductBlueprintPatch } from "../../application/usecase/getMintRequestProductBlueprintPatch";
import { submitMintRequest } from "../../application/usecase/submitMintRequest";

import type { InspectionBatchDTO } from "../../../../shared/types/inspections";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";

import type { MintRequestManagementRowDTO } from "../../infrastructure/dto/mintRequestManagementRow";
import type {
  MintModelMetaEntryDTO,
  ProductBlueprintPatchDTO,
} from "../../infrastructure/dto/mintRequestLocal.dto";

import { HttpMintRequestRepository } from "../../infrastructure/repository/HttpMintRequestRepository";

import {
  buildProductBlueprintCardView,
  buildTokenBlueprintCardVm,
} from "../viewModel/mintRequestDetailViewModel";

import { useInspectionResultCard } from "./useInspectionResultCard";
import { useMintAutoSelection } from "./useMintRequestDetail.useMintAutoSelection";

function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  return fallback;
}

const MINT_REQUEST_MANAGEMENT_PATH = "/mint";

export function useMintRequestDetail() {
  const navigate = useNavigate();

  /**
   * route名はrequestIdのままでも、
   * 実体はproductionIdとして扱う。
   */
  const { requestId } = useParams<{
    requestId: string;
  }>();

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

  /**
   * 現行GET /mint/requests responseには
   * mintProgressが存在しない。
   *
   * mintDetail.tsxとの互換性維持のため、
   * 現時点では常にnullとして返す。
   */
  const [mintProgress] =
    React.useState<MintTaskProgress | null>(null);

  const [productBlueprintId, setProductBlueprintId] =
    React.useState("");

  const [loading, setLoading] =
    React.useState(false);

  const [error, setError] =
    React.useState<string | null>(null);

  const [pbPatch, setPbPatch] =
    React.useState<ProductBlueprintPatchDTO | null>(null);

  const [pbPatchLoading, setPbPatchLoading] =
    React.useState(false);

  const [pbPatchError, setPbPatchError] =
    React.useState<string | null>(null);

  const [brandOptions, setBrandOptions] =
    React.useState<BrandSummary[]>([]);

  const [selectedBrandId, setSelectedBrandId] =
    React.useState("");

  const [tokenBlueprintOptions, setTokenBlueprintOptions] =
    React.useState<TokenBlueprintSummary[]>([]);

  const [selectedTokenBlueprintId, setSelectedTokenBlueprintId] =
    React.useState("");

  const [scheduledBurnDate, setScheduledBurnDate] =
    React.useState("");

  const [isSubmittingMintRequest, setIsSubmittingMintRequest] =
    React.useState(false);

  const [isCompletingInspection, setIsCompletingInspection] =
    React.useState(false);

  const title = "ミント申請詳細";

  const selectedBrandName = React.useMemo(() => {
    if (!selectedBrandId) {
      return "";
    }

    return (
      brandOptions.find(
        (brand) => brand.id === selectedBrandId,
      )?.name ?? ""
    );
  }, [brandOptions, selectedBrandId]);

  const reloadDetail = React.useCallback(async () => {
    if (!productionId) {
      return;
    }

    const detail = await getMintRequestDetail(
      mintRequestRepo,
      productionId,
    );

    setInspectionBatch(detail.inspectionBatch);
    setMintRequestRow(detail.mintRequestRow);
    setProductBlueprintId(detail.productBlueprintId || "");
  }, [mintRequestRepo, productionId]);

  React.useEffect(() => {
    if (!productionId) {
      return;
    }

    let cancelled = false;

    const run = async () => {
      setLoading(true);
      setError(null);

      try {
        const detail = await getMintRequestDetail(
          mintRequestRepo,
          productionId,
        );

        if (cancelled) {
          return;
        }

        setInspectionBatch(detail.inspectionBatch);
        setMintRequestRow(detail.mintRequestRow);
        setProductBlueprintId(detail.productBlueprintId || "");
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
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    void run();

    return () => {
      cancelled = true;
    };
  }, [mintRequestRepo, productionId]);

  React.useEffect(() => {
    if (!productBlueprintId) {
      setPbPatch(null);
      return;
    }

    let cancelled = false;

    const run = async () => {
      setPbPatchLoading(true);
      setPbPatchError(null);

      try {
        const patch =
          await getMintRequestProductBlueprintPatch(
            mintRequestRepo,
            productBlueprintId,
          );

        if (!cancelled) {
          setPbPatch(patch);
        }
      } catch (error: unknown) {
        if (!cancelled) {
          setPbPatchError(
            getErrorMessage(
              error,
              "プロダクト基本情報の取得に失敗しました",
            ),
          );
        }
      } finally {
        if (!cancelled) {
          setPbPatchLoading(false);
        }
      }
    };

    void run();

    return () => {
      cancelled = true;
    };
  }, [mintRequestRepo, productBlueprintId]);

  const batchForInspectionCard = React.useMemo(() => {
    if (!inspectionBatch) {
      return undefined;
    }

    /**
     * InspectionBatchDTO.modelMetaは、
     * modelIdをRecordのキーとして保持している。
     *
     * Backendが返したmodelNumber / size / colorName / rgb /
     * volume / volumeUnitをそのまま保持し、
     * 値側にもmodelIdを含める。
     */
    const modelMeta = Object.entries(
      inspectionBatch.modelMeta ?? {},
    ).reduce<Record<string, MintModelMetaEntryDTO>>(
      (result, [modelId, meta]) => {
        result[modelId] = {
          ...meta,
          modelId,
        };

        return result;
      },
      {},
    );

    return {
      ...inspectionBatch,
      modelMeta,
      productBlueprintPatch: pbPatch ?? null,
    };
  }, [inspectionBatch, pbPatch]);

  /**
   * Model情報はBackend responseのmodelMetaを正とする。
   * frontendからGET /models/{modelId}による個別補完は行わない。
   */
  const inspectionCardData = useInspectionResultCard({
    batch: batchForInspectionCard,
  });

  const totalMintQuantity =
    mintRequestRow?.mintQuantity ??
    inspectionCardData.totalPassed;

  const productBlueprintCardView = React.useMemo(
    () => buildProductBlueprintCardView(pbPatch),
    [pbPatch],
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

        if (cancelled) {
          return;
        }

        setBrandOptions(brands);
      } catch {
        if (!cancelled) {
          setBrandOptions([]);
        }
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
   * GET /mint/requests responseのmintStatusを
   * Mint状態の正とする。
   */
  const mintStatus = React.useMemo(
    () =>
      String(mintRequestRow?.mintStatus ?? "")
        .trim()
        .toUpperCase(),
    [mintRequestRow?.mintStatus],
  );

  /**
   * BackendはMintが存在しないProductionについても
   * management rowを返し得るため、
   * rowの存在そのものではhasMintを判定しない。
   */
  const hasMint = React.useMemo(() => {
    return Boolean(
      mintStatus ||
        mintRequestRow?.tokenBlueprintId ||
        mintRequestRow?.requestedBy ||
        mintRequestRow?.mintedAt,
    );
  }, [
    mintStatus,
    mintRequestRow?.tokenBlueprintId,
    mintRequestRow?.requestedBy,
    mintRequestRow?.mintedAt,
  ]);

  /**
   * 非同期Mint処理中としてポーリングする状態。
   */
  const isMintProcessing =
    mintStatus === "QUEUED" ||
    mintStatus === "MINTING" ||
    mintStatus === "PARTIALLY_MINTED";

  const isMintCompleted =
    mintStatus === "MINTED";

  const createdByName =
    mintRequestRow?.createdByName ||
    mintRequestRow?.createdBy ||
    null;

  const requestedByName =
    mintRequestRow?.requestedByName ||
    mintRequestRow?.requestedBy ||
    null;

  const mintRequestedTokenBlueprintId =
    mintRequestRow?.tokenBlueprintId ?? "";

  /**
   * 現行GET /mint/requests responseにはbrandIdがないため、
   * ProductBlueprintのbrandIdを使用する。
   */
  const mintRequestedBrandId =
    pbPatch?.brandId ?? "";

  /**
   * ミント申請の送信中、またはBackend上で
   * 非同期Mint処理中の状態を「ミント中」として扱う。
   */
  const isMinting =
    isSubmittingMintRequest ||
    isMintProcessing;

  /**
   * 現行GET /mint/requests responseには
   * mintProgressが含まれないため非表示とする。
   */
  const showMintProgress = false;

  /**
   * Mint状態のみを3秒ごとに再取得する。
   *
   * inspectionBatchが更新されても
   * Model Variationの個別取得は行わない。
   */
  React.useEffect(() => {
    if (!productionId || !isMintProcessing) {
      return;
    }

    let cancelled = false;

    const timer = window.setInterval(() => {
      if (cancelled) {
        return;
      }

      void reloadDetail().catch(() => {
        // 状態再取得失敗では画面全体をエラーにしない。
      });
    }, 3000);

    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [
    productionId,
    isMintProcessing,
    reloadDetail,
  ]);

  const inspectionStatus =
    inspectionBatch?.status ?? "";

  const isInspectionCompleted =
    inspectionStatus === "completed";

  const showCompleteInspectionButton =
    React.useMemo(() => {
      return Boolean(
        inspectionBatch &&
          !loading &&
          !error &&
          !isMinting &&
          !isMintCompleted &&
          !isInspectionCompleted,
      );
    }, [
      inspectionBatch,
      loading,
      error,
      isMinting,
      isMintCompleted,
      isInspectionCompleted,
    ]);

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

    /**
     * 現行GET /mint/requests responseには
     * scheduledBurnDateが存在しない。
     */
    mintScheduledBurnDate: undefined,
    scheduledBurnDate,
    setScheduledBurnDate,
  });

  const tokenBlueprintIdForPatch =
    React.useMemo(() => {
      return (
        selectedTokenBlueprintId ||
        mintRequestedTokenBlueprintId
      );
    }, [
      selectedTokenBlueprintId,
      mintRequestedTokenBlueprintId,
    ]);

  const handleCompleteInspection =
    React.useCallback(
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

        if (!confirmed) {
          return;
        }

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
            setInspectionBatch(
              result.inspectionBatch,
            );
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

  const handleMint =
    React.useCallback(
      async () => {
        if (
          !isInspectionCompleted ||
          isMinting ||
          isMintCompleted
        ) {
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
              scheduledBurnDate,
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
        mintRequestRepo,
        productionId,
        reloadDetail,
        scheduledBurnDate,
        selectedTokenBlueprintId,
        totalMintQuantity,
      ],
    );

  const handleSelectTokenBlueprint =
    React.useCallback(
      (tokenBlueprintId: string) => {
        setSelectedTokenBlueprintId(
          tokenBlueprintId,
        );
      },
      [],
    );

  const selectedTokenBlueprint =
    React.useMemo(
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

  const tokenBlueprintCardVm =
    React.useMemo(
      () =>
        buildTokenBlueprintCardVm({
          selectedTokenBlueprint,
          tokenBlueprintIdForPatch,
          selectedBrandName,
          pbPatch,
          brandOptions,
        }),
      [
        selectedTokenBlueprint,
        tokenBlueprintIdForPatch,
        selectedBrandName,
        pbPatch,
        brandOptions,
      ],
    );

  /**
   * 現行GET /mint/requests responseに
   * createdAtは含まれていない。
   */
  const mintCreatedAtLabel = "（未登録）";

  const mintCreatedByLabel =
    createdByName ||
    "（不明）";

  /**
   * 現行GET /mint/requests responseに
   * scheduledBurnDateは含まれていない。
   */
  const mintScheduledBurnDateLabel = "（未設定）";

  const mintMintedAtLabel =
    safeDateTimeLabelJa(
      mintRequestRow?.mintedAt ?? null,
      "（未完了）",
    );

  /**
   * 現行GET /mint/requests responseに
   * onChainTxSignatureは含まれていない。
   */
  const onChainTxSignature = "";

  return {
    title,
    loading,
    error,
    inspectionCardData,

    /**
     * 右カラムでGET /mint/requestsの
     * management rowを直接参照できるよう返す。
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
    mintProgress,
    showMintProgress,
    showCompleteInspectionButton,
    isCompletingInspection,
    handleCompleteInspection,
    requestedByName,
    productBlueprintCardView,
    pbPatchLoading,
    pbPatchError,
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
    mintScheduledBurnDateLabel,
    mintMintedAtLabel,
    onChainTxSignature,
    scheduledBurnDate,
    setScheduledBurnDate,
  };
}