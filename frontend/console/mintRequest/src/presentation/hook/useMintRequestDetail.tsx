// frontend/console/mintRequest/src/presentation/hook/useMintRequestDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useInspectionResultCard } from "./useInspectionResultCard";

import type { InspectionBatchDTO } from "../../domain/entity/inspections";
import type { MintDTO } from "../../infrastructure/api/mintRequestApi";
import { completeInspectionByProductionId } from "../../infrastructure/api/mintRequestApi";

import type { ProductBlueprintPatchDTO } from "../../infrastructure/dto/mintRequestLocal.dto";

import { asNonEmptyString } from "../../application/util/primitive";
import {
  toBrandOptionVMs,
  toTokenBlueprintOptionVMs,
} from "../../application/mapper/mintRequestOptionsMapper";
import { validateCompleteInspection } from "../../application/validator/validateCompleteInspection";
import { validateMintRequestSubmit } from "../../application/validator/validateMintRequestSubmit";

import type {
  BrandOptionVM as BrandOption,
  TokenBlueprintOptionVM as TokenBlueprintOption,
  ProductBlueprintCardVM as ProductBlueprintCardViewModel,
  TokenBlueprintCardVM as TokenBlueprintCardViewModel,
  TokenBlueprintCardHandlersVM as TokenBlueprintCardHandlers,
} from "../viewModel/mintRequestDetail.vm";

import { useMintInfo } from "./useMintRequestDetail.mintSelectors";
import { useMintAutoSelection } from "./useMintRequestDetail.useMintAutoSelection";
import { useTokenBlueprintPatch } from "./useMintRequestDetail.useTokenBlueprintPatch";
import {
  buildMintLabels,
  buildProductBlueprintCardView,
  buildTokenBlueprintCardHandlers,
  buildTokenBlueprintCardVm,
} from "./useMintRequestDetail.viewModels";

import { mintRequestContainer } from "../di/mintRequestContainer";
import { getMintRequestDetail } from "../../application/usecase/getMintRequestDetail";
import { getProductBlueprintPatch } from "../../application/usecase/getProductBlueprintPatch";
import { listBrandsForMint } from "../../application/usecase/listBrandsForMint";
import { listTokenBlueprintsByBrand } from "../../application/usecase/listTokenBlueprintsByBrand";
import { submitMintRequestAndRefresh } from "../../application/usecase/submitMintRequestAndRefresh";

