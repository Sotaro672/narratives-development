// frontend/console/mintRequest/src/presentation/hook/useMintRequestDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useInspectionResultCard } from "./useInspectionResultCard";

// ★ アプリケーション層サービス（MintUsecase 経由で model 情報も解決される）
import {
  loadInspectionBatchFromMintAPI,
  resolveBlueprintForMintRequest,
} from "../../application/mintRequestService";

import type { InspectionBatchDTO } from "../../infrastructure/api/mintRequestApi";
import type { TokenBlueprint } from "../../../../tokenBlueprint/src/domain/entity/tokenBlueprint";

export function useMintRequestDetail() {
  const navigate = useNavigate();
  const { requestId } = useParams<{ requestId: string }>();

  const [inspectionBatch, setInspectionBatch] =
    React.useState<InspectionBatchDTO | null>(null);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  // 画面タイトル
  const title = `ミント申請詳細`;

  // ================================
  // 初期化: MintUsecase から検査結果取得
  // ================================
  React.useEffect(() => {
    if (!requestId) return;

    let cancelled = false;

    const run = async () => {
      setLoading(true);
      setError(null);
      try {
        // ★ MintUsecase 経由の API → GetModelVariationByID が呼ばれる
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

  // ================================
  // 検査カード用データ構築
  // ================================
  const inspectionCardData = useInspectionResultCard({
    batch: inspectionBatch ?? undefined,
  });

  // 合格数 = ミント数
  const totalMintQuantity = inspectionCardData.totalPassed;

  // ================================
  // TokenBlueprint の解決
  // ================================
  const blueprint: TokenBlueprint | undefined = resolveBlueprintForMintRequest(
    requestId,
  );

  // ================================
  // UI 用イベント
  // ================================
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
  };
}
