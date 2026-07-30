// frontend/console/shell/src/features/mintRequest/presentation/hook/useMintRequestDetail.tsx

import * as React from "react";
import {
  useNavigate,
  useParams,
} from "react-router-dom";

import type {
  MintInfo,
} from "../../application/mapper/mintInfoMapper";

import type {
  BrandSummary,
  MintTaskProgress,
  TokenBlueprintSummary,
} from "../../application/port/MintRequestRepository";

import {
  selectMintRequestDetailState,
} from "../../application/selector/selectMintRequestDetailState";

import {
  completeMintInspection,
} from "../../application/usecase/completeMintInspection";

import {
  getMintRequestDetail,
} from "../../application/usecase/getMintRequestDetail";

import {
  getMintRequestProductBlueprintPatch,
} from "../../application/usecase/getMintRequestProductBlueprintPatch";

import {
  submitMintRequest,
} from "../../application/usecase/submitMintRequest";

import type {
  InspectionBatchDTO,
} from "../../domain/inspections";

import type {
  MintModelMetaEntryDTO,
  ProductBlueprintPatchDTO,
} from "../../infrastructure/dto/mintRequestLocal.dto";

import {
  HttpMintRequestRepository,
} from "../../infrastructure/repository/HttpMintRequestRepository";

import {
  buildMintLabels,
  buildProductBlueprintCardView,
  buildTokenBlueprintCardVm,
} from "../viewModel/mintRequestDetailViewModel";

import {
  useInspectionResultCard,
} from "./useInspectionResultCard";

import {
  useMintAutoSelection,
} from "./useMintRequestDetail.useMintAutoSelection";

function getErrorMessage(
  error: unknown,
  fallback: string,
): string {
  if (
    error instanceof Error &&
    error.message
  ) {
    return error.message;
  }

  return fallback;
}

const MINT_REQUEST_MANAGEMENT_PATH =
  "/mintRequest";

