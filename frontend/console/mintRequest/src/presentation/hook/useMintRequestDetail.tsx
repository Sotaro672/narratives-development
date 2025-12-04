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
  const blueprint = resolveBlueprintForMintRequest(requestId);

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

  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  const handleMint = React.useCallback(() => {
    alert(
      `ミント申請を実行しました（申請ID: ${
        requestId ?? "不明"
      } / ミント数: ${totalMintQuantity}）`,
    );
  }, [requestId, totalMintQuantity]);

  return {
    title,
    loading,
    error,
    inspectionCardData,
    blueprint,
    totalMintQuantity,
    onBack,
    handleMint,

    // productBlueprint Patch 系
    productBlueprintCardView,
    pbPatchLoading,
    pbPatchError,
  };
}
