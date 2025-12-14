// frontend/console/mintRequest/src/presentation/hook/useMintRequestDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useInspectionResultCard } from "./useInspectionResultCard";

import type { InspectionBatchDTO, MintDTO } from "../../infrastructure/api/mintRequestApi";

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
  brand?: string; // ✅ 表示用（brandNameのみ）
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

export type TokenBlueprintCardViewModel = {
  id: string;
  name: string;
  symbol: string;

  // ⚠️ brandId は UI 表示に使わせない（揺れ防止のため空文字を渡す）
  brandId: string;

  // ✅ UI 表示は brandName のみに統一
  brandName: string;

  description: string;
  iconUrl?: string;
  isEditMode: boolean;
  brandOptions: { id: string; name: string }[];
};

export type TokenBlueprintCardHandlers = {
  onPreview: () => void;
};

export type MintInfo = {
  id: string;

  // 取得元DTOの形状差（/mint/mints が「list row」返すケース等）に備え、空文字許容
  brandId: string;
  tokenBlueprintId: string;

  createdBy: string;
  createdByName?: string | null; // ★表示はこれを優先
  createdAt: string | null;

  minted: boolean;
  mintedAt?: string | null;
  onChainTxSignature?: string | null;
  scheduledBurnDate?: string | null;
};

// ============================================================
// ✅ model rows（まずは modelId 集計のみ）
// ============================================================

export type MintModelMetaEntry = {
  modelNumber?: string | null;
  size?: string | null;
  colorName?: string | null;
  rgb?: number | null;
};

