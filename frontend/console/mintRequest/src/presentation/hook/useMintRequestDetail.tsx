// frontend/console/mintRequest/src/presentation/hook/useMintRequestDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useInspectionResultCard } from "./useInspectionResultCard";

import type { InspectionBatchDTO } from "../../infrastructure/api/mintRequestApi";

// ✅ 以前は application/mintRequestService から import していたが、
// いまは export が揃っていないため Repository を直接呼ぶ（最短でビルドを通す）。
import {
  fetchInspectionBatchesHTTP,
  fetchProductBlueprintPatchHTTP,
  fetchBrandsForMintHTTP,
  fetchTokenBlueprintsByBrandHTTP,
  postMintRequestHTTP,
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
  brand?: string; // brandName 優先（なければ brandId をフォールバック表示）
  itemType?: string;
  fit?: string;
  materials?: string;
  weight?: number;
  washTags?: string[];
  productIdTag?: string;
};

// 右カラムのブランド選択カード用 VM
export type BrandOption = {
  id: string;
  name: string;
};

// トークン設計カード用（name / symbol / icon まで持つ）
export type TokenBlueprintOption = {
  id: string;
  name: string;
  symbol: string;
  iconUrl?: string;
};

// ★ mints テーブル由来の「表示用 Mint 情報」
// ※ 「mints テーブルが正」なので、createdBy/createdAt 等は mints 由来を優先する
export type MintInfo = {
  id: string;
  brandId: string;
  tokenBlueprintId: string;
  createdBy: string;
  createdAt: string; // ISO string 想定
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

/**
 * requestId (= productionId) をキーに /mint/inspections で 1 件取得する。
 * ※ repository には単発 API がないため、一覧を引いて絞り込む（必要なら単発 API を後で追加）
 */
async function loadInspectionBatchFromMintAPI(
  productionId: string,
): Promise<InspectionBatchDTO | null> {
  const pid = String(productionId ?? "").trim();
  if (!pid) return null;

  const batches = await fetchInspectionBatchesHTTP();
  const hit =
    (batches ?? []).find((b: any) => String(b?.productionId ?? "").trim() === pid) ??
    null;

  log("loadInspectionBatchFromMintAPI pid=", pid, "hit=", hit ?? null);
  return hit as any;
}

async function loadProductBlueprintPatch(
  productBlueprintId: string,
): Promise<ProductBlueprintPatchDTO | null> {
  const id = String(productBlueprintId ?? "").trim();
  if (!id) return null;

  const patch = await fetchProductBlueprintPatchHTTP(id);
  log("loadProductBlueprintPatch id=", id, "patch=", patch ?? null);
  return (patch ?? null) as any;
}

async function loadBrandsForMint(): Promise<BrandForMintDTO[]> {
  const brands = await fetchBrandsForMintHTTP();
  log("loadBrandsForMint length=", (brands ?? []).length, "sample[0]=", (brands ?? [])[0]);
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
 * 必要になれば backend の tokenBlueprint API を呼ぶ方式に置き換える。
 */
function resolveBlueprintForMintRequest(_requestId?: string) {
  return undefined;
}

// ★ InspectionBatchDTO に mint 情報が “埋め込まれて返ってくる” 前提で、幅広く吸収する
function extractMintInfoFromBatch(batch: any): MintInfo | null {
  if (!batch) return null;

  const mintObj =
    batch.mint ?? batch.Mint ?? batch.mintRequest ?? batch.MintRequest ?? null;

  const pick = (o: any, keys: string[]) => {
    for (const k of keys) {
      const v = o?.[k];
      const s = asNonEmptyString(v);
      if (s) return s;
    }
    return "";
  };

  const pickBool = (o: any, keys: string[]) => {
    for (const k of keys) {
      if (typeof o?.[k] === "boolean") return o[k] as boolean;
    }
    return false;
  };

  const pickMaybeStr = (o: any, keys: string[]) => {
    for (const k of keys) {
      const v = o?.[k];
      const s = asNonEmptyString(asMaybeISO(v));
      if (s) return s;
    }
    return "";
  };

  // A) batch.mint がある
  if (mintObj) {
    const id = pick(mintObj, ["id", "ID", "mintId", "MintID"]);
    const brandId = pick(mintObj, ["brandId", "BrandId", "BrandID"]);
    const tokenBlueprintId = pick(mintObj, [
      "tokenBlueprintId",
      "TokenBlueprintId",
      "TokenBlueprintID",
    ]);
    const createdBy = pick(mintObj, ["createdBy", "CreatedBy"]);
    const createdAt = pickMaybeStr(mintObj, ["createdAt", "CreatedAt"]);
    const minted = pickBool(mintObj, ["minted", "Minted"]);
    const mintedAt = pickMaybeStr(mintObj, ["mintedAt", "MintedAt"]);
    const onChainTxSignature = pick(mintObj, [
      "onChainTxSignature",
      "OnChainTxSignature",
    ]);
    const scheduledBurnDate = pickMaybeStr(mintObj, [
      "scheduledBurnDate",
      "ScheduledBurnDate",
    ]);

    if (id && brandId && tokenBlueprintId && createdBy && createdAt) {
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
  }

  // B) batch にフラットに載る
  const mintId = pick(batch, ["mintId", "MintId", "MintID"]);
  const brandId = pick(batch, ["mintBrandId", "BrandId", "BrandID", "brandId"]);
  const tokenBlueprintId = pick(batch, [
    "mintTokenBlueprintId",
    "tokenBlueprintId",
    "TokenBlueprintId",
    "TokenBlueprintID",
  ]);
  const createdBy = pick(batch, ["mintCreatedBy", "createdBy", "CreatedBy"]);
  const createdAt = pickMaybeStr(batch, ["mintCreatedAt", "createdAt", "CreatedAt"]);
  const minted = pickBool(batch, ["minted", "Minted"]);
  const mintedAt = pickMaybeStr(batch, ["mintedAt", "MintedAt"]);
  const onChainTxSignature = pick(batch, ["onChainTxSignature", "OnChainTxSignature"]);
  const scheduledBurnDate = pickMaybeStr(batch, ["scheduledBurnDate", "ScheduledBurnDate"]);

  if (mintId && brandId && tokenBlueprintId && createdBy && createdAt) {
    return {
      id: mintId,
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

  return null;
}

export function useMintRequestDetail() {
  const navigate = useNavigate();
  const { requestId } = useParams<{ requestId: string }>();

  const [inspectionBatch, setInspectionBatch] =
    React.useState<InspectionBatchDTO | null>(null);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  // productBlueprint Patch 用
  const [pbPatch, setPbPatch] =
    React.useState<ProductBlueprintPatchDTO | null>(null);
  const [pbPatchLoading, setPbPatchLoading] = React.useState(false);
  const [pbPatchError, setPbPatchError] = React.useState<string | null>(null);

  // 右カラム: ブランド選択カード用（デフォルトは未選択）
  const [brandOptions, setBrandOptions] = React.useState<BrandOption[]>([]);
  const [selectedBrandId, setSelectedBrandId] = React.useState<string>("");
  const [selectedBrandName, setSelectedBrandName] = React.useState<string>("");

  // 右カラム: ブランドに紐づく TokenBlueprint 一覧と選択中 ID
  const [tokenBlueprintOptions, setTokenBlueprintOptions] = React.useState<
    TokenBlueprintOption[]
  >([]);
  const [selectedTokenBlueprintId, setSelectedTokenBlueprintId] =
    React.useState<string>("");

  // ★ 焼却予定日（ScheduledBurnDate）
  const [scheduledBurnDate, setScheduledBurnDate] = React.useState<string>("");

  const title = `ミント申請詳細`;

  // ① 初期化: MintUsecase 経由で Inspection を取得
  React.useEffect(() => {
    if (!requestId) return;

    let cancelled = false;

    const run = async () => {
      setLoading(true);
      setError(null);
      try {
        log("load start requestId=", requestId);
        const batch = await loadInspectionBatchFromMintAPI(requestId);
        if (!cancelled) {
          setInspectionBatch(batch);
          log("inspectionBatch set", batch ?? null);
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

  // ② inspectionBatch → productBlueprintId を取り出し、Patch を取得
  React.useEffect(() => {
    if (!inspectionBatch) return;

    const pbId = (inspectionBatch as any).productBlueprintId as string | undefined;
    log("inspect batch -> productBlueprintId", pbId);

    if (!pbId) return;

    let cancelled = false;

    const run = async () => {
      setPbPatchLoading(true);
      setPbPatchError(null);
      try {
        const patch = await loadProductBlueprintPatch(pbId);
        if (!cancelled) {
          setPbPatch(patch);
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
  }, [inspectionBatch]);

  // ③ 検査カード用
  const inspectionCardData = useInspectionResultCard({
    batch: inspectionBatch ?? undefined,
  });

  // 合格数 = ミント数
  const totalMintQuantity = inspectionCardData.totalPassed;

  // TokenBlueprint（現状は undefined を返すダミー実装）
  const tokenBlueprint = resolveBlueprintForMintRequest(requestId);

  // ④ ProductBlueprintCard 用の ViewModel へ整形
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

  // ★ 戻る
  const MINT_REQUEST_MANAGEMENT_PATH = "/mintRequest";
  const onBack = React.useCallback(() => {
    navigate(MINT_REQUEST_MANAGEMENT_PATH);
  }, [navigate]);

  // ⑤ ブランド一覧のロード
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
  // ★ mints テーブル由来の判定と公開値
  // ============================================================

  const mint: MintInfo | null = React.useMemo(() => {
    const m = extractMintInfoFromBatch(inspectionBatch as any);
    log("extractMintInfoFromBatch =", m ?? null);
    return m;
  }, [inspectionBatch]);

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
      asNonEmptyString(mint?.brandId) || asNonEmptyString((pbPatch as any)?.brandId);

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

  // 申請済み表示制御
  const showMintButton = !isMintRequested;
  const showBrandSelectorCard = !isMintRequested;
  const showTokenSelectorCard = !isMintRequested;

  // ★ ミント申請処理（未申請時のみ呼ばれる想定）
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

    // ★ ページ側で使うために追加
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

    // 焼却予定日（ScheduledBurnDate）
    scheduledBurnDate,
    setScheduledBurnDate,
  };
}