export function useMintRequestDetail() {
  const navigate = useNavigate();

  /**
   * route 名は requestId のままでも、実体は productionId。
   */
  const { requestId } = useParams<{ requestId: string }>();

  const productionId = React.useMemo(() => {
    return String(requestId ?? "").trim();
  }, [requestId]);

  const { mintRequestRepo } = React.useMemo(() => mintRequestContainer(), []);

  const [inspectionBatch, setInspectionBatch] =
    React.useState<InspectionBatchDTO | null>(null);

  const [mintDTO, setMintDTO] = React.useState<MintDTO | null>(null);

  const [productBlueprintId, setProductBlueprintId] =
    React.useState<string>("");

  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const [pbPatch, setPbPatch] =
    React.useState<ProductBlueprintPatchDTO | null>(null);
  const [pbPatchLoading, setPbPatchLoading] = React.useState(false);
  const [pbPatchError, setPbPatchError] = React.useState<string | null>(null);

  const [brandOptions, setBrandOptions] = React.useState<BrandOption[]>([]);
  const [selectedBrandId, setSelectedBrandId] = React.useState<string>("");

  const [tokenBlueprintOptions, setTokenBlueprintOptions] = React.useState<
    TokenBlueprintOption[]
  >([]);
  const [selectedTokenBlueprintId, setSelectedTokenBlueprintId] =
    React.useState<string>("");

  const [scheduledBurnDate, setScheduledBurnDate] = React.useState<string>("");
  const [isMinting, setIsMinting] = React.useState(false);
  const [isCompletingInspection, setIsCompletingInspection] =
    React.useState(false);

  const title = `ミント申請詳細`;

  const selectedBrandName = React.useMemo(() => {
    if (!selectedBrandId) return "";
    return brandOptions.find((b) => b.id === selectedBrandId)?.name ?? "";
  }, [brandOptions, selectedBrandId]);

  const reloadDetail = React.useCallback(async () => {
    if (!productionId) return;

    const detail = await getMintRequestDetail(mintRequestRepo, productionId);

    setInspectionBatch((detail.inspectionBatch ?? null) as any);
    setMintDTO((detail.mintDTO ?? null) as any);
    setProductBlueprintId(
      detail.productBlueprintId ? detail.productBlueprintId : "",
    );
  }, [productionId, mintRequestRepo]);

  React.useEffect(() => {
    if (!productionId) return;

    let cancelled = false;

    const run = async () => {
      setLoading(true);
      setError(null);

      try {
        const detail = await getMintRequestDetail(
          mintRequestRepo,
          productionId,
        );

        if (cancelled) return;

        setInspectionBatch((detail.inspectionBatch ?? null) as any);
        setMintDTO((detail.mintDTO ?? null) as any);
        setProductBlueprintId(
          detail.productBlueprintId ? detail.productBlueprintId : "",
        );
      } catch (e: any) {
        if (!cancelled) {
          setError(e?.message ?? "検査結果の取得に失敗しました");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    run();

    return () => {
      cancelled = true;
    };
  }, [productionId, mintRequestRepo]);

  React.useEffect(() => {
    if (!productBlueprintId) return;

    let cancelled = false;

    const run = async () => {
      setPbPatchLoading(true);
      setPbPatchError(null);

      try {
        const patch = await getProductBlueprintPatch(
          mintRequestRepo,
          productBlueprintId,
        );

        if (!cancelled) {
          setPbPatch((patch ?? null) as any);
        }
      } catch (e: any) {
        if (!cancelled) {
          setPbPatchError(
            e?.message ?? "プロダクト基本情報の取得に失敗しました",
          );
        }
      } finally {
        if (!cancelled) {
          setPbPatchLoading(false);
        }
      }
    };

    run();

    return () => {
      cancelled = true;
    };
  }, [productBlueprintId, mintRequestRepo]);

  const batchForInspectionCard = React.useMemo(() => {
    if (!inspectionBatch) return undefined;

    return {
      ...(inspectionBatch as any),
      productBlueprintPatch: pbPatch ?? null,
    };
  }, [inspectionBatch, pbPatch]);

  const inspectionCardData = useInspectionResultCard({
    batch: batchForInspectionCard,
  });

  const totalMintQuantity = inspectionCardData.totalPassed;

  const productBlueprintCardView: ProductBlueprintCardViewModel | null =
    React.useMemo(() => buildProductBlueprintCardView(pbPatch), [pbPatch]);

  const MINT_REQUEST_MANAGEMENT_PATH = "/mintRequest";

  const onBack = React.useCallback(() => {
    navigate(MINT_REQUEST_MANAGEMENT_PATH);
  }, [navigate]);

  React.useEffect(() => {
    let cancelled = false;

    const run = async () => {
      try {
        const brands = await listBrandsForMint(mintRequestRepo);
        if (cancelled) return;

        setBrandOptions(toBrandOptionVMs(brands));
      } catch {
        // noop
      }
    };

    run();

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
        const opts = await listTokenBlueprintsByBrand(
          mintRequestRepo,
          brandId,
        );

        setTokenBlueprintOptions(toTokenBlueprintOptionVMs(opts));
        setSelectedTokenBlueprintId("");
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
  } = useMintInfo({ mintDTO, inspectionBatch, pbPatch });

  const inspectionStatus = React.useMemo(() => {
    return String((inspectionBatch as any)?.status ?? "").trim();
  }, [inspectionBatch]);

  const isInspectionCompleted = React.useMemo(() => {
    return inspectionStatus === "completed";
  }, [inspectionStatus]);

  const showCompleteInspectionButton = React.useMemo(() => {
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

  const showMintButton = !isMintRequested;
  const showBrandSelectorCard = !isMintRequested;
  const showTokenSelectorCard = !isMintRequested;

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

  const tokenBlueprintIdForPatch = React.useMemo(() => {
    const a = asNonEmptyString(selectedTokenBlueprintId);
    if (a) return a;

    const b = asNonEmptyString(mintRequestedTokenBlueprintId);
    return b ? b : "";
  }, [selectedTokenBlueprintId, mintRequestedTokenBlueprintId]);

  const { tokenBlueprintPatch } =
    useTokenBlueprintPatch(tokenBlueprintIdForPatch);

  const handleCompleteInspection = React.useCallback(async () => {
    if (isCompletingInspection || isMinting) {
      return;
    }

    const validation = validateCompleteInspection({
      inspectionBatch,
      productionId,
    });

    if (!validation.ok) {
      alert(validation.message);
      return;
    }

    const ok = window.confirm(
      "検品を完了します。未入力の検品結果は合格として確定されます。よろしいですか？",
    );

    if (!ok) return;

    setIsCompletingInspection(true);

    try {
      const updatedBatch = await completeInspectionByProductionId(
        validation.productionId,
      );

      if (updatedBatch) {
        setInspectionBatch(updatedBatch as any);
      }

      await reloadDetail();

      alert("検品を完了しました。");
    } catch (e: any) {
      alert(
        `検品完了に失敗しました: ${
          e?.message ?? "不明なエラーが発生しました"
        }`,
      );
    } finally {
      setIsCompletingInspection(false);
    }
  }, [
    inspectionBatch,
    isCompletingInspection,
    isMinting,
    productionId,
    reloadDetail,
  ]);

  const handleMint = React.useCallback(async () => {
    if (isMinting) {
      return;
    }

    const validation = validateMintRequestSubmit({
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

    try {
      const { updatedBatch, refreshedMint } = await submitMintRequestAndRefresh(
        validation.productionId,
        validation.tokenBlueprintId,
        scheduledBurnDate || undefined,
      );

      if (updatedBatch) {
        setInspectionBatch(updatedBatch as any);
      }

      if (refreshedMint) {
        setMintDTO(refreshedMint as any);
      }

      alert(
        `ミントが完了しました（生産ID: ${validation.productionId} / ミント数: ${totalMintQuantity}）`,
      );

      navigate(0);
    } catch (e: any) {
      alert(
        `ミント申請に失敗しました: ${
          e?.message ?? "不明なエラーが発生しました"
        }`,
      );
    } finally {
      setIsMinting(false);
    }
  }, [
    inspectionBatch,
    isInspectionCompleted,
    isMinting,
    navigate,
    productionId,
    scheduledBurnDate,
    selectedTokenBlueprintId,
    totalMintQuantity,
  ]);

  const handleSelectTokenBlueprint = React.useCallback(
    (tokenBlueprintId: string) => {
      setSelectedTokenBlueprintId(tokenBlueprintId);
    },
    [],
  );

  const selectedTokenBlueprint = React.useMemo(
    () =>
      tokenBlueprintOptions.find((tb) => tb.id === selectedTokenBlueprintId) ??
      null,
    [tokenBlueprintOptions, selectedTokenBlueprintId],
  );

  const tokenBlueprintCardVm: TokenBlueprintCardViewModel | null =
    React.useMemo(
      () =>
        buildTokenBlueprintCardVm({
          selectedTokenBlueprint,
          tokenBlueprintIdForPatch,
          selectedBrandName,
          tokenBlueprintPatch: tokenBlueprintPatch as any,
          pbPatch,
          brandOptions,
        }),
      [
        selectedTokenBlueprint,
        tokenBlueprintIdForPatch,
        selectedBrandName,
        tokenBlueprintPatch,
        pbPatch,
        brandOptions,
      ],
    );

  const tokenBlueprintCardHandlers: TokenBlueprintCardHandlers =
    React.useMemo(
      () => buildTokenBlueprintCardHandlers(tokenBlueprintCardVm?.iconUrl),
      [tokenBlueprintCardVm?.iconUrl],
    );

  const {
    mintCreatedAtLabel,
    mintCreatedByLabel,
    mintScheduledBurnDateLabel,
    mintMintedAtLabel,
    onChainTxSignature,
  } = React.useMemo(
    () => buildMintLabels({ mint, requestedByName }),
    [mint, requestedByName],
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
    isInspectionCompleted,
    showMintButton,
    showBrandSelectorCard,
    showTokenSelectorCard,

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