export function useMintRequestDetail() {
  const navigate =
    useNavigate();

  /**
   * route名はrequestIdのままでも、
   * 実体はproductionIdとして扱う。
   */
  const {
    requestId,
  } = useParams<{
    requestId: string;
  }>();

  const productionId =
    React.useMemo(
      () =>
        String(
          requestId ?? "",
        ).trim(),
      [requestId],
    );

  const mintRequestRepo =
    React.useMemo(
      () =>
        new HttpMintRequestRepository(),
      [],
    );

  const [
    inspectionBatch,
    setInspectionBatch,
  ] =
    React.useState<
      InspectionBatchDTO | null
    >(null);

  const [
    mintInfo,
    setMintInfo,
  ] =
    React.useState<
      MintInfo | null
    >(null);

  const [
    mintProgress,
    setMintProgress,
  ] =
    React.useState<
      MintTaskProgress | null
    >(null);

  const [
    productBlueprintId,
    setProductBlueprintId,
  ] =
    React.useState("");

  const [
    loading,
    setLoading,
  ] =
    React.useState(false);

  const [
    error,
    setError,
  ] =
    React.useState<
      string | null
    >(null);

  const [
    pbPatch,
    setPbPatch,
  ] =
    React.useState<
      ProductBlueprintPatchDTO | null
    >(null);

  const [
    pbPatchLoading,
    setPbPatchLoading,
  ] =
    React.useState(false);

  const [
    pbPatchError,
    setPbPatchError,
  ] =
    React.useState<
      string | null
    >(null);

  const [
    brandOptions,
    setBrandOptions,
  ] =
    React.useState<
      BrandSummary[]
    >([]);

  const [
    selectedBrandId,
    setSelectedBrandId,
  ] =
    React.useState("");

  const [
    tokenBlueprintOptions,
    setTokenBlueprintOptions,
  ] =
    React.useState<
      TokenBlueprintSummary[]
    >([]);

  const [
    selectedTokenBlueprintId,
    setSelectedTokenBlueprintId,
  ] =
    React.useState("");

  const [
    scheduledBurnDate,
    setScheduledBurnDate,
  ] =
    React.useState("");

  const [
    isSubmittingMintRequest,
    setIsSubmittingMintRequest,
  ] =
    React.useState(false);

  const [
    isCompletingInspection,
    setIsCompletingInspection,
  ] =
    React.useState(false);

  const title =
    "ミント申請詳細";

  const selectedBrandName =
    React.useMemo(
      () => {
        if (!selectedBrandId) {
          return "";
        }

        return (
          brandOptions.find(
            (brand) =>
              brand.id ===
              selectedBrandId,
          )?.name ?? ""
        );
      },
      [
        brandOptions,
        selectedBrandId,
      ],
    );

  const reloadDetail =
    React.useCallback(
      async () => {
        if (!productionId) {
          return;
        }

        const detail =
          await getMintRequestDetail(
            mintRequestRepo,
            productionId,
          );

        setInspectionBatch(
          detail.inspectionBatch,
        );

        setMintInfo(
          detail.mint,
        );

        setMintProgress(
          detail.mintProgress,
        );

        setProductBlueprintId(
          detail.productBlueprintId ||
            "",
        );
      },
      [
        mintRequestRepo,
        productionId,
      ],
    );

  React.useEffect(
    () => {
      if (!productionId) {
        return;
      }

      let cancelled =
        false;

      const run =
        async () => {
          setLoading(true);
          setError(null);

          try {
            const detail =
              await getMintRequestDetail(
                mintRequestRepo,
                productionId,
              );

            if (cancelled) {
              return;
            }

            setInspectionBatch(
              detail.inspectionBatch,
            );

            setMintInfo(
              detail.mint,
            );

            setMintProgress(
              detail.mintProgress,
            );

            setProductBlueprintId(
              detail.productBlueprintId ||
                "",
            );
          } catch (
            error: unknown
          ) {
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
        cancelled =
          true;
      };
    },
    [
      mintRequestRepo,
      productionId,
    ],
  );

  React.useEffect(
    () => {
      if (
        !productBlueprintId
      ) {
        setPbPatch(null);
        return;
      }

      let cancelled =
        false;

      const run =
        async () => {
          setPbPatchLoading(
            true,
          );

          setPbPatchError(
            null,
          );

          try {
            const patch =
              await getMintRequestProductBlueprintPatch(
                mintRequestRepo,
                productBlueprintId,
              );

            if (!cancelled) {
              setPbPatch(
                patch,
              );
            }
          } catch (
            error: unknown
          ) {
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
              setPbPatchLoading(
                false,
              );
            }
          }
        };

      void run();

      return () => {
        cancelled =
          true;
      };
    },
    [
      mintRequestRepo,
      productBlueprintId,
    ],
  );

  const batchForInspectionCard =
    React.useMemo(
      () => {
        if (
          !inspectionBatch
        ) {
          return undefined;
        }

        /**
         * InspectionBatchDTO.modelMetaは、
         * modelIdをRecordのキーとして保持している。
         *
         * InspectionResultCard側ではBackendの
         * MintModelMetaEntryDTOに合わせて、
         * 値側にもmodelIdを含める。
         */
        const modelMeta =
          Object.entries(
            inspectionBatch
              .modelMeta ?? {},
          ).reduce<
            Record<
              string,
              MintModelMetaEntryDTO
            >
          >(
            (
              result,
              [
                modelId,
                meta,
              ],
            ) => {
              result[
                modelId
              ] = {
                modelId,

                size:
                  meta.size,

                colorName:
                  meta.colorName,

                rgb:
                  meta.rgb,
              };

              return result;
            },
            {},
          );

        return {
          ...inspectionBatch,

          modelMeta,

          productBlueprintPatch:
            pbPatch ?? null,
        };
      },
      [
        inspectionBatch,
        pbPatch,
      ],
    );

  const inspectionCardData =
    useInspectionResultCard({
      batch:
        batchForInspectionCard,

      modelMetaRepository:
        mintRequestRepo,
    });

  const totalMintQuantity =
    inspectionCardData
      .totalPassed;

  const productBlueprintCardView =
    React.useMemo(
      () =>
        buildProductBlueprintCardView(
          pbPatch,
        ),
      [pbPatch],
    );

  const onBack =
    React.useCallback(
      () => {
        navigate(
          MINT_REQUEST_MANAGEMENT_PATH,
        );
      },
      [navigate],
    );

  React.useEffect(
    () => {
      let cancelled =
        false;

      const run =
        async () => {
          try {
            const brands =
              await mintRequestRepo
                .fetchBrandsForMint();

            if (cancelled) {
              return;
            }

            setBrandOptions(
              brands,
            );
          } catch {
            if (!cancelled) {
              setBrandOptions(
                [],
              );
            }
          }
        };

      void run();

      return () => {
        cancelled =
          true;
      };
    },
    [mintRequestRepo],
  );

  const handleSelectBrand =
    React.useCallback(
      async (
        brandId: string,
      ) => {
        setSelectedBrandId(
          brandId,
        );

        if (!brandId) {
          setTokenBlueprintOptions(
            [],
          );

          setSelectedTokenBlueprintId(
            "",
          );

          return;
        }

        try {
          const options =
            await mintRequestRepo
              .fetchTokenBlueprintsByBrand(
                brandId,
              );

          setTokenBlueprintOptions(
            options,
          );

          setSelectedTokenBlueprintId(
            "",
          );
        } catch {
          setTokenBlueprintOptions(
            [],
          );

          setSelectedTokenBlueprintId(
            "",
          );
        }
      },
      [mintRequestRepo],
    );

  const {
    mint,
    hasMint,
    isMinting:
      isMintProcessing,
    isMintCompleted,
    createdByName,
    requestedByName,
    mintRequestedTokenBlueprintId,
    mintRequestedBrandId,
  } =
    React.useMemo(
      () =>
        selectMintRequestDetailState({
          mint:
            mintInfo,

          productBlueprintBrandId:
            pbPatch?.brandId ??
            null,
        }),
      [
        mintInfo,
        pbPatch?.brandId,
      ],
    );

  /**
   * ミント申請の送信中、またはBackend上で
   * MintがMINTEDになる前の状態を「ミント中」として扱う。
   */
  const isMinting =
    isSubmittingMintRequest ||
    isMintProcessing;

  const showMintProgress =
    React.useMemo(
      () => {
        return Boolean(
          isMintProcessing &&
            mintProgress &&
            mintProgress.total >
              0,
        );
      },
      [
        isMintProcessing,
        mintProgress,
      ],
    );

  React.useEffect(
    () => {
      if (!productionId) {
        return;
      }

      if (
        !isMintProcessing
      ) {
        return;
      }

      let cancelled =
        false;

      const timer =
        window.setInterval(
          () => {
            if (cancelled) {
              return;
            }

            void reloadDetail().catch(
              () => {
                // 進捗取得失敗では画面全体をエラーにしない。
              },
            );
          },
          3000,
        );

      return () => {
        cancelled =
          true;

        window.clearInterval(
          timer,
        );
      };
    },
    [
      productionId,
      isMintProcessing,
      reloadDetail,
    ],
  );

  const inspectionStatus =
    inspectionBatch?.status ??
    "";

  const isInspectionCompleted =
    inspectionStatus ===
    "completed";

  const showCompleteInspectionButton =
    React.useMemo(
      () => {
        return Boolean(
          inspectionBatch &&
            !loading &&
            !error &&
            !isMinting &&
            !isMintCompleted &&
            !isInspectionCompleted,
        );
      },
      [
        inspectionBatch,
        loading,
        error,
        isMinting,
        isMintCompleted,
        isInspectionCompleted,
      ],
    );

  const showMintControls =
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

    mintScheduledBurnDate:
      mint?.scheduledBurnDate,

    scheduledBurnDate,
    setScheduledBurnDate,
  });

  const tokenBlueprintIdForPatch =
    React.useMemo(
      () => {
        return (
          selectedTokenBlueprintId ||
          mintRequestedTokenBlueprintId
        );
      },
      [
        selectedTokenBlueprintId,
        mintRequestedTokenBlueprintId,
      ],
    );

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

        const confirmed =
          window.confirm(
            "検品を完了します。未入力の検品結果は合格として確定されます。よろしいですか？",
          );

        if (!confirmed) {
          return;
        }

        setIsCompletingInspection(
          true,
        );

        try {
          const result =
            await completeMintInspection(
              mintRequestRepo,
              {
                inspectionBatch,
                productionId,
              },
            );

          if (!result.ok) {
            alert(
              result.message,
            );

            return;
          }

          if (
            result.inspectionBatch
          ) {
            setInspectionBatch(
              result.inspectionBatch,
            );
          }

          await reloadDetail();

          alert(
            "検品を完了しました。",
          );
        } catch (
          error: unknown
        ) {
          alert(
            `検品完了に失敗しました: ${getErrorMessage(
              error,
              "不明なエラーが発生しました",
            )}`,
          );
        } finally {
          setIsCompletingInspection(
            false,
          );
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
          isMinting ||
          isMintCompleted
        ) {
          return;
        }

        setIsSubmittingMintRequest(
          true,
        );

        setError(null);

        try {
          const result =
            await submitMintRequest(
              mintRequestRepo,
              {
                inspectionBatch,

                selectedTokenBlueprintId,

                productionId,

                scheduledBurnDate,
              },
            );

          if (!result.ok) {
            if (
              result.reason ===
              "validation"
            ) {
              alert(
                result.message,
              );

              return;
            }

            setError(
              result.message,
            );

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

          const {
            queuedResponse,
          } =
            result;

          await reloadDetail();

          alert(
            `ミント申請を受け付けました（生産ID: ${queuedResponse.productionId} / ミント数: ${totalMintQuantity}）。順次ミント処理を実行します。`,
          );
        } catch (
          error: unknown
        ) {
          const message =
            getErrorMessage(
              error,
              "不明なエラーが発生しました",
            );

          setError(
            message,
          );

          alert(
            `ミント申請に失敗しました: ${message}`,
          );

          try {
            await reloadDetail();
          } catch {
            // エラー表示を優先するため、再取得失敗は握りつぶす。
          }
        } finally {
          setIsSubmittingMintRequest(
            false,
          );
        }
      },
      [
        inspectionBatch,
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
      (
        tokenBlueprintId: string,
      ) => {
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
          (
            tokenBlueprint,
          ) =>
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

  const {
    mintCreatedAtLabel,
    mintCreatedByLabel,
    mintScheduledBurnDateLabel,
    mintMintedAtLabel,
    onChainTxSignature,
  } =
    React.useMemo(
      () =>
        buildMintLabels({
          mint,
          createdByName,
        }),
      [
        mint,
        createdByName,
      ],
    );

  return {
    title,
    loading,
    error,

    inspectionCardData,

    totalMintQuantity,

    onBack,
    handleMint,
    isMinting,

    hasMint,

    isMintCompleted,
    isInspectionCompleted,

    showMintButton:
      showMintControls,

    showBrandSelectorCard:
      showMintControls,

    showTokenSelectorCard:
      showMintControls,

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