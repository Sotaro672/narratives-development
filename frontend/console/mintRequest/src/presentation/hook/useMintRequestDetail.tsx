// frontend/console/mintRequest/src/presentation/hook/useMintRequestDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useInspectionResultCard } from "./useInspectionResultCard";

import type {
  InspectionBatchDTO,
  MintDTO,
} from "../../infrastructure/api/mintRequestApi";

// ✅ Repository を直接呼ぶ
import {
  fetchInspectionByProductionIdHTTP,
  fetchProductBlueprintPatchHTTP,
  fetchBrandsForMintHTTP,
  fetchTokenBlueprintsByBrandHTTP,
  postMintRequestHTTP,
  fetchMintByInspectionIdHTTP,
  fetchProductBlueprintIdByProductionIdHTTP, // ★追加
} from "../../infrastructure/repository/mintRequestRepositoryHTTP";

// -------------------------------
// Local DTOs (hook 内で完結させる)
// -------------------------------

export type ProductBlueprintPatchDTO = {
  productName?: string | null;
  brandId?: string | null;
  brandName?: string | null;

  itemType?: string | null;
  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;
  productIdTag?: { type?: string | null } | null;
  assigneeId?: string | null;
};

export type BrandForMintDTO = {
  id: string;
  name: string;
};

export type TokenBlueprintForMintDTO = {
  id: string;
  name: string;
  symbol: string;
  iconUrl?: string;
};

export type ProductBlueprintCardViewModel = {
  productName?: string;
  brand?: string;
  itemType?: string;
  fit?: string;
  materials?: string;
  weight?: number;
  washTags?: string[];
  productIdTag?: string;
};

export type BrandOption = {
  id: string;
  name: string;
};

export type TokenBlueprintOption = {
  id: string;
  name: string;
  symbol: string;
  iconUrl?: string;
};

export type MintInfo = {
  id: string;
  brandId: string;
  tokenBlueprintId: string;
  createdBy: string;
  createdAt: string;
  minted: boolean;
  mintedAt?: string | null;
  onChainTxSignature?: string | null;
  scheduledBurnDate?: string | null;
};

const LOG_PREFIX = "[mintRequest/useMintRequestDetail]";
function log(...args: any[]) {
  // eslint-disable-next-line no-console
  console.log(LOG_PREFIX, ...args);
}

function asNonEmptyString(v: any): string {
  return typeof v === "string" && v.trim() ? v.trim() : "";
}

function asMaybeISO(v: any): string {
  if (!v) return "";
  if (typeof v === "string") return v;
  if (v instanceof Date) return v.toISOString();
  return String(v);
}

// -------------------------------
// ★ productBlueprintId 抽出/解決
// -------------------------------

function extractProductBlueprintIdFromBatch(batch: any): string {
  if (!batch) return "";
  const v =
    batch.productBlueprintId ??
    batch.productBlueprintID ??
    batch.ProductBlueprintId ??
    batch.ProductBlueprintID ??
    batch.productBlueprint?.id ??
    batch.productBlueprint?.ID ??
    "";
  return asNonEmptyString(v);
}

// -------------------------------
// data loaders
// -------------------------------

async function loadProductBlueprintPatch(
  productBlueprintId: string,
): Promise<ProductBlueprintPatchDTO | null> {
  const id = String(productBlueprintId ?? "").trim();
  if (!id) return null;

  const patch = await fetchProductBlueprintPatchHTTP(id);

  // ★追加: 受け取れた要素が分かる summary log
  const summary = patch
    ? {
        productName: patch.productName ?? null,
        brandId: patch.brandId ?? null,
        brandName: patch.brandName ?? null,
        itemType: patch.itemType ?? null,
        fit: patch.fit ?? null,
        material: patch.material ?? null,
        weight: patch.weight ?? null,
        qualityAssuranceCount: Array.isArray(patch.qualityAssurance)
          ? patch.qualityAssurance.length
          : 0,
        qualityAssuranceSample: Array.isArray(patch.qualityAssurance)
          ? patch.qualityAssurance.slice(0, 5)
          : [],
        productIdTagType: patch.productIdTag?.type ?? null,
        assigneeId: patch.assigneeId ?? null,
        keys: Object.keys(patch as any),
      }
    : null;

  log("loadProductBlueprintPatch id=", id, "patch=", patch ?? null);
  log("loadProductBlueprintPatch summary=", summary);

  return (patch ?? null) as any;
}

