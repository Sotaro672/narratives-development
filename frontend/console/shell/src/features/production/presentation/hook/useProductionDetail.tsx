// frontend/console/shell/src/features/production/presentation/hook/useProductionDetail.tsx

import * as React from "react";
import {
  useNavigate,
  useParams,
} from "react-router-dom";

import { useAuth } from "../../../../auth/presentation/hook/useCurrentMember";

import {
  loadProductionDetail,
  loadModelVariationIndexByProductBlueprintId,
  updateProductionDetail,
  type ProductionDetail,
  type ModelVariationSummary,
} from "../../application/detail/index";

import { getProductBlueprintDetail } from "../../../productBlueprint/application/productBlueprintDetailService";

import type { ProductionQuantityRowVM } from "../viewModels/productionQuantityRowVM";
import { buildProductionQuantityRowVMs } from "../viewModels/buildProductionQuantityRowVMs";
import { toProductionDetailUpdateRows } from "../viewModels/toProductionDetailUpdateRows";

type Mode = "view" | "edit";

type ProductBlueprintDetailForProduction = Awaited<
  ReturnType<typeof getProductBlueprintDetail>
>;

export function useProductionDetail() {
  const navigate = useNavigate();
  const { productionId } = useParams<{
    productionId: string;
  }>();

  const { currentMember } = useAuth();
  const creator =
    currentMember?.displayName ?? "-";

  const [production, setProduction] =
    React.useState<ProductionDetail | null>(
      null,
    );

  // ======================================================
  // 画面全体のモード（view / edit）
  // ======================================================
  const [mode, setMode] =
    React.useState<Mode>("view");

  const isViewMode = mode === "view";
  const isEditMode = mode === "edit";

  // printed=true（印刷済）のときは編集不可
  const canEdit =
    production?.printed !== true;

  const switchToView = React.useCallback(
    () => {
      setMode("view");
    },
    [],
  );

  const switchToEdit = React.useCallback(
    () => {
      if (!canEdit) {
        return;
      }

      setMode("edit");
    },
    [canEdit],
  );

  // AdminCard 用モード
  const adminMode: "view" | "edit" =
    mode;

  const [loading, setLoading] =
    React.useState(false);

  const [error, setError] =
    React.useState<string | null>(null);

  const [
    productBlueprint,
    setProductBlueprint,
  ] =
    React.useState<ProductBlueprintDetailForProduction | null>(
      null,
    );

  const [pbLoading, setPbLoading] =
    React.useState(false);

  const [pbError, setPbError] =
    React.useState<string | null>(null);

  const [modelIndex, setModelIndex] =
    React.useState<
      Record<string, ModelVariationSummary>
    >({});

  // 画面 state / 返却は VM を正にする
  const [quantityRows, setQuantityRows] =
    React.useState<
      ProductionQuantityRowVM[]
    >([]);

  // ======================================================
  // Production 詳細取得
  // ======================================================
  React.useEffect(() => {
    if (!productionId) {
      return;
    }

    let cancelled = false;

    void (async () => {
      try {
        setLoading(true);
        setError(null);

        setProductBlueprint(null);
        setPbError(null);
        setModelIndex({});
        setQuantityRows([]);

        const data =
          await loadProductionDetail(
            productionId,
          );

        if (cancelled) {
          return;
        }

        setProduction(data);
      } catch {
        if (!cancelled) {
          setError(
            "生産情報の取得に失敗しました",
          );
          setProduction(null);
          setQuantityRows([]);
          setProductBlueprint(null);
          setModelIndex({});
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

  // ======================================================
  // ProductBlueprint 詳細取得
  // ======================================================
  React.useEffect(() => {
    const productBlueprintId =
      production?.productBlueprintId;

    if (!productBlueprintId) {
      setProductBlueprint(null);
      setPbError(null);
      return;
    }

    let cancelled = false;

    void (async () => {
      try {
        setPbLoading(true);
        setPbError(null);

        const pb =
          await getProductBlueprintDetail(
            productBlueprintId,
          );

        if (cancelled) {
          return;
        }

        setProductBlueprint(pb);
      } catch {
        if (!cancelled) {
          setPbError(
            "商品設計情報の取得に失敗しました",
          );
          setProductBlueprint(null);
        }
      } finally {
        if (!cancelled) {
          setPbLoading(false);
        }
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
    const productBlueprintId =
      production?.productBlueprintId;

    if (!productBlueprintId) {
      setModelIndex({});
      return;
    }

    let cancelled = false;

    void (async () => {
      try {
        const index =
          await loadModelVariationIndexByProductBlueprintId(
            productBlueprintId,
          );

        if (cancelled) {
          return;
        }

        setModelIndex(index);
      } catch {
        if (!cancelled) {
          setModelIndex({});
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [production?.productBlueprintId]);

  // ======================================================
  // production.Models × modelIndex → quantityRows
  // backendレスポンスのPascalCaseを正とする
  // ======================================================
  React.useEffect(() => {
    const raw =
      (production as any)?.Models;

    if (!Array.isArray(raw)) {
      setQuantityRows([]);
      return;
    }

    const viewModels =
      buildProductionQuantityRowVMs(
        raw as any[],
        modelIndex,
      );

    setQuantityRows(viewModels);
  }, [production, modelIndex]);

  // ======================================================
  // 保存処理（quantity + assigneeId）
  // ======================================================
  const onSave =
    React.useCallback(async () => {
      if (
        !productionId ||
        !production
      ) {
        return;
      }

      if (!canEdit) {
        alert(
          "この生産は編集できません（印刷済みです）。",
        );
        return;
      }

      try {
        const rowsForUpdate =
          toProductionDetailUpdateRows(
            quantityRows,
          );

        const updated =
          await updateProductionDetail({
            productionId,
            rows: rowsForUpdate,
            assigneeId:
              production.assigneeId ??
              null,
          });

        if (updated) {
          setProduction(updated);
        }

        setMode("view");
      } catch {
        alert("更新に失敗しました");
      }
    }, [
      productionId,
      production,
      quantityRows,
      canEdit,
    ]);

  // ======================================================
  // 戻る
  // ======================================================
  const handleBack =
    React.useCallback(() => {
      navigate("/production");
    }, [navigate]);

  return {
    isViewMode,
    isEditMode,
    switchToView,
    switchToEdit,

    canEdit,
    adminMode,

    onBack: handleBack,
    onSave,

    productionId:
      productionId ?? null,
    production,
    loading,
    error,

    productBlueprint,
    pbLoading,
    pbError,

    quantityRows,
    setQuantityRows,

    creator,
  };
}

export default useProductionDetail;