// frontend/console/mintRequest/src/presentation/hook/useMintRequestDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useInspectionResultCard } from "./useInspectionResultCard";

import type { InspectionBatchDTO, MintDTO } from "../../infrastructure/api/mintRequestApi";

import type { ProductBlueprintPatchDTO } from "../../infrastructure/dto/mintRequestLocal.dto";

import {
  fetchInspectionByProductionIdHTTP,
  fetchMintByInspectionIdHTTP,
  fetchProductBlueprintIdByProductionIdHTTP,
  fetchProductBlueprintPatchHTTP,
  fetchBrandsForMintHTTP,
  fetchTokenBlueprintsByBrandHTTP,
  postMintRequestHTTP,
} from "../../infrastructure/repository";

import {
  fetchInventoryTokenBlueprintPatch,
  type TokenBlueprintPatchDTO,
} from "../../infrastructure/adapter/inventoryTokenBlueprintPatch";

import { safeDateLabelJa, safeDateTimeLabelJa } from "../../../../shell/src/shared/util/dateJa";
import { asNonEmptyString } from "../../application/mapper/modelInspectionMapper";

import {
  extractMintInfoFromBatch,
  extractMintInfoFromMintDTO,
  type MintInfo,
} from "../../application/mapper/mintInfoMapper";

import type {
  BrandOptionVM as BrandOption,
  TokenBlueprintOptionVM as TokenBlueprintOption,
  ProductBlueprintCardVM as ProductBlueprintCardViewModel,
  TokenBlueprintCardVM as TokenBlueprintCardViewModel,
  TokenBlueprintCardHandlersVM as TokenBlueprintCardHandlers,
} from "../viewModel/mintRequestDetail.vm";

function extractProductBlueprintIdFromBatch(batch: any): string {
  if (!batch) return "";
  const v = batch.productBlueprintId ?? batch.productBlueprint?.id ?? "";
  return asNonEmptyString(v);
}

async function resolveProductBlueprintIdByRequestId(
  requestId: string,
  batch: InspectionBatchDTO | null,
): Promise<string> {
  const rid = String(requestId ?? "").trim();
  if (!rid) return "";

  const pbFromBatch = extractProductBlueprintIdFromBatch(batch as any);
  if (pbFromBatch) return pbFromBatch;

  const pbFromProduction = await fetchProductBlueprintIdByProductionIdHTTP(rid).catch(
    () => null,
  );
  return asNonEmptyString(pbFromProduction);
}

async function fetchProductBlueprintPatchById(
  productBlueprintId: string,
): Promise<ProductBlueprintPatchDTO | null> {
  const id = String(productBlueprintId ?? "").trim();
  if (!id) return null;

  const patch = await fetchProductBlueprintPatchHTTP(id);
  return (patch ?? null) as any;
}

async function fetchBrandOptionsForMint(): Promise<BrandOption[]> {
  const brands = await fetchBrandsForMintHTTP();
  return (brands ?? []).map((b: any) => ({
    id: String(b?.id ?? "").trim(),
    name: String(b?.name ?? "").trim(),
  }));
}

async function fetchTokenBlueprintOptionsByBrand(
  brandId: string,
): Promise<TokenBlueprintOption[]> {
  const id = String(brandId ?? "").trim();
  if (!id) return [];

  const list = await fetchTokenBlueprintsByBrandHTTP(id);

  return (list ?? []).map((tb: any) => ({
    id: String(tb?.id ?? "").trim(),
    name: String(tb?.name ?? "").trim(),
    symbol: String(tb?.symbol ?? "").trim(),
    iconUrl: asNonEmptyString(tb?.iconUrl) || undefined,
  }));
}

async function fetchTokenBlueprintPatchById(
  tokenBlueprintId: string,
): Promise<TokenBlueprintPatchDTO | null> {
  const tbId = String(tokenBlueprintId ?? "").trim();
  if (!tbId) return null;
  return await fetchInventoryTokenBlueprintPatch(tbId);
}

