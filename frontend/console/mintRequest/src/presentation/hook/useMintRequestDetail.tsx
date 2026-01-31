// frontend/console/mintRequest/src/presentation/hook/useMintRequestDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useInspectionResultCard } from "./useInspectionResultCard";

import type { InspectionBatchDTO, MintDTO } from "../../infrastructure/api/mintRequestApi";
import type { ProductBlueprintPatchDTO } from "../../infrastructure/dto/mintRequestLocal.dto";

import { asNonEmptyString } from "../../application/mapper/modelInspectionMapper";

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

// ✅ B設計: application/usecase + DI + port(repo interface)
// NOTE: 今回は submit だけ submitMintRequestAndRefresh（HTTP直呼び）を採用する前提
import { mintRequestContainer } from "../di/mintRequestContainer";
import { getMintRequestDetail } from "../../application/usecase/getMintRequestDetail";
import { getProductBlueprintPatch } from "../../application/usecase/getProductBlueprintPatch";
import { listBrandsForMint } from "../../application/usecase/listBrandsForMint";
import { listTokenBlueprintsByBrand } from "../../application/usecase/listTokenBlueprintsByBrand";

// ✅ 採用する方（重複解消）：repo引数なし
import { submitMintRequestAndRefresh } from "../../application/usecase/submitMintRequestAndRefresh";