async function loadBrandsForMint(): Promise<BrandForMintDTO[]> {
  const brands = await fetchBrandsForMintHTTP();
  log(
    "loadBrandsForMint length=",
    (brands ?? []).length,
    "sample[0]=",
    (brands ?? [])[0],
  );
  return (brands ?? []) as any;
}

async function loadTokenBlueprintsByBrand(
  brandId: string,
): Promise<TokenBlueprintForMintDTO[]> {
  const id = String(brandId ?? "").trim();
  if (!id) return [];

  const list = await fetchTokenBlueprintsByBrandHTTP(id);
  log(
    "loadTokenBlueprintsByBrand brandId=",
    id,
    "length=",
    (list ?? []).length,
    "sample[0]=",
    (list ?? [])[0],
  );
  return (list ?? []) as any;
}

/**
 * 互換: 現状は個別 TokenBlueprint 詳細 API がないため undefined を返す。
 */
function resolveBlueprintForMintRequest(_requestId?: string) {
  return undefined;
}

// -------------------------------
// ★ MintInfo 解決（mintDTO 優先）
// -------------------------------

function extractMintInfoFromMintDTO(m: any): MintInfo | null {
  if (!m) return null;

  const id = asNonEmptyString(m.id ?? m.ID ?? m.mintId ?? m.MintID);
  const brandId = asNonEmptyString(m.brandId ?? m.BrandID ?? m.BrandId);
  const tokenBlueprintId = asNonEmptyString(
    m.tokenBlueprintId ?? m.TokenBlueprintID ?? m.TokenBlueprintId,
  );
  const createdBy = asNonEmptyString(m.createdBy ?? m.CreatedBy);
  const createdAt = asNonEmptyString(asMaybeISO(m.createdAt ?? m.CreatedAt));
  const minted =
    typeof m.minted === "boolean"
      ? m.minted
      : Boolean(m.mintedAt ?? m.MintedAt);
  const mintedAt = asNonEmptyString(asMaybeISO(m.mintedAt ?? m.MintedAt));
  const onChainTxSignature = asNonEmptyString(
    m.onChainTxSignature ?? m.OnChainTxSignature,
  );
  const scheduledBurnDate = asNonEmptyString(
    asMaybeISO(m.scheduledBurnDate ?? m.ScheduledBurnDate),
  );

  if (!id || !brandId || !tokenBlueprintId || !createdBy || !createdAt)
    return null;

  return {
    id,
    brandId,
    tokenBlueprintId,
    createdBy,
    createdAt,
    minted,
    mintedAt: mintedAt ? mintedAt : null,
    onChainTxSignature: onChainTxSignature ? onChainTxSignature : null,
    scheduledBurnDate: scheduledBurnDate ? scheduledBurnDate : null,
  };
}

// “inspectionBatch に mint が埋め込まれて返る”可能性も一応吸収（ただし今回のログでは null）
function extractMintInfoFromBatch(batch: any): MintInfo | null {
  if (!batch) return null;

  const mintObj =
    batch.mint ?? batch.Mint ?? batch.mintRequest ?? batch.MintRequest ?? null;
  if (!mintObj) return null;

  return extractMintInfoFromMintDTO(mintObj);
}

