// frontend/console/shell/src/features/mintRequest/presentation/hook/useMintRequestDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import { asNonEmptyString } from "../../application/util/primitive";

import {
  toBrandOptionVMs,
  toTokenBlueprintOptionVMs,
} from "../../application/mapper/mintRequestOptionsMapper";

import { getMintRequestDetail } from "../../application/usecase/getMintRequestDetail";
import { getProductBlueprintPatch } from "../../application/usecase/getProductBlueprintPatch";
import { listBrandsForMint } from "../../application/usecase/listBrandsForMint";
import { listTokenBlueprintsByBrand } from "../../application/usecase/listTokenBlueprintsByBrand";
import { submitMintRequestAndRefresh } from "../../application/usecase/submitMintRequestAndRefresh";

import { validateCompleteInspection } from "../../application/validator/validateCompleteInspection";
import { validateMintRequestSubmit } from "../../application/validator/validateMintRequestSubmit";

import type { InspectionBatchDTO } from "../../domain/inspections";

import type { MintDTO } from "../../infrastructure/dto/mint.dto";
import type { ProductBlueprintPatchDTO } from "../../infrastructure/dto/mintRequestLocal.dto";

import { completeInspectionHTTP } from "../../infrastructure/repository";
import { HttpMintRequestRepository } from "../../infrastructure/repository/HttpMintRequestRepository";

import type {
  BrandOptionVM as BrandOption,
  ProductBlueprintCardVM as ProductBlueprintCardViewModel,
  TokenBlueprintCardHandlersVM as TokenBlueprintCardHandlers,
  TokenBlueprintCardVM as TokenBlueprintCardViewModel,
  TokenBlueprintOptionVM as TokenBlueprintOption,
} from "../viewModel/mintRequestDetail.vm";

import { useInspectionResultCard } from "./useInspectionResultCard";
import { useMintInfo } from "./useMintRequestDetail.mintSelectors";
import { useMintAutoSelection } from "./useMintRequestDetail.useMintAutoSelection";

import {
  buildMintLabels,
  buildProductBlueprintCardView,
  buildTokenBlueprintCardHandlers,
  buildTokenBlueprintCardVm,
} from "./useMintRequestDetail.viewModels";

type MintTaskProgressVM = {
  total: number;
  pending: number;
  minting: number;
  minted: number;
  failedRetryable: number;
  failedFatal: number;
  percentage: number;
};

function normalizeProgressNumber(
  value: unknown,
): number {
  const numberValue = Number(value);

  if (!Number.isFinite(numberValue)) {
    return 0;
  }

  if (numberValue <= 0) {
    return 0;
  }

  return Math.trunc(numberValue);
}

function clampProgressPercentage(
  value: unknown,
): number {
  const numberValue = Number(value);

  if (!Number.isFinite(numberValue)) {
    return 0;
  }

  if (numberValue <= 0) {
    return 0;
  }

  if (numberValue >= 100) {
    return 100;
  }

  return Math.trunc(numberValue);
}

function normalizeMintTaskProgress(
  raw: unknown,
): MintTaskProgressVM | null {
  if (
    !raw ||
    typeof raw !== "object" ||
    Array.isArray(raw)
  ) {
    return null;
  }

  const progress = raw as Record<string, unknown>;

  const total = normalizeProgressNumber(
    progress.total,
  );

  const minted = normalizeProgressNumber(
    progress.minted,
  );

  const calculatedPercentage =
    total > 0
      ? Math.trunc(
          (Math.min(minted, total) / total) * 100,
        )
      : 0;

  return {
    total,
    pending: normalizeProgressNumber(
      progress.pending,
    ),
    minting: normalizeProgressNumber(
      progress.minting,
    ),
    minted,
    failedRetryable: normalizeProgressNumber(
      progress.failedRetryable,
    ),
    failedFatal: normalizeProgressNumber(
      progress.failedFatal,
    ),
    percentage:
      progress.percentage === undefined
        ? clampProgressPercentage(
            calculatedPercentage,
          )
        : clampProgressPercentage(
            progress.percentage,
          ),
  };
}

function getErrorMessage(
  error: unknown,
  fallback: string,
): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  return fallback;
}