export function useMintRequestDetail() {
  const navigate = useNavigate();
  const { requestId } = useParams<{ requestId: string }>();

  // DI（infra repo はここで組み立て、hook内ではusecase経由でしか触らない）
  // ※ submitMintRequestAndRefresh は repo を使わないが、他のusecaseで必要なので残す
  const { mintRequestRepo } = React.useMemo(() => mintRequestContainer(), []);

  const [inspectionBatch, setInspectionBatch] =
    React.useState<InspectionBatchDTO | null>(null);

  const [mintDTO, setMintDTO] = React.useState<MintDTO | null>(null);

  const [productBlueprintId, setProductBlueprintId] = React.useState<string>("");

  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const [pbPatch, setPbPatch] = React.useState<ProductBlueprintPatchDTO | null>(null);
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

  const title = `ミント申請詳細`;

  const selectedBrandName = React.useMemo(() => {
    if (!selectedBrandId) return "";
    return brandOptions.find((b) => b.id === selectedBrandId)?.name ?? "";
  }, [brandOptions, selectedBrandId]);

  // ① 初期化: inspection + mintDTO + productBlueprintId を解決（usecase化）
  React.useEffect(() => {
    if (!requestId) return;

    let cancelled = false;

    const run = async () => {
      setLoading(true);
      setError(null);

      try {
        const rid = String(requestId).trim();

        const detail = await getMintRequestDetail(mintRequestRepo, rid);
        if (cancelled) return;

        setInspectionBatch((detail.inspectionBatch ?? null) as any);
        setMintDTO((detail.mintDTO ?? null) as any);
        setProductBlueprintId(detail.productBlueprintId ? detail.productBlueprintId : "");
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
  }, [requestId, mintRequestRepo]);

  // ② productBlueprintId が解決できたら Patch を取得（usecase化）
  React.useEffect(() => {
    if (!productBlueprintId) return;

    let cancelled = false;

    const run = async () => {
      setPbPatchLoading(true);
      setPbPatchError(null);
      try {
        const patch = await getProductBlueprintPatch(mintRequestRepo, productBlueprintId);
        if (!cancelled) {
          setPbPatch((patch ?? null) as any);
        }
      } catch (e: any) {
        if (!cancelled) {
          setPbPatchError(e?.message ?? "プロダクト基本情報の取得に失敗しました");
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

  // ③ 検査カード用（★ pbPatch を batch に合成して渡す：displayOrder のソース）
  const batchForInspectionCard = React.useMemo(() => {
    if (!inspectionBatch) return undefined;

    return {
      ...(inspectionBatch as any),
      // useInspectionResultCard 側が参照するキー名に合わせて注入
      productBlueprintPatch: pbPatch ?? null,
    };
  }, [inspectionBatch, pbPatch]);

  const inspectionCardData = useInspectionResultCard({
    batch: batchForInspectionCard,
  });

  const totalMintQuantity = inspectionCardData.totalPassed;

  // ④ ProductBlueprintCard 用 VM
  const productBlueprintCardView: ProductBlueprintCardViewModel | null = React.useMemo(
    () => buildProductBlueprintCardView(pbPatch),
    [pbPatch],
  );

  const MINT_REQUEST_MANAGEMENT_PATH = "/mintRequest";
  const onBack = React.useCallback(() => {
    navigate(MINT_REQUEST_MANAGEMENT_PATH);
  }, [navigate]);

  // ⑤ ブランド一覧（usecase化）
  React.useEffect(() => {
    let cancelled = false;

    const run = async () => {
      try {
        const brands = await listBrandsForMint(mintRequestRepo);
        if (cancelled) return;

        setBrandOptions(
          (brands ?? []).map((b) => ({
            id: String((b as any)?.id ?? "").trim(),
            name: String((b as any)?.name ?? "").trim(),
          })) as any,
        );
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
        // tokenBlueprintPatch は “詳細表示の正” なので消さない
        return;
      }

      try {
        const opts = await listTokenBlueprintsByBrand(mintRequestRepo, brandId);
        setTokenBlueprintOptions(
          (opts ?? []).map((tb) => ({
            id: String((tb as any)?.id ?? "").trim(),
            name: String((tb as any)?.name ?? "").trim(),
            symbol: String((tb as any)?.symbol ?? "").trim(),
            iconUrl: asNonEmptyString((tb as any)?.iconUrl) || undefined,
          })) as any,
        );
        setSelectedTokenBlueprintId("");
      } catch {
        setTokenBlueprintOptions([]);
        setSelectedTokenBlueprintId("");
      }
    },
    [mintRequestRepo],
  );

  // ============================================================
  // mint 情報（mintDTO 優先）
  // ============================================================

  const {
    mint,
    hasMint,
    isMintRequested,
    requestedByName,
    mintRequestedTokenBlueprintId,
    mintRequestedBrandId,
  } = useMintInfo({ mintDTO, inspectionBatch, pbPatch });

  // minted=true のときだけ非表示
  const showMintButton = !isMintRequested;
  const showBrandSelectorCard = !isMintRequested;
  const showTokenSelectorCard = !isMintRequested;

  // mint 情報に基づく「初回だけ」自動選択
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

  // ============================================================
  // TokenBlueprintPatch を “正” として取得
  // ============================================================

  const tokenBlueprintIdForPatch = React.useMemo(() => {
    const a = asNonEmptyString(selectedTokenBlueprintId);
    if (a) return a;
    const b = asNonEmptyString(mintRequestedTokenBlueprintId);
    return b ? b : "";
  }, [selectedTokenBlueprintId, mintRequestedTokenBlueprintId]);

  const { tokenBlueprintPatch } = useTokenBlueprintPatch(tokenBlueprintIdForPatch);

  // ミント申請（未申請時のみ）: submitMintRequestAndRefresh を採用（repo引数なし）
  const handleMint = React.useCallback(async () => {
    if (!inspectionBatch) {
      alert("検査バッチ情報が取得できていません。");
      return;
    }

    if (!selectedTokenBlueprintId) {
      alert("トークン設計を選択してください。");
      return;
    }

    const productionId = (inspectionBatch as any).productionId ?? requestId ?? "";
    if (!productionId) {
      alert("productionId が特定できません。");
      return;
    }

    try {
      const { updatedBatch, refreshedMint } = await submitMintRequestAndRefresh(
        productionId,
        selectedTokenBlueprintId,
        scheduledBurnDate || undefined,
      );

      if (updatedBatch) {
        setInspectionBatch(updatedBatch as any);
      }

      if (refreshedMint) {
        setMintDTO(refreshedMint as any);
      }

      alert(
        `ミント申請を登録しました（生産ID: ${productionId} / ミント数: ${totalMintQuantity}）`,
      );
    } catch (e: any) {
      alert(`ミント申請に失敗しました: ${e?.message ?? "不明なエラーが発生しました"}`);
    }
  }, [
    inspectionBatch,
    selectedTokenBlueprintId,
    requestId,
    totalMintQuantity,
    scheduledBurnDate,
  ]);

  const handleSelectTokenBlueprint = React.useCallback((tokenBlueprintId: string) => {
    setSelectedTokenBlueprintId(tokenBlueprintId);
  }, []);

  const selectedTokenBlueprint = React.useMemo(
    () =>
      tokenBlueprintOptions.find((tb) => tb.id === selectedTokenBlueprintId) ?? null,
    [tokenBlueprintOptions, selectedTokenBlueprintId],
  );

  const tokenBlueprintCardVm: TokenBlueprintCardViewModel | null = React.useMemo(
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

  const tokenBlueprintCardHandlers: TokenBlueprintCardHandlers = React.useMemo(
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

    // ★ mint 情報
    hasMint,

    // ✅ 表示制御
    isMintRequested,
    showMintButton,
    showBrandSelectorCard,
    showTokenSelectorCard,

    // ✅ requester display name
    requestedByName,

    productBlueprintCardView,
    pbPatchLoading,
    pbPatchError,

    // ブランド選択カード用
    brandOptions,
    selectedBrandId,
    selectedBrandName,
    handleSelectBrand,

    // トークン設計一覧カード用
    tokenBlueprintOptions,
    selectedTokenBlueprintId,
    handleSelectTokenBlueprint,

    // mint 情報表示用ラベル
    tokenBlueprintCardVm,
    tokenBlueprintCardHandlers,
    mintCreatedAtLabel,
    mintCreatedByLabel,
    mintScheduledBurnDateLabel,
    mintMintedAtLabel,
    onChainTxSignature,

    // 焼却予定日
    scheduledBurnDate,
    setScheduledBurnDate,
  };
}