export function useMintRequestDetail() {
  const navigate = useNavigate();
  const { requestId } = useParams<{ requestId: string }>();

  const [inspectionBatch, setInspectionBatch] =
    React.useState<InspectionBatchDTO | null>(null);

  // ★ 追加: MintDTO を単体取得して detail 情報のソースにする
  const [mintDTO, setMintDTO] = React.useState<MintDTO | null>(null);

  // ★ 追加: productBlueprintId（/mint/inspections に無いので別経路で解決）
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
  const [selectedBrandName, setSelectedBrandName] = React.useState<string>("");

  const [tokenBlueprintOptions, setTokenBlueprintOptions] = React.useState<
    TokenBlueprintOption[]
  >([]);
  const [selectedTokenBlueprintId, setSelectedTokenBlueprintId] =
    React.useState<string>("");

  const [scheduledBurnDate, setScheduledBurnDate] = React.useState<string>("");

  const title = `ミント申請詳細`;

  // ① 初期化: inspection + mintDTO + productBlueprintId を解決
  React.useEffect(() => {
    if (!requestId) return;

    let cancelled = false;

    const run = async () => {
      setLoading(true);
      setError(null);

      try {
        log("load start requestId=", requestId);

        // inspection は 1件取得
        const [batch, mint] = await Promise.all([
          fetchInspectionByProductionIdHTTP(requestId),
          fetchMintByInspectionIdHTTP(requestId).catch((e) => {
            log(
              "fetchMintByInspectionIdHTTP failed -> treat as null",
              e?.message ?? e,
            );
            return null;
          }),
        ]);

        if (cancelled) return;

        setInspectionBatch(batch ?? null);
        log("inspectionBatch set", batch ?? null);
        log("inspectionBatch keys=", batch ? Object.keys(batch as any) : []);

        setMintDTO(mint ?? null);
        log("mintDTO set (by inspectionId)", mint ?? null);

        // ★ productBlueprintId: batchから→無ければ /productions で解決
        const pbFromBatch = extractProductBlueprintIdFromBatch(batch as any);
        log(
          "extractProductBlueprintIdFromBatch =",
          pbFromBatch ? pbFromBatch : "(empty)",
        );

        let resolvedPB = pbFromBatch;
        if (!resolvedPB) {
          const pbFromProduction =
            await fetchProductBlueprintIdByProductionIdHTTP(requestId).catch(
              (e) => {
                log(
                  "fetchProductBlueprintIdByProductionIdHTTP failed",
                  e?.message ?? e,
                );
                return null;
              },
            );
          resolvedPB = asNonEmptyString(pbFromProduction);
        }

        if (resolvedPB) {
          setProductBlueprintId(resolvedPB);
          log("productBlueprintId resolved =", resolvedPB);
        } else {
          setProductBlueprintId("");
          log("WARN: productBlueprintId not resolved (inspections/prod both missing?)");
        }
      } catch (e: any) {
        if (!cancelled) {
          setError(e?.message ?? "検査結果の取得に失敗しました");
          log("load failed", e?.message ?? e);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
          log("load end");
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
        const patch = await loadProductBlueprintPatch(productBlueprintId);
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

  // ★追加: pbPatch が「実際に state に入ったか」を監視して要素を全チェック
  React.useEffect(() => {
    const p = pbPatch as any;

    if (!p) {
      log("pbPatch state =", null);
      return;
    }

    const snapshot = {
      productName: p?.productName ?? null,
      brandId: p?.brandId ?? null,
      brandName: p?.brandName ?? null,
      itemType: p?.itemType ?? null,
      fit: p?.fit ?? null,
      material: p?.material ?? null,
      weight: p?.weight ?? null,
      qualityAssurance: Array.isArray(p?.qualityAssurance) ? p.qualityAssurance : null,
      qualityAssuranceCount: Array.isArray(p?.qualityAssurance)
        ? p.qualityAssurance.length
        : 0,
      productIdTag: p?.productIdTag ?? null,
      productIdTagType: p?.productIdTag?.type ?? null,
      assigneeId: p?.assigneeId ?? null,
      keys: Object.keys(p ?? {}),
    };

    log("pbPatch state set -> snapshot =", snapshot);
  }, [pbPatch]);

  // ③ 検査カード用
  const inspectionCardData = useInspectionResultCard({
    batch: inspectionBatch ?? undefined,
  });

  const totalMintQuantity = inspectionCardData.totalPassed;
  const tokenBlueprint = resolveBlueprintForMintRequest(requestId);

  // ④ ProductBlueprintCard 用 VM
  const productBlueprintCardView: ProductBlueprintCardViewModel | null =
    React.useMemo(() => {
      if (!pbPatch) return null;

      return {
        productName: pbPatch.productName ?? undefined,
        brand: pbPatch.brandName ?? pbPatch.brandId ?? undefined,
        itemType: pbPatch.itemType ?? undefined,
        fit: pbPatch.fit ?? undefined,
        materials: pbPatch.material ?? undefined,
        weight: pbPatch.weight ?? undefined,
        washTags: pbPatch.qualityAssurance ?? undefined,
        productIdTag: pbPatch.productIdTag?.type ?? undefined,
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
        const brands = await loadBrandsForMint();
        if (!cancelled) {
          setBrandOptions(
            (brands ?? []).map((b: BrandForMintDTO): BrandOption => ({
              id: b.id,
              name: b.name,
            })),
          );
          log("brandOptions loaded length=", (brands ?? []).length);
        }
      } catch (e) {
        // eslint-disable-next-line no-console
        console.error("[useMintRequestDetail] failed to load brands", e);
      }
    };

    run();
    return () => {
      cancelled = true;
    };
  }, []);

  const handleSelectBrand = React.useCallback(
    async (brandId: string) => {
      setSelectedBrandId(brandId);

      if (!brandId) {
        setSelectedBrandName("");
        setTokenBlueprintOptions([]);
        setSelectedTokenBlueprintId("");
        return;
      }

      const found = brandOptions.find((b) => b.id === brandId);
      setSelectedBrandName(found ? found.name : "");

      try {
        const list = await loadTokenBlueprintsByBrand(brandId);
        const opts: TokenBlueprintOption[] = (list ?? []).map(
          (tb: TokenBlueprintForMintDTO): TokenBlueprintOption => ({
            id: tb.id,
            name: tb.name,
            symbol: tb.symbol,
            iconUrl: tb.iconUrl,
          }),
        );
        setTokenBlueprintOptions(opts);
        setSelectedTokenBlueprintId("");
        log(
          "tokenBlueprintOptions loaded brandId=",
          brandId,
          "length=",
          opts.length,
          "sample[0]=",
          opts[0],
        );
      } catch (e) {
        // eslint-disable-next-line no-console
        console.error(
          "[useMintRequestDetail] failed to load tokenBlueprints by brand",
          e,
        );
        setTokenBlueprintOptions([]);
        setSelectedTokenBlueprintId("");
      }
    },
    [brandOptions],
  );

  // ============================================================
  // ★ mint 情報（mintDTO 優先）
  // ============================================================

  const mint: MintInfo | null = React.useMemo(() => {
    const fromDTO = extractMintInfoFromMintDTO(mintDTO as any);
    if (fromDTO) {
      log("mint resolved from mintDTO =", fromDTO);
      return fromDTO;
    }

    const fromBatch = extractMintInfoFromBatch(inspectionBatch as any);
    log("mint resolved from inspectionBatch(embed) =", fromBatch ?? null);
    return fromBatch;
  }, [mintDTO, inspectionBatch]);

  const hasMint = React.useMemo(() => !!mint, [mint]);
  const isMintRequested = hasMint;

  const requestedBy: string | null = React.useMemo(() => {
    const v = asNonEmptyString(mint?.createdBy);
    return v ? v : null;
  }, [mint]);

  const requestedAt: string | null = React.useMemo(() => {
    const v = asNonEmptyString(mint?.createdAt);
    return v ? v : null;
  }, [mint]);

  const mintRequestedTokenBlueprintId = React.useMemo(() => {
    const v = asNonEmptyString(mint?.tokenBlueprintId);
    return v ? v : "";
  }, [mint]);

  // 申請済みの場合: ブランド自動選択
  React.useEffect(() => {
    if (!hasMint) return;

    const brandId =
      asNonEmptyString(mint?.brandId) ||
      asNonEmptyString((pbPatch as any)?.brandId);

    if (!brandId) return;
    if (selectedBrandId === brandId) return;

    (async () => {
      try {
        await handleSelectBrand(brandId);
      } catch (e) {
        // eslint-disable-next-line no-console
        console.error(
          "[useMintRequestDetail] auto-select brand for requested batch failed",
          e,
        );
      }
    })();
  }, [hasMint, mint, pbPatch, selectedBrandId, handleSelectBrand]);

  // tokenBlueprintId を自動選択
  React.useEffect(() => {
    if (!hasMint) return;
    if (!mintRequestedTokenBlueprintId) return;
    if (selectedTokenBlueprintId) return;

    const exists = tokenBlueprintOptions.some(
      (tb) => tb.id === mintRequestedTokenBlueprintId,
    );
    if (!exists) return;

    setSelectedTokenBlueprintId(mintRequestedTokenBlueprintId);
  }, [
    hasMint,
    mintRequestedTokenBlueprintId,
    selectedTokenBlueprintId,
    tokenBlueprintOptions,
  ]);

  // scheduledBurnDate を同期
  React.useEffect(() => {
    if (!hasMint) return;

    const raw = mint?.scheduledBurnDate;
    if (!raw) return;

    const s = String(raw);
    const asDate = s.length >= 10 ? s.slice(0, 10) : s;
    if (asDate && asDate !== scheduledBurnDate) {
      setScheduledBurnDate(asDate);
    }
  }, [hasMint, mint, scheduledBurnDate]);

  const showMintButton = !isMintRequested;
  const showBrandSelectorCard = !isMintRequested;
  const showTokenSelectorCard = !isMintRequested;

  // ★ ミント申請（未申請時のみ）
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
      log("postMintRequest start", {
        productionId,
        selectedTokenBlueprintId,
        scheduledBurnDate: scheduledBurnDate || null,
      });

      const updated = await postMintRequestHTTP(
        productionId,
        selectedTokenBlueprintId,
        scheduledBurnDate || undefined,
      );

      if (updated) {
        setInspectionBatch(updated as any);
      }

      alert(
        `ミント申請を登録しました（生産ID: ${productionId} / ミント数: ${totalMintQuantity}）`,
      );
    } catch (e: any) {
      // eslint-disable-next-line no-console
      console.error("[useMintRequestDetail] failed to post mint request", e);
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

  return {
    title,
    loading,
    error,
    inspectionCardData,
    tokenBlueprint,
    totalMintQuantity,
    onBack,
    handleMint,

    // ★ detail へ渡したいキー
    productBlueprintId, // ★追加
    hasMint,
    mint,

    // 申請済みフラグ＆表示制御
    isMintRequested,
    showMintButton,
    showBrandSelectorCard,
    showTokenSelectorCard,

    // 申請済み表示用（mints 正）
    requestedBy,
    requestedAt,

    // productBlueprint Patch 系
    productBlueprintCardView,
    pbPatchLoading,
    pbPatchError,

    // ブランド選択カード用
    brandOptions,
    selectedBrandId,
    selectedBrandName,
    handleSelectBrand,

    // トークン設計カード用
    tokenBlueprintOptions,
    selectedTokenBlueprintId,
    handleSelectTokenBlueprint,
    selectedTokenBlueprint,

    // 焼却予定日
    scheduledBurnDate,
    setScheduledBurnDate,
  };
}
