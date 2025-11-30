// frontend/console/production/src/presentation/hook/useProductionDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

import {
  loadProductionDetail,
  loadProductBlueprintDetail,
  loadModelVariationIndexByProductBlueprintId,
  buildQuantityRowsFromModels,
  updateProductionDetail,
  type ProductionDetail,
  type ProductBlueprintDetail,
  type ModelVariationSummary,
  type ProductionQuantityRow as DetailQuantityRow,
} from "../../application/productionDetailService";

// create 用行型（modelNumber / color / rgb / quantity を持つ）
import type { ProductionQuantityRow as CreateQuantityRow } from "../../application/productionCreateService";

type Mode = "view" | "edit";

export function useProductionDetail() {
  const navigate = useNavigate();
  const { productionId } = useParams<{ productionId: string }>();

  const { currentMember } = useAuth();
  const creator = currentMember?.fullName ?? "-";

  // ======================================================
  // 画面全体のモード（view / edit）
  // ======================================================
  const [mode, setMode] = React.useState<Mode>("view");
  const isViewMode = mode === "view";
  const isEditMode = mode === "edit";

  const switchToView = React.useCallback(() => setMode("view"), []);
  const switchToEdit = React.useCallback(() => setMode("edit"), []);
  const toggleMode = React.useCallback(
    () => setMode((prev) => (prev === "view" ? "edit" : "view")),
    [],
  );

  // AdminCard 用モード
  const adminMode: "view" | "edit" = mode;

  const [production, setProduction] = React.useState<ProductionDetail | null>(
    null,
  );
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const [productBlueprint, setProductBlueprint] =
    React.useState<ProductBlueprintDetail | null>(null);
  const [pbLoading, setPbLoading] = React.useState(false);
  const [pbError, setPbError] = React.useState<string | null>(null);

  const [modelIndex, setModelIndex] = React.useState<
    Record<string, ModelVariationSummary>
  >({});

  const [quantityRows, setQuantityRows] = React.useState<CreateQuantityRow[]>(
    [],
  );

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
    const blueprintId = production?.productBlueprintId;
    if (!blueprintId) {
      setProductBlueprint(null);
      setPbError(null);
      return;
    }

    let cancelled = false;
    (async () => {
      try {
        setPbLoading(true);
        setPbError(null);

        const pb = await loadProductBlueprintDetail(blueprintId);
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
    const blueprintId = production?.productBlueprintId;
    if (!blueprintId) {
      setModelIndex({});
      return;
    }

    let cancelled = false;
    (async () => {
      try {
        const index =
          await loadModelVariationIndexByProductBlueprintId(blueprintId);
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
  // production.models × modelIndex → quantityRows
  // ======================================================
  React.useEffect(() => {
    if (!production?.models || !Array.isArray((production as any).models)) {
      setQuantityRows([]);
      return;
    }

    const rawModels = (production as any).models as any[];

    const detailRows: DetailQuantityRow[] = buildQuantityRowsFromModels(
      rawModels,
      modelIndex,
    );

    const mapped: CreateQuantityRow[] = detailRows.map((row) => {
      const quantity = row.quantity ?? 0;

      const createRow: CreateQuantityRow & {
        id?: string;
      } = {
        modelVariationId: row.id,
        modelNumber: row.modelNumber,
        size: row.size,
        color: row.color,
        rgb: row.rgb ?? null,
        quantity,
        id: row.id,
      };

      return createRow;
    });

    setQuantityRows(mapped);
  }, [production, modelIndex]);

  // ======================================================
  // 保存処理（ログ削除済）
  // ======================================================
  const onSave = React.useCallback(async () => {
    if (!productionId || !production) return;

    try {
      const rowsForUpdate: DetailQuantityRow[] = quantityRows.map((row) => ({
        id: row.modelVariationId,
        modelNumber: row.modelNumber,
        size: row.size,
        color: row.color,
        rgb: row.rgb ?? null,
        quantity: row.quantity ?? 0,
      }));

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
      alert("更新に失敗しました");
    }
  }, [productionId, production, quantityRows]);

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

    adminMode,

    onBack: handleBack,
    onSave,

    productionId: productionId ?? null,
    production,
    setProduction,
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