async function submitMintRequestAndRefresh(
  productionId: string,
  tokenBlueprintId: string,
  scheduledBurnDate?: string,
): Promise<{
  updatedBatch: InspectionBatchDTO | null;
  refreshedMint: MintDTO | null;
}> {
  const pid = String(productionId ?? "").trim();
  const tbId = String(tokenBlueprintId ?? "").trim();
  if (!pid || !tbId) return { updatedBatch: null, refreshedMint: null };

  const updated = await postMintRequestHTTP(pid, tbId, scheduledBurnDate);

  let refreshed: MintDTO | null = null;
  try {
    refreshed = await fetchMintByInspectionIdHTTP(pid).catch(() => null);
  } catch {
    refreshed = null;
  }

  return {
    updatedBatch: (updated ?? null) as any,
    refreshedMint: (refreshed ?? null) as any,
  };
}

export function useMintRequestDetail() {
  const navigate = useNavigate();
  const { requestId } = useParams<{ requestId: string }>();

  const [inspectionBatch, setInspectionBatch] =
    React.useState<InspectionBatchDTO | null>(null);

  const [mintDTO, setMintDTO] = React.useState<MintDTO | null>(null);

  const [productBlueprintId, setProductBlueprintId] = React.useState<string>("");

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

  // tokenBlueprint patch（iconUrl/description 等の “正” をここから取る）
  const [tokenBlueprintPatch, setTokenBlueprintPatch] =
    React.useState<TokenBlueprintPatchDTO | null>(null);

  const title = `ミント申請詳細`;

  const selectedBrandName = React.useMemo(() => {
    if (!selectedBrandId) return "";
    return brandOptions.find((b) => b.id === selectedBrandId)?.name ?? "";
  }, [brandOptions, selectedBrandId]);

  // ① 初期化: inspection + mintDTO + productBlueprintId を解決
  React.useEffect(() => {
    if (!requestId) return;

    let cancelled = false;

    const run = async () => {
      setLoading(true);
      setError(null);

      try {
        const rid = String(requestId).trim();

        const [batch, mint] = await Promise.all([
          fetchInspectionByProductionIdHTTP(rid).catch(() => null),
          fetchMintByInspectionIdHTTP(rid).catch(() => null),
        ]);

        if (cancelled) return;

        setInspectionBatch((batch ?? null) as any);
        setMintDTO((mint ?? null) as any);

        const resolvedPB = await resolveProductBlueprintIdByRequestId(
          rid,
          (batch ?? null) as any,
        );
        if (cancelled) return;

        setProductBlueprintId(resolvedPB ? resolvedPB : "");
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
  }, [requestId]);

  // ② productBlueprintId が解決できたら Patch を取得
  React.useEffect(() => {
    if (!productBlueprintId) return;

    let cancelled = false;

    const run = async () => {
      setPbPatchLoading(true);
      setPbPatchError(null);
      try {
        const patch = await fetchProductBlueprintPatchById(productBlueprintId);
        if (!cancelled) {
          setPbPatch(patch);
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
  }, [productBlueprintId]);

  // ③ 検査カード用
  const inspectionCardData = useInspectionResultCard({
    batch: inspectionBatch ?? undefined,
  });

  const totalMintQuantity = inspectionCardData.totalPassed;

  // ④ ProductBlueprintCard 用 VM（brandName のみ渡す）
  const productBlueprintCardView: ProductBlueprintCardViewModel | null =
    React.useMemo(() => {
      if (!pbPatch) return null;

      return {
        productName: (pbPatch as any)?.productName ?? undefined,
        brand: (pbPatch as any)?.brandName ?? undefined,
        itemType: (pbPatch as any)?.itemType ?? undefined,
        fit: (pbPatch as any)?.fit ?? undefined,
        materials: (pbPatch as any)?.material ?? undefined,
        weight: (pbPatch as any)?.weight ?? undefined,
        washTags: (pbPatch as any)?.qualityAssurance ?? undefined,
        productIdTag: (pbPatch as any)?.productIdTag?.type ?? undefined,
      };
    }, [pbPatch]);

  const MINT_REQUEST_MANAGEMENT_PATH = "/mintRequest";
  const onBack = React.useCallback(() => {
    navigate(MINT_REQUEST_MANAGEMENT_PATH);
  }, [navigate]);

  // ⑤ ブランド一覧
  React.useEffect(() => {
    let cancelled = false;

    const run = async () => {
      try {
        const brands = await fetchBrandOptionsForMint();
        if (!cancelled) {
          setBrandOptions(brands ?? []);
        }
      } catch {
        // noop
      }
    };

    run();
    return () => {
      cancelled = true;
    };
  }, []);

  const handleSelectBrand = React.useCallback(async (brandId: string) => {
    setSelectedBrandId(brandId);

    if (!brandId) {
      setTokenBlueprintOptions([]);
      setSelectedTokenBlueprintId("");
      // tokenBlueprintPatch は “詳細表示の正” なので消さない
      return;
    }

    try {
      const opts = await fetchTokenBlueprintOptionsByBrand(brandId);
      setTokenBlueprintOptions(opts ?? []);
      setSelectedTokenBlueprintId("");
    } catch {
      setTokenBlueprintOptions([]);
      setSelectedTokenBlueprintId("");
    }
  }, []);

  // ============================================================
  // mint 情報（mintDTO 優先）
  // ============================================================

  const mint: MintInfo | null = React.useMemo(() => {
    const fromDTO = extractMintInfoFromMintDTO(mintDTO as any);
    if (fromDTO) return fromDTO;

    const fromBatch = extractMintInfoFromBatch(inspectionBatch as any);
    return fromBatch;
  }, [mintDTO, inspectionBatch]);

  const hasMint = React.useMemo(() => !!mint, [mint]);

  // minted=true のときのみ非表示判定（= mint 完了扱い）
  const isMintRequested = React.useMemo(() => {
    return Boolean(mint?.minted === true);
  }, [mint]);

  // ✅ requestedByName（表示名）
  // - mintInfo が requestedByName を持つ場合はそれを最優先
  // - 次に createdByName
  // - 最後に createdBy（id）
  const requestedByName: string | null = React.useMemo(() => {
    const a = asNonEmptyString((mint as any)?.requestedByName);
    if (a) return a;

    const b = asNonEmptyString((mint as any)?.createdByName);
    if (b) return b;

    const c = asNonEmptyString((mint as any)?.createdBy);
    return c ? c : null;
  }, [mint]);

  const mintRequestedTokenBlueprintId = React.useMemo(() => {
    const v = asNonEmptyString(mint?.tokenBlueprintId);
    return v ? v : "";
  }, [mint]);

  const mintRequestedBrandId = React.useMemo(() => {
    // mint.brandId を最優先。無ければ pbPatch.brandId を fallback
    const fromMint = asNonEmptyString(mint?.brandId);
    if (fromMint) return fromMint;
    const fromPatch = asNonEmptyString((pbPatch as any)?.brandId);
    return fromPatch ? fromPatch : "";
  }, [mint, pbPatch]);

  // mint が存在し、brandId が取れるなら「初回だけ」ブランド自動選択
  React.useEffect(() => {
    if (!hasMint) return;
    if (!mintRequestedBrandId) return;
    if (selectedBrandId) return; // 手動選択を尊重

    (async () => {
      try {
        await handleSelectBrand(mintRequestedBrandId);
      } catch {
        // noop
      }
    })();
  }, [hasMint, mintRequestedBrandId, selectedBrandId, handleSelectBrand]);

  // mint が存在し、tokenBlueprintId が取れるなら「初回だけ」tokenBlueprint 自動選択
  React.useEffect(() => {
    if (!hasMint) return;
    if (!mintRequestedTokenBlueprintId) return;
    if (selectedTokenBlueprintId) return; // 手動選択を尊重
    setSelectedTokenBlueprintId(mintRequestedTokenBlueprintId);
  }, [hasMint, mintRequestedTokenBlueprintId, selectedTokenBlueprintId]);

  // mint が存在し、scheduledBurnDate があるなら「初回だけ」入力欄へ反映（手入力を尊重）
  React.useEffect(() => {
    if (!hasMint) return;
    if (scheduledBurnDate) return; // 既に入力されているなら上書きしない

    const raw = mint?.scheduledBurnDate;
    if (!raw) return;

    const s = String(raw);
    const asDate = s.length >= 10 ? s.slice(0, 10) : s;
    if (asDate) setScheduledBurnDate(asDate);
  }, [hasMint, mint, scheduledBurnDate]);

  // minted=true のときだけ非表示
  const showMintButton = !isMintRequested;
  const showBrandSelectorCard = !isMintRequested;
  const showTokenSelectorCard = !isMintRequested;

  // ============================================================
  // TokenBlueprintPatch を “正” として取得
  // ============================================================

  const tokenBlueprintIdForPatch = React.useMemo(() => {
    const a = asNonEmptyString(selectedTokenBlueprintId);
    if (a) return a;
    const b = asNonEmptyString(mintRequestedTokenBlueprintId);
    return b ? b : "";
  }, [selectedTokenBlueprintId, mintRequestedTokenBlueprintId]);

  React.useEffect(() => {
    if (!tokenBlueprintIdForPatch) {
      setTokenBlueprintPatch(null);
      return;
    }

    let cancelled = false;

    (async () => {
      try {
        const p = await fetchTokenBlueprintPatchById(tokenBlueprintIdForPatch);
        if (cancelled) return;
        setTokenBlueprintPatch((p ?? null) as any);
      } catch {
        if (cancelled) return;
        setTokenBlueprintPatch(null);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [tokenBlueprintIdForPatch]);

  // ミント申請（未申請時のみ）
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
      alert(
        `ミント申請に失敗しました: ${e?.message ?? "不明なエラーが発生しました"}`,
      );
    }
  }, [
    inspectionBatch,
    selectedTokenBlueprintId,
    requestId,
    totalMintQuantity,
    scheduledBurnDate,
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
    React.useMemo(() => {
      const tbId =
        asNonEmptyString(selectedTokenBlueprint?.id) ||
        asNonEmptyString(tokenBlueprintIdForPatch);
      if (!tbId) return null;

      const brandName =
        selectedBrandName ||
        asNonEmptyString((tokenBlueprintPatch as any)?.brandName) ||
        asNonEmptyString((pbPatch as any)?.brandName) ||
        "";

      const name =
        asNonEmptyString((tokenBlueprintPatch as any)?.tokenName) ||
        asNonEmptyString(selectedTokenBlueprint?.name);

      const symbol =
        asNonEmptyString((tokenBlueprintPatch as any)?.symbol) ||
        asNonEmptyString(selectedTokenBlueprint?.symbol);

      const description = asNonEmptyString(
        (tokenBlueprintPatch as any)?.description,
      );

      const iconUrl =
        asNonEmptyString((tokenBlueprintPatch as any)?.iconUrl) ||
        asNonEmptyString(selectedTokenBlueprint?.iconUrl) ||
        undefined;

      return {
        id: tbId,
        name: name || tbId,
        symbol: symbol || "",
        brandId: "",
        brandName,
        description: description || "",
        iconUrl,
        isEditMode: false,
        brandOptions: brandOptions.map((b) => ({ id: b.id, name: b.name })),
      };
    }, [
      selectedTokenBlueprint,
      tokenBlueprintIdForPatch,
      selectedBrandName,
      tokenBlueprintPatch,
      pbPatch,
      brandOptions,
    ]);

  const tokenBlueprintCardHandlers: TokenBlueprintCardHandlers = React.useMemo(
    () => ({
      onPreview: () => {
        const url = tokenBlueprintCardVm?.iconUrl;
        if (url) window.open(url, "_blank", "noopener,noreferrer");
      },
    }),
    [tokenBlueprintCardVm?.iconUrl],
  );

  const mintCreatedAtLabel = React.useMemo(
    () => safeDateTimeLabelJa(mint?.createdAt ?? null, "（未登録）"),
    [mint?.createdAt],
  );

  const mintCreatedByLabel = React.useMemo(() => {
    const name = asNonEmptyString(requestedByName);
    if (name) return name;

    const fallback = asNonEmptyString(mint?.createdBy);
    return fallback ? fallback : "（不明）";
  }, [requestedByName, mint?.createdBy]);

  const mintScheduledBurnDateLabel = React.useMemo(
    () => safeDateLabelJa(mint?.scheduledBurnDate ?? null, "（未設定）"),
    [mint?.scheduledBurnDate],
  );

  const mintMintedAtLabel = React.useMemo(
    () => safeDateTimeLabelJa(mint?.mintedAt ?? null, "（未完了）"),
    [mint?.mintedAt],
  );

  const onChainTxSignature = React.useMemo(
    () => asNonEmptyString(mint?.onChainTxSignature),
    [mint?.onChainTxSignature],
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