const MINT_REQUEST_MANAGEMENT_PATH =
  "/mintRequest";

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
    React.useState<InspectionBatchDTO | null>(
      null,
    );

  const [mintDTO, setMintDTO] =
    React.useState<MintDTO | null>(null);

  const [
    productBlueprintId,
    setProductBlueprintId,
  ] = React.useState("");

  const [loading, setLoading] =
    React.useState(false);

  const [error, setError] =
    React.useState<string | null>(null);

  const [pbPatch, setPbPatch] =
    React.useState<ProductBlueprintPatchDTO | null>(
      null,
    );

  const [
    pbPatchLoading,
    setPbPatchLoading,
  ] = React.useState(false);

  const [
    pbPatchError,
    setPbPatchError,
  ] = React.useState<string | null>(null);

  const [brandOptions, setBrandOptions] =
    React.useState<BrandOption[]>([]);

  const [
    selectedBrandId,
    setSelectedBrandId,
  ] = React.useState("");

  const [
    tokenBlueprintOptions,
    setTokenBlueprintOptions,
  ] = React.useState<TokenBlueprintOption[]>(
    [],
  );

  const [
    selectedTokenBlueprintId,
    setSelectedTokenBlueprintId,
  ] = React.useState("");

  const [
    scheduledBurnDate,
    setScheduledBurnDate,
  ] = React.useState("");

  const [isMinting, setIsMinting] =
    React.useState(false);

  const [
    isCompletingInspection,
    setIsCompletingInspection,
  ] = React.useState(false);

  const title = "ミント申請詳細";

  const selectedBrandName =
    React.useMemo(() => {
      if (!selectedBrandId) {
        return "";
      }

      return (
        brandOptions.find(
          (brand) =>
            brand.id === selectedBrandId,
        )?.name ?? ""
      );
    }, [
      brandOptions,
      selectedBrandId,
    ]);

  const reloadDetail =
    React.useCallback(async () => {
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

      setMintDTO(detail.mintDTO);

      setProductBlueprintId(
        detail.productBlueprintId || "",
      );
    }, [
      mintRequestRepo,
      productionId,
    ]);

  React.useEffect(() => {
    if (!productionId) {
      return;
    }

    let cancelled = false;

    const run = async () => {
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

        setMintDTO(detail.mintDTO);

        setProductBlueprintId(
          detail.productBlueprintId || "",
        );
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
  }, [
    mintRequestRepo,
    productionId,
  ]);

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
          await getProductBlueprintPatch(
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
  }, [
    mintRequestRepo,
    productBlueprintId,
  ]);

  const batchForInspectionCard =
    React.useMemo(() => {
      if (!inspectionBatch) {
        return undefined;
      }

      return {
        ...inspectionBatch,
        productBlueprintPatch:
          pbPatch ?? null,
      };
    }, [
      inspectionBatch,
      pbPatch,
    ]);

  const inspectionCardData =
    useInspectionResultCard({
      batch: batchForInspectionCard,
    });

  const totalMintQuantity =
    inspectionCardData.totalPassed;

  const productBlueprintCardView:
    | ProductBlueprintCardViewModel
    | null = React.useMemo(
    () =>
      buildProductBlueprintCardView(
        pbPatch,
      ),
    [pbPatch],
  );

  const onBack = React.useCallback(() => {
    navigate(
      MINT_REQUEST_MANAGEMENT_PATH,
    );
  }, [navigate]);

  React.useEffect(() => {
    let cancelled = false;

    const run = async () => {
      try {
        const brands =
          await listBrandsForMint(
            mintRequestRepo,
          );

        if (cancelled) {
          return;
        }

        setBrandOptions(
          toBrandOptionVMs(brands),
        );
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

  const handleSelectBrand =
    React.useCallback(
      async (brandId: string) => {
        setSelectedBrandId(brandId);

        if (!brandId) {
          setTokenBlueprintOptions([]);
          setSelectedTokenBlueprintId("");
          return;
        }

        try {
          const options =
            await listTokenBlueprintsByBrand(
              mintRequestRepo,
              brandId,
            );

          setTokenBlueprintOptions(
            toTokenBlueprintOptionVMs(
              options,
            ),
          );

          setSelectedTokenBlueprintId(
            "",
          );
        } catch {
          setTokenBlueprintOptions([]);
          setSelectedTokenBlueprintId("");
        }
      },
      [mintRequestRepo],
    );

  const {
    mint,
    hasMint,
    isMintRequested,
    requestedByName,
    mintRequestedTokenBlueprintId,
    mintRequestedBrandId,
  } = useMintInfo({
    mintDTO,
    inspectionBatch,
    pbPatch,
  });

  /**
   * MintDTOの正規型には現時点で
   * mintProgressが含まれていないため、
   * APIレスポンス上の追加フィールドとして読み取る。
   */
  const mintProgress =
    React.useMemo(() => {
      const mintResponse =
        mintDTO as
          | (MintDTO & {
              mintProgress?: unknown;
            })
          | null;

      return normalizeMintTaskProgress(
        mintResponse?.mintProgress,
      );
    }, [mintDTO]);

  const isMintCompleted =
    React.useMemo(
      () => mint?.minted === true,
      [mint],
    );

  const showMintProgress =
    React.useMemo(() => {
      return Boolean(
        isMintRequested &&
          !isMintCompleted &&
          mintProgress &&
          mintProgress.total > 0,
      );
    }, [
      isMintRequested,
      isMintCompleted,
      mintProgress,
    ]);

  React.useEffect(() => {
    if (!productionId) {
      return;
    }

    if (!isMintRequested) {
      return;
    }

    if (isMintCompleted) {
      return;
    }

    let cancelled = false;

    const timer = window.setInterval(
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
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [
    productionId,
    isMintRequested,
    isMintCompleted,
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
          !isMintRequested &&
          !isInspectionCompleted,
      );
    }, [
      inspectionBatch,
      loading,
      error,
      isMintRequested,
      isInspectionCompleted,
    ]);

  const showMintButton =
    !isMintRequested;

  const showBrandSelectorCard =
    !isMintRequested;

  const showTokenSelectorCard =
    !isMintRequested;

  useMintAutoSelection({
    hasMint,
    mintRequestedBrandId,
    selectedBrandId,
    handleSelectBrand,
    mintRequestedTokenBlueprintId,
    selectedTokenBlueprintId,
    setSelectedTokenBlueprintId,
    mint,
    scheduledBurnDate,
    setScheduledBurnDate,
  });

  const tokenBlueprintIdForPatch =
    React.useMemo(() => {
      const selectedId =
        asNonEmptyString(
          selectedTokenBlueprintId,
        );

      if (selectedId) {
        return selectedId;
      }

      return asNonEmptyString(
        mintRequestedTokenBlueprintId,
      );
    }, [
      selectedTokenBlueprintId,
      mintRequestedTokenBlueprintId,
    ]);

  const handleCompleteInspection =
    React.useCallback(async () => {
      if (
        isCompletingInspection ||
        isMinting
      ) {
        return;
      }

      const validation =
        validateCompleteInspection({
          inspectionBatch,
          productionId,
        });

      if (!validation.ok) {
        alert(validation.message);
        return;
      }

      const confirmed =
        window.confirm(
          "検品を完了します。未入力の検品結果は合格として確定されます。よろしいですか？",
        );

      if (!confirmed) {
        return;
      }

      setIsCompletingInspection(true);

      try {
        const updatedBatch =
          await completeInspectionHTTP(
            validation.productionId,
          );

        if (updatedBatch) {
          setInspectionBatch(
            updatedBatch,
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
        setIsCompletingInspection(
          false,
        );
      }
    }, [
      inspectionBatch,
      isCompletingInspection,
      isMinting,
      productionId,
      reloadDetail,
    ]);

  const handleMint =
    React.useCallback(async () => {
      if (isMinting) {
        return;
      }

      const validation =
        validateMintRequestSubmit({
          inspectionBatch,
          isInspectionCompleted,
          selectedTokenBlueprintId,
          productionId,
        });

      if (!validation.ok) {
        alert(validation.message);
        return;
      }

      setIsMinting(true);
      setError(null);

      try {
        const queuedResponse =
          await submitMintRequestAndRefresh(
            validation.productionId,
            validation.tokenBlueprintId,
            scheduledBurnDate ||
              undefined,
          );

        if (!queuedResponse) {
          const message =
            "ミント申請の受付結果を取得できませんでした。";

          setError(message);

          alert(
            `ミント申請に失敗しました: ${message}`,
          );

          try {
            await reloadDetail();
          } catch {
            // エラー表示を優先するため、再取得失敗は握りつぶす。
          }

          return;
        }

        await reloadDetail();

        alert(
          `ミント申請を受け付けました（生産ID: ${queuedResponse.productionId} / ミント数: ${totalMintQuantity}）。順次ミント処理を実行します。`,
        );
      } catch (error: unknown) {
        const message =
          getErrorMessage(
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
        setIsMinting(false);
      }
    }, [
      inspectionBatch,
      isInspectionCompleted,
      isMinting,
      productionId,
      reloadDetail,
      scheduledBurnDate,
      selectedTokenBlueprintId,
      totalMintQuantity,
    ]);

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
          (tokenBlueprint) =>
            tokenBlueprint.id ===
            selectedTokenBlueprintId,
        ) ?? null,
      [
        tokenBlueprintOptions,
        selectedTokenBlueprintId,
      ],
    );

  const tokenBlueprintCardVm:
    | TokenBlueprintCardViewModel
    | null = React.useMemo(
    () =>
      buildTokenBlueprintCardVm({
        selectedTokenBlueprint,
        tokenBlueprintIdForPatch,
        selectedBrandName,
        tokenBlueprintPatch: null,
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

  const tokenBlueprintCardHandlers:
    TokenBlueprintCardHandlers =
      React.useMemo(
        () =>
          buildTokenBlueprintCardHandlers(
            tokenBlueprintCardVm?.iconUrl,
          ),
        [tokenBlueprintCardVm?.iconUrl],
      );

  const {
    mintCreatedAtLabel,
    mintCreatedByLabel,
    mintScheduledBurnDateLabel,
    mintMintedAtLabel,
    onChainTxSignature,
  } = React.useMemo(
    () =>
      buildMintLabels({
        mint,
        requestedByName,
      }),
    [
      mint,
      requestedByName,
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

    isMintRequested,
    isMintCompleted,
    isInspectionCompleted,
    showMintButton,
    showBrandSelectorCard,
    showTokenSelectorCard,

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
    tokenBlueprintCardHandlers,

    mintCreatedAtLabel,
    mintCreatedByLabel,
    mintScheduledBurnDateLabel,
    mintMintedAtLabel,
    onChainTxSignature,

    scheduledBurnDate,
    setScheduledBurnDate,
  };
}