export type ModelInspectionRow = {
  modelId: string;

  // 現状は未解決（後で /models 側から解決予定）
  modelNumber: string | null;
  size: string | null;
  colorName: string | null;
  rgb: number | null;

  passedCount: number; // 合格数
  totalCount: number; // 生産数（このモデルの対象件数）
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

function safeDateTimeLabelJa(v: string | null | undefined, fallback: string) {
  const s = asNonEmptyString(v);
  if (!s) return fallback;
  const t = Date.parse(s);
  if (Number.isNaN(t)) return s; // 解析不可なら生文字
  return new Date(t).toLocaleString("ja-JP");
}

function safeDateLabelJa(v: string | null | undefined, fallback: string) {
  const s = asNonEmptyString(v);
  if (!s) return fallback;
  const t = Date.parse(s);
  if (Number.isNaN(t)) {
    // "YYYY-MM-DD" などはそのまま出したいケースがあるので生文字返す
    return s;
  }
  return new Date(t).toLocaleDateString("ja-JP");
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

function isPassedResult(v: any): boolean {
  const s = asNonEmptyString(v).toLowerCase();
  return s === "passed";
}

function buildModelRows(batch: InspectionBatchDTO | null): ModelInspectionRow[] {
  const inspections: any[] = Array.isArray((batch as any)?.inspections)
    ? ((batch as any).inspections as any[])
    : [];

  const agg = new Map<string, { modelId: string; passed: number; total: number }>();

  for (const it of inspections) {
    const modelId = asNonEmptyString(it?.modelId ?? it?.ModelID ?? it?.modelID);
    if (!modelId) continue;

    const prev = agg.get(modelId) ?? { modelId, passed: 0, total: 0 };
    prev.total += 1;

    const result =
      it?.inspectionResult ??
      it?.InspectionResult ??
      it?.result ??
      it?.Result ??
      null;

    if (isPassedResult(result)) prev.passed += 1;

    agg.set(modelId, prev);
  }

  const rows: ModelInspectionRow[] = Array.from(agg.values()).map((g) => ({
    modelId: g.modelId,
    modelNumber: null,
    size: null,
    colorName: null,
    rgb: null,
    passedCount: g.passed,
    totalCount: g.total,
  }));

  rows.sort((a, b) => a.modelId.localeCompare(b.modelId));
  return rows;
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
  return (patch ?? null) as any;
}

async function loadBrandsForMint(): Promise<BrandForMintDTO[]> {
  const brands = await fetchBrandsForMintHTTP();
  return (brands ?? []) as any;
}

async function loadTokenBlueprintsByBrand(
  brandId: string,
): Promise<TokenBlueprintForMintDTO[]> {
  const id = String(brandId ?? "").trim();
  if (!id) return [];

  const list = await fetchTokenBlueprintsByBrandHTTP(id);
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

/**
 * MintDTO は環境/エンドポイントによって shape が揺れる可能性があるため、
 * ここでは「持っている情報を最大限拾って MintInfo に詰める」方針にする。
 *
 * - full mint: {id, brandId, tokenBlueprintId, createdBy, createdAt, ...}
 * - list row: {mintId, tokenBlueprint, createdByName, mintedAt, ...} など
 */
function extractMintInfoFromMintDTO(m: any): MintInfo | null {
  if (!m) return null;

  // id 系（MintEntity / ListRow 両対応）
  const id = asNonEmptyString(m.id ?? m.ID ?? m.mintId ?? m.MintID ?? m.mintID);

  // tokenBlueprintId 系（MintEntity / ListRow 両対応）
  const tokenBlueprintId = asNonEmptyString(
    m.tokenBlueprintId ??
      m.TokenBlueprintID ??
      m.TokenBlueprintId ??
      m.tokenBlueprint ??
      m.TokenBlueprint ??
      "",
  );

  const brandId = asNonEmptyString(m.brandId ?? m.BrandID ?? m.BrandId ?? "");

  // createdBy / createdByName（createdByName が来れば UI はそれを優先表示）
  const createdBy = asNonEmptyString(m.createdBy ?? m.CreatedBy ?? "");
  const createdByName = asNonEmptyString(
    m.createdByName ?? m.CreatedByName ?? m.created_by_name ?? "",
  );

  // createdAt（list row では無い場合がある）
  const createdAtStr = asNonEmptyString(asMaybeISO(m.createdAt ?? m.CreatedAt));
  const createdAt = createdAtStr ? createdAtStr : null;

  // minted 系
  const mintedAtStr = asNonEmptyString(asMaybeISO(m.mintedAt ?? m.MintedAt));
  const minted =
    typeof m.minted === "boolean"
      ? m.minted
      : Boolean(mintedAtStr); // mintedAt があれば minted 扱い

  const onChainTxSignature = asNonEmptyString(
    m.onChainTxSignature ?? m.OnChainTxSignature,
  );

  const scheduledBurnDate = asNonEmptyString(
    asMaybeISO(m.scheduledBurnDate ?? m.ScheduledBurnDate),
  );

  // id が無いなら何もできない
  if (!id) return null;

  return {
    id,
    brandId,
    tokenBlueprintId,
    createdBy,
    createdByName: createdByName ? createdByName : null,
    createdAt,
    minted,
    mintedAt: mintedAtStr ? mintedAtStr : null,
    onChainTxSignature: onChainTxSignature ? onChainTxSignature : null,
    scheduledBurnDate: scheduledBurnDate ? scheduledBurnDate : null,
  };
}

// “inspectionBatch に mint が埋め込まれて返る”可能性も一応吸収
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
  const [productBlueprintId, setProductBlueprintId] = React.useState<string>("");

  // ✅ modelId 集計（メタデータは後で解決）
  const [modelMetaMap] = React.useState<Record<string, MintModelMetaEntry>>({});
  const [modelRows, setModelRows] = React.useState<ModelInspectionRow[]>([]);

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

  const title = `ミント申請詳細`;

  // ✅ selectedBrandName は state に持たず、options から常に導出（brandId表示の揺れを防止）
  const selectedBrandName = React.useMemo(() => {
    if (!selectedBrandId) return "";
    return brandOptions.find((b) => b.id === selectedBrandId)?.name ?? "";
  }, [brandOptions, selectedBrandId]);

  // ① 初期化: inspection + mintDTO + productBlueprintId を解決（※新ルートは使わない）
  React.useEffect(() => {
    if (!requestId) return;

    let cancelled = false;

    const run = async () => {
      setLoading(true);
      setError(null);

      log("load start requestId=", requestId);

      try {
        // inspection は 1件取得（旧ルート）
        const [batch, mint] = await Promise.all([
          fetchInspectionByProductionIdHTTP(requestId),
          fetchMintByInspectionIdHTTP(requestId).catch(() => null),
        ]);

        if (cancelled) return;

        setInspectionBatch(batch ?? null);
        setMintDTO(mint ?? null);

        // modelRows（まずは modelId 集計だけ）
        setModelRows(buildModelRows(batch ?? null));

        log("loaded inspection/mint", {
          hasInspection: !!batch,
          hasMint: !!mint,
          sampleInspection: (batch as any)?.inspections?.[0] ?? null,
          modelRowsLen: buildModelRows(batch ?? null).length,
          // ★createdByName が来ているかの確認用
          mintCreatedByName:
            asNonEmptyString((mint as any)?.createdByName ?? (mint as any)?.CreatedByName) ||
            null,
        });

        // ★ productBlueprintId: batchから→無ければ /productions で解決
        const pbFromBatch = extractProductBlueprintIdFromBatch(batch as any);

        let resolvedPB = pbFromBatch;
        if (!resolvedPB) {
          const pbFromProduction =
            await fetchProductBlueprintIdByProductionIdHTTP(requestId).catch(
              () => null,
            );
          resolvedPB = asNonEmptyString(pbFromProduction);
        }

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

  // ③ 検査カード用
  const inspectionCardData = useInspectionResultCard({
    batch: inspectionBatch ?? undefined,
  });

  const totalMintQuantity = inspectionCardData.totalPassed;
  const tokenBlueprint = resolveBlueprintForMintRequest(requestId);

  // ④ ProductBlueprintCard 用 VM（✅ brandName のみ渡す）
  const productBlueprintCardView: ProductBlueprintCardViewModel | null =
    React.useMemo(() => {
      if (!pbPatch) return null;

      return {
        productName: pbPatch.productName ?? undefined,
        brand: pbPatch.brandName ?? undefined, // ✅ brandId を表示に混ぜない
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
      return;
    }

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
    } catch {
      setTokenBlueprintOptions([]);
      setSelectedTokenBlueprintId("");
    }
  }, []);

  // ============================================================
  // ★ mint 情報（mintDTO 優先）
  // ============================================================

  const mint: MintInfo | null = React.useMemo(() => {
    const fromDTO = extractMintInfoFromMintDTO(mintDTO as any);
    if (fromDTO) return fromDTO;

    const fromBatch = extractMintInfoFromBatch(inspectionBatch as any);
    return fromBatch;
  }, [mintDTO, inspectionBatch]);

  const hasMint = React.useMemo(() => !!mint, [mint]);
  const isMintRequested = hasMint;

  const requestedBy: string | null = React.useMemo(() => {
    const v = asNonEmptyString(mint?.createdBy);
    return v ? v : null;
  }, [mint]);

  const requestedAt: string | null = React.useMemo(() => {
    const v = asNonEmptyString(mint?.createdAt ?? null);
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
      } catch {
        // noop
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
      const updated = await postMintRequestHTTP(
        productionId,
        selectedTokenBlueprintId,
        scheduledBurnDate || undefined,
      );

      if (updated) {
        setInspectionBatch(updated as any);
        setModelRows(buildModelRows(updated as any));
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

  // ============================================================
  // ★ view-model / labels（page から移譲）
  // ============================================================

  const tokenBlueprintCardVm: TokenBlueprintCardViewModel | null =
    React.useMemo(() => {
      if (!selectedTokenBlueprint) return null;

      // ✅ brandName のみ UI に渡す（brandId は空文字）
      const brandName = selectedBrandName || "";

      return {
        id: selectedTokenBlueprint.id,
        name: selectedTokenBlueprint.name,
        symbol: selectedTokenBlueprint.symbol,

        // ⚠️ ここを selectedBrandId にしない（brandId 表示の揺れ対策）
        brandId: "",

        // ✅ 表示は brandName のみ
        brandName,

        description: "", // description は取得していないので空
        iconUrl: selectedTokenBlueprint.iconUrl,
        isEditMode: false,
        brandOptions: brandOptions.map((b) => ({ id: b.id, name: b.name })),
      };
    }, [selectedTokenBlueprint, selectedBrandName, brandOptions]);

  const tokenBlueprintCardHandlers: TokenBlueprintCardHandlers =
    React.useMemo(() => ({ onPreview: () => {} }), []);

  // mints テーブル由来の表示用ラベル（page のロジックを移譲）
  const mintCreatedAtLabel = React.useMemo(
    () => safeDateTimeLabelJa(mint?.createdAt ?? null, "（未登録）"),
    [mint?.createdAt],
  );

  // ✅ createdByName → mintCreatedByLabel へ確実に流す
  // - mintDTO が「list row DTO」でも createdByName を拾うので、ここで表示に乗る
  const mintCreatedByLabel = React.useMemo(() => {
    const name = asNonEmptyString(mint?.createdByName);
    if (name) return name;

    const fallback = asNonEmptyString(mint?.createdBy);
    return fallback ? fallback : "（不明）";
  }, [mint?.createdByName, mint?.createdBy]);

  const mintScheduledBurnDateLabel = React.useMemo(
    () => safeDateLabelJa(mint?.scheduledBurnDate ?? null, "（未設定）"),
    [mint?.scheduledBurnDate],
  );

  const mintMintedAtLabel = React.useMemo(
    () => safeDateTimeLabelJa(mint?.mintedAt ?? null, "（未完了）"),
    [mint?.mintedAt],
  );

  const mintedLabel = React.useMemo(() => {
    if (typeof mint?.minted === "boolean") {
      return mint.minted ? "minted" : "notYet";
    }
    return "（不明）";
  }, [mint?.minted]);

  const onChainTxSignature = React.useMemo(
    () => asNonEmptyString(mint?.onChainTxSignature),
    [mint?.onChainTxSignature],
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
    productBlueprintId,
    hasMint,
    mint,

    // ✅ ここは「戻した状態」でも返せる（今は modelId 集計だけ）
    modelMetaMap, // 現状 {}
    modelRows, // modelId / passedCount / totalCount

    // 申請済みフラグ＆表示制御
    isMintRequested,
    showMintButton,
    showBrandSelectorCard,
    showTokenSelectorCard,

    // 申請済み表示用（必要ならUIで使用）
    requestedBy,
    requestedAt,

    // productBlueprint Patch 系
    productBlueprintCardView,
    pbPatchLoading,
    pbPatchError,

    // ブランド選択カード用
    brandOptions,
    selectedBrandId,
    selectedBrandName, // ✅ derived（表示は name のみ）
    handleSelectBrand,

    // トークン設計カード用
    tokenBlueprintOptions,
    selectedTokenBlueprintId,
    handleSelectTokenBlueprint,

    // 選択中 TokenBlueprintOption（必要ならUIで使用）
    selectedTokenBlueprint,

    // ★ page から移譲した VM / handlers
    tokenBlueprintCardVm,
    tokenBlueprintCardHandlers,

    // ★ page から移譲した labels
    mintCreatedAtLabel,
    mintCreatedByLabel, // ✅ createdByName がここに出る
    mintScheduledBurnDateLabel,
    mintMintedAtLabel,
    mintedLabel,
    onChainTxSignature,

    // 焼却予定日
    scheduledBurnDate,
    setScheduledBurnDate,
  };
}
