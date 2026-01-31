// frontend/console/production/src/presentation/hook/useProductionDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

import {
  loadProductionDetail,
  loadModelVariationIndexByProductBlueprintId,
  updateProductionDetail,
  type ProductionDetail,
  type ModelVariationSummary,
} from "../../application/detail/index";

import {
  loadProductBlueprintDetail,
  type ProductBlueprintDetail,
} from "../../application/productBlueprint/index";

// ★ 印刷用ロジックを分離した hook を利用（modelId を正にした版）
import { usePrintCard } from "../../../../product/src/presentation/hook/usePrintCard";

// ★ ViewModel（modelId を正）
import type { ProductionQuantityRowVM } from "../viewModels/productionQuantityRowVM";
import { buildProductionQuantityRowVMs } from "../viewModels/buildProductionQuantityRowVMs";
import { normalizeProductionModels } from "../viewModels/normalizeProductionModels";
import { toProductionDetailUpdateRows } from "../viewModels/toProductionDetailUpdateRows";

type Mode = "view" | "edit";

export function useProductionDetail() {
  const navigate = useNavigate();
  const { productionId } = useParams<{ productionId: string }>();

  const { currentMember } = useAuth();
  const creator = currentMember?.fullName ?? "-";

  const [production, setProduction] = React.useState<ProductionDetail | null>(
    null,
  );

  // ======================================================
  // 画面全体のモード（view / edit）
  // ======================================================
  const [mode, setMode] = React.useState<Mode>("view");
  const isViewMode = mode === "view";
  const isEditMode = mode === "edit";

  // ★ printed=true（印刷済）のときは編集不可（ヘッダー編集ボタン非表示に利用）
  const canEdit = production?.printed !== true;

  const switchToView = React.useCallback(() => setMode("view"), []);

  const switchToEdit = React.useCallback(() => {
    if (!canEdit) return;
    setMode("edit");
  }, [canEdit]);

  const toggleMode = React.useCallback(() => {
    if (!canEdit) return;
    setMode((prev) => (prev === "view" ? "edit" : "view"));
  }, [canEdit]);

  // AdminCard 用モード
  const adminMode: "view" | "edit" = mode;

  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const [productBlueprint, setProductBlueprint] =
    React.useState<ProductBlueprintDetail | null>(null);
  const [pbLoading, setPbLoading] = React.useState(false);
  const [pbError, setPbError] = React.useState<string | null>(null);

  const [modelIndex, setModelIndex] = React.useState<
    Record<string, ModelVariationSummary>
  >({});

  // ✅ 画面 state / 返却は VM を正にする（modelId をキー）
  const [quantityRows, setQuantityRows] = React.useState<
    ProductionQuantityRowVM[]
  >([]);

  // ======================================================
  // Production 詳細取得
  // ======================================================
  React.useEffect(() => {
    if (!productionId) return;

    let cancelled = false;

    (async () => {
      try {
        setLoading(true);
        setError(null);

        setProductBlueprint(null);
        setPbError(null);
        setModelIndex({});
        setQuantityRows([]);

        const data = await loadProductionDetail(productionId);
        if (cancelled) return;

        setProduction(data);
      } catch {
        if (!cancelled) {
          setError("生産情報の取得に失敗しました");
          setProduction(null);
          setQuantityRows([]);
          setProductBlueprint(null);
          setModelIndex({});
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [productionId]);

  // ======================================================
  // ProductBlueprint 詳細取得
  // ======================================================
  React.useEffect(() => {
    const productBlueprintId = production?.productBlueprintId;
    if (!productBlueprintId) {
      setProductBlueprint(null);
      setPbError(null);
      return;
    }

    let cancelled = false;
    (async () => {
      try {
        setPbLoading(true);
        setPbError(null);

        const pb = await loadProductBlueprintDetail(productBlueprintId);
        if (cancelled) return;

        setProductBlueprint(pb);
      } catch {
        if (!cancelled) {
          setPbError("商品設計情報の取得に失敗しました");
          setProductBlueprint(null);
        }
      } finally {
        if (!cancelled) setPbLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [production?.productBlueprintId]);

  // ======================================================
  // ModelVariation index 取得
  // ======================================================
  React.useEffect(() => {
    const productBlueprintId = production?.productBlueprintId;
    if (!productBlueprintId) {
      setModelIndex({});
      return;
    }

    let cancelled = false;
    (async () => {
      try {
        const index =
          await loadModelVariationIndexByProductBlueprintId(productBlueprintId);
        if (cancelled) return;

        setModelIndex(index);
      } catch {
        if (!cancelled) setModelIndex({});
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [production?.productBlueprintId]);

  // ======================================================
  // production.models × modelIndex → quantityRows(VM)
  // ======================================================
  React.useEffect(() => {
    const raw = (production as unknown as { models?: unknown })?.models;

    if (!Array.isArray(raw)) {
      setQuantityRows([]);
      return;
    }

    const normalized = normalizeProductionModels(raw);
    const vms = buildProductionQuantityRowVMs(normalized, modelIndex);

    setQuantityRows(vms);
  }, [production, modelIndex]);

  // ======================================================
  // 保存処理（quantity + assigneeId）
  // ======================================================
  const onSave = React.useCallback(async () => {
    if (!productionId || !production) return;

    if (!canEdit) {
      // eslint-disable-next-line no-alert
      alert("この生産は編集できません（印刷済みです）。");
      return;
    }

    try {
      const rowsForUpdate = toProductionDetailUpdateRows(quantityRows);

      const updated = await updateProductionDetail({
        productionId,
        rows: rowsForUpdate,
        assigneeId: production.assigneeId ?? null,
      });

      if (updated) {
        setProduction(updated);
      }

      setMode("view");
    } catch {
      // eslint-disable-next-line no-alert
      alert("更新に失敗しました");
    }
  }, [productionId, production, quantityRows, canEdit]);

  // ======================================================
  // 印刷（usePrintCard は QuantityRowBase: modelId を要求）
  // ======================================================
  const rowsForPrint = React.useMemo(
    () =>
      (Array.isArray(quantityRows) ? quantityRows : []).map((vm, index) => ({
        // ✅ modelId を正として渡す（VM の modelId をそのまま）
        modelId: String(vm.modelId ?? "").trim() || String(index),
        quantity: vm.quantity ?? 0,
      })),
    [quantityRows],
  );

  const { onPrint } = usePrintCard({
    productionId: productionId ?? null,
    hasProduction: !!production,
    rows: rowsForPrint,
  });

  // ======================================================
  // 戻る
  // ======================================================
  const handleBack = React.useCallback(() => {
    navigate("/production");
  }, [navigate]);

  return {
    mode,
    isViewMode,
    isEditMode,
    switchToView,
    switchToEdit,
    toggleMode,

    canEdit,
    adminMode,

    onBack: handleBack,
    onSave,
    onPrint,

    productionId: productionId ?? null,
    production,
    setProduction,
    loading,
    error,

    productBlueprint,
    pbLoading,
    pbError,

    // ✅ VM 正（state / 返却ともに VM）
    quantityRows,
    setQuantityRows,

    creator,
  };
}
