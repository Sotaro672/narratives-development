// frontend/console/mintRequest/src/presentation/hook/useMintRequestDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useInspectionResultCard } from "./useInspectionResultCard";

import type { InspectionBatchDTO } from "../../domain/entity/inspections";
import type { MintDTO } from "../../infrastructure/api/mintRequestApi";
import { completeInspectionByProductionId } from "../../infrastructure/api/mintRequestApi";

import type { ProductBlueprintPatchDTO } from "../../infrastructure/dto/mintRequestLocal.dto";

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
import {
  type TokenBlueprintPatchDTO,
  useTokenBlueprintPatch,
} from "./useMintRequestDetail.useTokenBlueprintPatch";
import {
  buildMintLabels,
  buildProductBlueprintCardView,
  buildTokenBlueprintCardHandlers,
  buildTokenBlueprintCardVm,
} from "./useMintRequestDetail.viewModels";

import { mintRequestContainer } from "../di/mintRequestContainer";
import { getMintRequestDetail } from "../../application/usecase/getMintRequestDetail";
import { listBrandsForMint } from "../../application/usecase/listBrandsForMint";
import { listTokenBlueprintsByBrand } from "../../application/usecase/listTokenBlueprintsByBrand";
import { submitMintRequestAndRefresh } from "../../application/usecase/submitMintRequestAndRefresh";

