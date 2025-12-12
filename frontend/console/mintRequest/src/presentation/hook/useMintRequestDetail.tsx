// frontend/console/mintRequest/src/presentation/hook/useMintRequestDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useInspectionResultCard } from "./useInspectionResultCard";

import type { InspectionBatchDTO } from "../../infrastructure/api/mintRequestApi";
import {
  loadInspectionBatchFromMintAPI,
  loadProductBlueprintPatch,
  resolveBlueprintForMintRequest,
  type ProductBlueprintPatchDTO,
  type BrandForMintDTO,
  type TokenBlueprintForMintDTO,
  loadBrandsForMint,
  loadTokenBlueprintsByBrand,
  postMintRequest, // ★ service 経由でミント申請
} from "../../application/mintRequestService";

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

function asNonEmptyString(v: any): string {
  return typeof v === "string" && v.trim() ? v.trim() : "";
}

function asMaybeISO(v: any): string {
  if (!v) return "";
  if (typeof v === "string") return v;
  if (v instanceof Date) return v.toISOString();
  return String(v);
}

// ★ InspectionBatchDTO に mint 情報が “埋め込まれて返ってくる” 前提で、幅広く吸収する
// 期待値: minted モード判定は mints レコードの有無
function extractMintInfoFromBatch(batch: any): MintInfo | null {
  if (!batch) return null;

  // パターンA: batch.mint がある（推奨）
  const mintObj =
    batch.mint ?? batch.Mint ?? batch.mintRequest ?? batch.MintRequest;
  if (mintObj) {
    const id =
      asNonEmptyString(mintObj.id) ||
      asNonEmptyString(mintObj.ID) ||
      asNonEmptyString(mintObj.mintId) ||
      asNonEmptyString(mintObj.MintID);

    const brandId =
      asNonEmptyString(mintObj.brandId) ||
      asNonEmptyString(mintObj.BrandId) ||
      asNonEmptyString(mintObj.BrandID);

    const tokenBlueprintId =
      asNonEmptyString(mintObj.tokenBlueprintId) ||
      asNonEmptyString(mintObj.TokenBlueprintId) ||
      asNonEmptyString(mintObj.TokenBlueprintID);

    const createdBy =
      asNonEmptyString(mintObj.createdBy) || asNonEmptyString(mintObj.CreatedBy);

    const createdAt =
      asNonEmptyString(asMaybeISO(mintObj.createdAt)) ||
      asNonEmptyString(asMaybeISO(mintObj.CreatedAt));

    const minted =
      typeof mintObj.minted === "boolean"
        ? mintObj.minted
        : typeof mintObj.Minted === "boolean"
          ? mintObj.Minted
          : false;

    const mintedAt =
      asNonEmptyString(asMaybeISO(mintObj.mintedAt)) ||
      asNonEmptyString(asMaybeISO(mintObj.MintedAt)) ||
      "";

    const onChainTxSignature =
      asNonEmptyString(mintObj.onChainTxSignature) ||
      asNonEmptyString(mintObj.OnChainTxSignature) ||
      "";

    const scheduledBurnDate =
      asNonEmptyString(asMaybeISO(mintObj.scheduledBurnDate)) ||
      asNonEmptyString(asMaybeISO(mintObj.ScheduledBurnDate)) ||
      "";

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

  // パターンB: batch に mint* がフラットに載る
  const mintId =
    asNonEmptyString(batch.mintId) ||
    asNonEmptyString(batch.MintId) ||
    asNonEmptyString(batch.MintID);

  const brandId =
    asNonEmptyString(batch.mintBrandId) ||
    asNonEmptyString(batch.MintBrandId) ||
    asNonEmptyString(batch.brandId) ||
    asNonEmptyString(batch.BrandId) ||
    asNonEmptyString(batch.BrandID);

  const tokenBlueprintId =
    asNonEmptyString(batch.mintTokenBlueprintId) ||
    asNonEmptyString(batch.MintTokenBlueprintId) ||
    asNonEmptyString(batch.tokenBlueprintId) ||
    asNonEmptyString(batch.TokenBlueprintId) ||
    asNonEmptyString(batch.TokenBlueprintID);

  const createdBy =
    asNonEmptyString(batch.mintCreatedBy) ||
    asNonEmptyString(batch.MintCreatedBy) ||
    asNonEmptyString(batch.createdBy) ||
    asNonEmptyString(batch.CreatedBy);

  const createdAt =
    asNonEmptyString(asMaybeISO(batch.mintCreatedAt)) ||
    asNonEmptyString(asMaybeISO(batch.MintCreatedAt)) ||
    asNonEmptyString(asMaybeISO(batch.createdAt)) ||
    asNonEmptyString(asMaybeISO(batch.CreatedAt));

  const minted =
    typeof batch.minted === "boolean"
      ? batch.minted
      : typeof batch.Minted === "boolean"
        ? batch.Minted
        : false;

  const mintedAt =
    asNonEmptyString(asMaybeISO(batch.mintedAt)) ||
    asNonEmptyString(asMaybeISO(batch.MintedAt)) ||
    "";

  const onChainTxSignature =
    asNonEmptyString(batch.onChainTxSignature) ||
    asNonEmptyString(batch.OnChainTxSignature) ||
    "";

  const scheduledBurnDate =
    asNonEmptyString(asMaybeISO(batch.scheduledBurnDate)) ||
    asNonEmptyString(asMaybeISO(batch.ScheduledBurnDate)) ||
    "";

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

  // パターンC: batch.mints が配列で返る（先頭を採用）
  const mintsArr = batch.mints ?? batch.Mints;
  if (Array.isArray(mintsArr) && mintsArr.length > 0) {
    const first = mintsArr[0];

    const id =
      asNonEmptyString(first.id) ||
      asNonEmptyString(first.ID) ||
      asNonEmptyString(first.mintId) ||
      asNonEmptyString(first.MintID);

    const brandId =
      asNonEmptyString(first.brandId) ||
      asNonEmptyString(first.BrandId) ||
      asNonEmptyString(first.BrandID);

    const tokenBlueprintId =
      asNonEmptyString(first.tokenBlueprintId) ||
      asNonEmptyString(first.TokenBlueprintId) ||
      asNonEmptyString(first.TokenBlueprintID);

    const createdBy =
      asNonEmptyString(first.createdBy) || asNonEmptyString(first.CreatedBy);

    const createdAt =
      asNonEmptyString(asMaybeISO(first.createdAt)) ||
      asNonEmptyString(asMaybeISO(first.CreatedAt));

    const minted =
      typeof first.minted === "boolean"
        ? first.minted
        : typeof first.Minted === "boolean"
          ? first.Minted
          : false;

    const mintedAt =
      asNonEmptyString(asMaybeISO(first.mintedAt)) ||
      asNonEmptyString(asMaybeISO(first.MintedAt)) ||
      "";

    const onChainTxSignature =
      asNonEmptyString(first.onChainTxSignature) ||
      asNonEmptyString(first.OnChainTxSignature) ||
      "";

    const scheduledBurnDate =
      asNonEmptyString(asMaybeISO(first.scheduledBurnDate)) ||
      asNonEmptyString(asMaybeISO(first.ScheduledBurnDate)) ||
      "";

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

  // 画面タイトル
  const title = `ミント申請詳細`;

  // ① 初期化: MintUsecase 経由で Inspection を取得
  React.useEffect(() => {
    if (!requestId) return;

    let cancelled = false;

    const run = async () => {
      setLoading(true);
      setError(null);
      try {
        const batch = await loadInspectionBatchFromMintAPI(requestId);
        if (!cancelled) {
          setInspectionBatch(batch);
        }
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

  // ② inspectionBatch → productBlueprintId を取り出し、Patch を取得
  React.useEffect(() => {
    if (!inspectionBatch) return;

    const pbId = (inspectionBatch as any).productBlueprintId as
      | string
      | undefined;
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
            (brands ?? []).map(
              (b: BrandForMintDTO): BrandOption => ({
                id: b.id,
                name: b.name,
              }),
            ),
          );
        }
      } catch (e) {
        console.error("[useMintRequestDetail] failed to load brands", e);
      }
    };

    run();
    return () => {
      cancelled = true;
    };
  }, []);

  // Popover から、または自動選択からブランドを選択 → TokenBlueprint 一覧を取得
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
      } catch (e) {
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
    return extractMintInfoFromBatch(inspectionBatch as any);
  }, [inspectionBatch]);

  const hasMint = React.useMemo(() => !!mint, [mint]);

  // 「申請済み」＝ mints がある
  const isMintRequested = hasMint;

  // 申請済み表示用（mints 正）
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

  // 申請済みの場合:
  // - mints.brandId を優先してブランド自動選択（無ければ pbPatch.brandId）
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
        console.error(
          "[useMintRequestDetail] auto-select brand for requested batch failed",
          e,
        );
      }
    })();
  }, [hasMint, mint, pbPatch, selectedBrandId, handleSelectBrand]);

  // トークン一覧が取れたら tokenBlueprintId を自動選択
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

  // 申請済みなら scheduledBurnDate を同期（入力欄は非表示だが state の整合性用）
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

    const productionId =
      (inspectionBatch as any).productionId ?? requestId ?? "";

    if (!productionId) {
      alert("productionId が特定できません。");
      return;
    }

    try {
      const updated = await postMintRequest(
        productionId,
        selectedTokenBlueprintId,
        scheduledBurnDate || undefined,
      );

      if (updated) {
        setInspectionBatch(updated);
      }

      alert(
        `ミント申請を登録しました（生産ID: ${productionId} / ミント数: ${totalMintQuantity}）`,
      );
    } catch (e: any) {
      console.error("[useMintRequestDetail] failed to post mint request", e);
      alert(
        `ミント申請に失敗しました: ${
          e?.message ?? "不明なエラーが発生しました"
        }`,
      );
    }
  }, [
    inspectionBatch,
    selectedTokenBlueprintId,
    requestId,
    totalMintQuantity,
    scheduledBurnDate,
  ]);

  // トークン設計カード側からの選択ハンドラ
  const handleSelectTokenBlueprint = React.useCallback(
    (tokenBlueprintId: string) => {
      setSelectedTokenBlueprintId(tokenBlueprintId);
    },
    [],
  );

  // 左カラムの TokenBlueprintCard 用に、選択中の TokenBlueprintOption を解決
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
