// frontend/console/production/src/presentation/hook/useProductionDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

// ★ currentMember.fullName 取得など（将来の表示用）
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

// ★ Production 詳細取得サービス
import {
  loadProductionDetail,
  type ProductionDetail,
} from "../../application/productionDetailService";

type Mode = "view" | "edit";

/**
 * Production 詳細画面用 Hook
 *
 * - URL パラメータの productionId を保持
 * - edit / view モードの出し分け機能を提供（デフォルトは view）
 * - 戻るボタン用の onBack を提供
 * - loadProductionDetail を使って詳細データを取得
 */
export function useProductionDetail() {
  const navigate = useNavigate();
  const { productionId } = useParams<{ productionId: string }>();

  // ==========================
  // currentMember 情報（表示などに使える）
  // ==========================
  const { currentMember } = useAuth();
  const creator = currentMember?.fullName ?? "-";

  // ==========================
  // モード管理（view / edit）
  // ==========================
  const [mode, setMode] = React.useState<Mode>("view"); // ★ デフォルト view

  const isViewMode = mode === "view";
  const isEditMode = mode === "edit";

  const switchToView = React.useCallback(() => setMode("view"), []);
  const switchToEdit = React.useCallback(() => setMode("edit"), []);
  const toggleMode = React.useCallback(
    () => setMode((prev) => (prev === "view" ? "edit" : "view")),
    [],
  );

  // ==========================
  // Production 詳細データ
  // ==========================
  const [production, setProduction] = React.useState<ProductionDetail | null>(
    null,
  );
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (!productionId) return;

    let cancelled = false;

    (async () => {
      try {
        setLoading(true);
        setError(null);

        const data = await loadProductionDetail(productionId);
        if (cancelled) return;

        setProduction(data);
      } catch (e) {
        console.error("[useProductionDetail] failed to load:", e);
        if (!cancelled) {
          setError("生産情報の取得に失敗しました");
          setProduction(null);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [productionId]);

  // ==========================
  // 戻る
  // ==========================
  const handleBack = React.useCallback(() => {
    navigate("/production");
  }, [navigate]);

  return {
    // モード関連
    mode,
    isViewMode,
    isEditMode,
    switchToView,
    switchToEdit,
    toggleMode,

    // 画面制御
    onBack: handleBack,

    // 詳細データ関連
    productionId: productionId ?? null,
    production,
    setProduction,
    loading,
    error,

    // 参考情報（ヘッダなどで使用可）
    creator,
  };
}