const DEBUG_MINT_REQUEST_DETAIL = true;

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

  const [detailTokenBlueprintPatch, setDetailTokenBlueprintPatch] =
    React.useState<TokenBlueprintPatchDTO | null>(null);

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

  const applyDetail = React.useCallback((detail: any) => {
    if (DEBUG_MINT_REQUEST_DETAIL) {
      console.log("[mintRequestDetail] applyDetail:raw", detail);
    }

    setInspectionBatch(
      (detail?.inspection ?? detail?.inspectionBatch ?? null) as any,
    );

    setMintDTO((detail?.mint ?? detail?.mintDTO ?? null) as any);

    setProductBlueprintId(
      detail?.productBlueprintId
        ? String(detail.productBlueprintId).trim()
        : "",
    );

    setPbPatch((detail?.productBlueprintPatch ?? null) as any);
    setDetailTokenBlueprintPatch((detail?.tokenBlueprintPatch ?? null) as any);
  }, []);

  const reloadDetail = React.useCallback(async () => {
    if (!productionId) return;

    const detail = await getMintRequestDetail(mintRequestRepo, productionId);
    applyDetail(detail);
  }, [productionId, mintRequestRepo, applyDetail]);

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

        applyDetail(detail);
      } catch (e: any) {
        if (!cancelled) {
          setInspectionBatch(null);
          setMintDTO(null);
          setProductBlueprintId("");
          setPbPatch(null);
          setDetailTokenBlueprintPatch(null);
          setError(e?.message ?? "検査結果の取得に失敗しました");

          if (DEBUG_MINT_REQUEST_DETAIL) {
            console.error("[mintRequestDetail] get detail failed", e);
          }
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
  }, [productionId, mintRequestRepo, applyDetail]);

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

        const brandVms = toBrandOptionVMs(brands);
        setBrandOptions(brandVms);

        if (DEBUG_MINT_REQUEST_DETAIL) {
          console.log("[mintRequestDetail] brands", {
            raw: brands,
            vms: brandVms,
          });
        }
      } catch (e) {
        if (DEBUG_MINT_REQUEST_DETAIL) {
          console.error("[mintRequestDetail] brands failed", e);
        }
      }
    };

    run();

    return () => {
      cancelled = true;
    };
  }, [mintRequestRepo]);

  const {
    mint,
    hasMint,
    isMintRequested,
    requestedByName,
    mintRequestedTokenBlueprintId,
    mintRequestedBrandId,
  } = useMintInfo({ mintDTO, inspectionBatch, pbPatch });

  const handleSelectBrand = React.useCallback(
    async (brandId: string) => {
      const nextBrandId = String(brandId ?? "").trim();

      if (DEBUG_MINT_REQUEST_DETAIL) {
        console.log("[mintRequestDetail] handleSelectBrand:start", {
          brandId,
          nextBrandId,
          mintRequestedTokenBlueprintId,
        });
      }

      setSelectedBrandId(nextBrandId);

      if (!nextBrandId) {
        setTokenBlueprintOptions([]);
        setSelectedTokenBlueprintId("");

        if (DEBUG_MINT_REQUEST_DETAIL) {
          console.log("[mintRequestDetail] handleSelectBrand:empty brand");
        }

        return;
      }

      try {
        const opts = await listTokenBlueprintsByBrand(
          mintRequestRepo,
          nextBrandId,
        );

        if (DEBUG_MINT_REQUEST_DETAIL) {
          console.log("[mintRequestDetail] token opts", opts);
        }

        const vms = toTokenBlueprintOptionVMs(opts);

        if (DEBUG_MINT_REQUEST_DETAIL) {
          console.log("[mintRequestDetail] token vms", vms);
        }

        setTokenBlueprintOptions(vms);

        setSelectedTokenBlueprintId((current) => {
          let nextSelectedTokenBlueprintId = "";

          if (current && vms.some((tb) => tb.id === current)) {
            nextSelectedTokenBlueprintId = current;
          } else if (
            mintRequestedTokenBlueprintId &&
            vms.some((tb) => tb.id === mintRequestedTokenBlueprintId)
          ) {
            nextSelectedTokenBlueprintId = mintRequestedTokenBlueprintId;
          }

          if (DEBUG_MINT_REQUEST_DETAIL) {
            console.log("[mintRequestDetail] select token after brand", {
              current,
              mintRequestedTokenBlueprintId,
              nextSelectedTokenBlueprintId,
              existsCurrent: Boolean(
                current && vms.some((tb) => tb.id === current),
              ),
              existsRequested: Boolean(
                mintRequestedTokenBlueprintId &&
                  vms.some((tb) => tb.id === mintRequestedTokenBlueprintId),
              ),
            });
          }

          return nextSelectedTokenBlueprintId;
        });
      } catch (e) {
        setTokenBlueprintOptions([]);
        setSelectedTokenBlueprintId("");

        if (DEBUG_MINT_REQUEST_DETAIL) {
          console.error("[mintRequestDetail] token opts failed", e);
        }
      }
    },
    [mintRequestRepo, mintRequestedTokenBlueprintId],
  );

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

  /**
   * 未mint時は選択UIとして表示。
   * mint済みでも selectedBrandId がある場合は、取得済み tokenBlueprintOptions を確認できるよう表示する。
   */
  const showTokenSelectorCard = !isMintRequested || Boolean(selectedBrandId);

  useMintAutoSelection({
    hasMint,
    mintRequestedBrandId,
    selectedBrandId,
    handleSelectBrand,
    mintRequestedTokenBlueprintId,
    selectedTokenBlueprintId,
    setSelectedTokenBlueprintId,
    tokenBlueprintOptions,
    mint,
    scheduledBurnDate,
    setScheduledBurnDate,
  });

  const tokenBlueprintIdForPatch = React.useMemo(() => {
    if (selectedTokenBlueprintId) return selectedTokenBlueprintId;
    return mintRequestedTokenBlueprintId || "";
  }, [selectedTokenBlueprintId, mintRequestedTokenBlueprintId]);

  const { tokenBlueprintPatch } = useTokenBlueprintPatch(
    mintRequestRepo,
    tokenBlueprintIdForPatch,
    detailTokenBlueprintPatch,
  );

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
      if (DEBUG_MINT_REQUEST_DETAIL) {
        console.log("[mintRequestDetail] manual select token", {
          tokenBlueprintId,
        });
      }

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
          tokenBlueprintPatch,
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

  React.useEffect(() => {
    if (!DEBUG_MINT_REQUEST_DETAIL) return;

    console.log("[mintRequestDetail] token final", {
      selectedBrandId,
      selectedBrandName,
      mintRequestedBrandId,
      selectedTokenBlueprintId,
      mintRequestedTokenBlueprintId,
      tokenBlueprintOptions,
      selectedTokenBlueprint,
      tokenBlueprintIdForPatch,
      detailTokenBlueprintPatch,
      tokenBlueprintPatch,
      tokenBlueprintCardVm,
    });
  }, [
    selectedBrandId,
    selectedBrandName,
    mintRequestedBrandId,
    selectedTokenBlueprintId,
    mintRequestedTokenBlueprintId,
    tokenBlueprintOptions,
    selectedTokenBlueprint,
    tokenBlueprintIdForPatch,
    detailTokenBlueprintPatch,
    tokenBlueprintPatch,
    tokenBlueprintCardVm,
  ]);

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
    pbPatchLoading: loading,
    pbPatchError: error,

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