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
  const [selectedBrandId, setSelectedBrandId] = React.useState<string>(""); // "" = 未選択 / すべて
  const [selectedBrandName, setSelectedBrandName] = React.useState<string>("");

  // 右カラム: ブランドに紐づく TokenBlueprint 一覧と選択中 ID
  const [tokenBlueprintOptions, setTokenBlueprintOptions] = React.useState<
    TokenBlueprintOption[]
  >([]);
  const [selectedTokenBlueprintId, setSelectedTokenBlueprintId] =
    React.useState<string>("");

  // ★ 焼却予定日（ScheduledBurnDate）: HTML date input から "YYYY-MM-DD" 形式で入る想定
  const [scheduledBurnDate, setScheduledBurnDate] = React.useState<string>("");

  // 画面タイトル
  const title = `ミント申請詳細`;

  // ① 初期化: MintUsecase 経由で Inspection + MintInspectionView を取得
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
  //    ※ 右カラムのブランド選択の初期値には反映せず、
  //       あくまで左カラムの Product 情報表示専用とする。
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
        // brandName があればそれを表示、なければフォールバックとして brandId を表示
        brand: pbPatch.brandName ?? pbPatch.brandId ?? undefined,
        itemType: pbPatch.itemType ?? undefined,
        fit: pbPatch.fit ?? undefined,
        materials: pbPatch.material ?? undefined,
        weight: pbPatch.weight ?? undefined,
        washTags: pbPatch.qualityAssurance ?? undefined,
        productIdTag: pbPatch.productIdTag?.type ?? undefined,
      };
    }, [pbPatch]);

  // ★ 戻る: -1 ではなく MintRequestManagement へ絶対パス遷移
  // ※ shell 側のマウントが "mintRequest/*" 想定なら "/mintRequest" が管理画面のルートになります
  const MINT_REQUEST_MANAGEMENT_PATH = "/mintRequest";
  const onBack = React.useCallback(() => {
    navigate(MINT_REQUEST_MANAGEMENT_PATH);
  }, [navigate]);

  // ⑤ ブランド一覧のロード（右カラム / 自動選択どちらでも使う）
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

      // ブランド名の表示更新
      if (!brandId) {
        // 「未選択」扱い
        setSelectedBrandName("");
        setTokenBlueprintOptions([]);
        setSelectedTokenBlueprintId("");
        return;
      }

      const found = brandOptions.find((b) => b.id === brandId);
      if (found) {
        setSelectedBrandName(found.name);
      } else {
        setSelectedBrandName("");
      }

      // 選択したブランドに紐づく TokenBlueprint 一覧を取得
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

        // ブランド変更時は選択中トークン設計をリセット
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
  // 申請済みモード判定
  // ============================================================
  const isMintRequested = React.useMemo(() => {
    const batch = inspectionBatch as any;
    if (!batch) return false;

    const requestedBy = batch.requestedBy ?? batch.RequestedBy;
    const requestedAt = batch.requestedAt ?? batch.RequestedAt;
    const tokenBlueprintId =
      batch.tokenBlueprintId ?? batch.TokenBlueprintId ?? batch.TokenBlueprintID;

    return !!requestedBy && !!requestedAt && !!tokenBlueprintId;
  }, [inspectionBatch]);

  const mintRequestedTokenBlueprintId = React.useMemo(() => {
    const batch = inspectionBatch as any;
    if (!batch) return "";
    const tokenBlueprintId =
      batch.tokenBlueprintId ?? batch.TokenBlueprintId ?? batch.TokenBlueprintID;
    return typeof tokenBlueprintId === "string" ? tokenBlueprintId : "";
  }, [inspectionBatch]);

  // 申請済みの場合に表示用の requestedBy / requestedAt を抽出
  const requestedBy: string | null = React.useMemo(() => {
    const batch = inspectionBatch as any;
    if (!batch) return null;
    const raw = batch.requestedBy ?? batch.RequestedBy;
    return typeof raw === "string" && raw.trim() ? raw.trim() : null;
  }, [inspectionBatch]);

  const requestedAt: string | null = React.useMemo(() => {
    const batch = inspectionBatch as any;
    if (!batch) return null;
    const raw = batch.requestedAt ?? batch.RequestedAt;
    if (!raw) return null;
    // string or Date 想定。表示側でフォーマットするのでここでは string 化だけしておく
    if (typeof raw === "string") return raw;
    if (raw instanceof Date) return raw.toISOString();
    return String(raw);
  }, [inspectionBatch]);

  // 申請済みの場合の表示制御
  const showMintButton = !isMintRequested;
  const showBrandSelectorCard = !isMintRequested;
  const showTokenSelectorCard = !isMintRequested;

  // 申請済みの場合:
  // - pbPatch.brandId を元にブランドを自動選択（ブランド一覧取得済み前提）
  // - tokenBlueprintId に一致するトークンを自動選択
  React.useEffect(() => {
    if (!isMintRequested) return;
    if (!pbPatch) return;

    const brandId = (pbPatch.brandId ?? "") as string;
    if (!brandId) return;

    // すでに同じブランドが選択されている場合は何もしない
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
  }, [isMintRequested, pbPatch, selectedBrandId, handleSelectBrand]);

  // トークン一覧が取れたら tokenBlueprintId を自動選択
  React.useEffect(() => {
    if (!isMintRequested) return;
    if (!mintRequestedTokenBlueprintId) return;

    // すでに選択済みならスキップ
    if (selectedTokenBlueprintId) return;

    const exists = tokenBlueprintOptions.some(
      (tb) => tb.id === mintRequestedTokenBlueprintId,
    );
    if (!exists) return;

    setSelectedTokenBlueprintId(mintRequestedTokenBlueprintId);
  }, [
    isMintRequested,
    mintRequestedTokenBlueprintId,
    selectedTokenBlueprintId,
    tokenBlueprintOptions,
  ]);

  // ★ ミント申請処理（未申請時のみ呼ばれる想定）
  //   ScheduledBurnDate も service(postMintRequest) に渡す
  const handleMint = React.useCallback(async () => {
    if (!inspectionBatch) {
      alert("検査バッチ情報が取得できていません。");
      return;
    }

    if (!selectedTokenBlueprintId) {
      alert("トークン設計を選択してください。");
      return;
    }

    // MintUsecase 側の UpdateRequestInfo は productionId をキーにしているので、
    // InspectionBatchDTO 側の productionId を優先的に利用する。
    const productionId =
      (inspectionBatch as any).productionId ?? requestId ?? "";

    if (!productionId) {
      alert("productionId が特定できません。");
      return;
    }

    // ★ ミント申請ボタン押下時点でのデータをログ出力する
    const batchAny = inspectionBatch as any;
    const inspectionId =
      batchAny.id ??
      batchAny.ID ??
      batchAny.inspectionId ??
      batchAny.InspectionId;

    const passedProducts =
      Array.isArray(batchAny.inspections) || Array.isArray(batchAny.Inspections)
        ? ((batchAny.inspections ?? batchAny.Inspections) as any[])
            .filter((it) => {
              const result = it.result ?? it.Result ?? it.inspectionResult;
              return result === "passed" || result === "ok" || result === "pass";
            })
            .map((it) => it.productId ?? it.ProductId ?? it.productID)
        : [];

    // eslint-disable-next-line no-console
    console.log("[useMintRequestDetail] handleMint payload", {
      productionId,
      tokenBlueprintId: selectedTokenBlueprintId,
      scheduledBurnDate: scheduledBurnDate || null,
      inspectionId,
      rawInspectionBatch: inspectionBatch,
      passedProducts,
    });

    try {
      const updated = await postMintRequest(
        productionId,
        selectedTokenBlueprintId,
        // HTML date input の "YYYY-MM-DD" をそのまま渡す（API 側でパース想定）
        scheduledBurnDate || undefined,
      );

      // eslint-disable-next-line no-console
      console.log("[useMintRequestDetail] handleMint backend response", updated);

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

    // 申請済みフラグ＆表示制御
    isMintRequested,
    showMintButton,
    showBrandSelectorCard,
    showTokenSelectorCard,

    // 申請済み表示用フィールド
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

    // ★ 焼却予定日（ScheduledBurnDate）を画面側へ公開
    scheduledBurnDate,
    setScheduledBurnDate,
  };
}
