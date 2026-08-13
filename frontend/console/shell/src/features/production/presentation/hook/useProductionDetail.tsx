// frontend/console/shell/src/features/production/presentation/hook/useProductionDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import {
  loadProductionDetail,
  updateProductionDetail,
  type ProductionDetail,
} from "../../application/productionDetailService";
import type { ProductionQuantityRow } from "../../application/productionQuantityRow";
import { ProductionRepositoryHTTP } from "../../infrastructure/http/productionRepositoryHTTP";

type Mode = "view" | "edit";

export function useProductionDetail() {
  const navigate = useNavigate();
  const { productionId } = useParams<{ productionId: string }>();

  const [production, setProduction] = React.useState<ProductionDetail | null>(null);
  const [mode, setMode] = React.useState<Mode>("view");
  const [loading, setLoading] = React.useState(false);
  const [deleting, setDeleting] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [quantityRows, setQuantityRows] = React.useState<ProductionQuantityRow[]>([]);

  const isViewMode = mode === "view";
  const isEditMode = mode === "edit";
  const canEdit = production?.printed !== true;

  const switchToView = React.useCallback(() => {
    setMode("view");
  }, []);

  const switchToEdit = React.useCallback(() => {
    if (!canEdit) return;
    setMode("edit");
  }, [canEdit]);

  React.useEffect(() => {
    if (!productionId) return;

    const targetProductionId = productionId;
    let cancelled = false;

    async function loadDetail(id: string) {
      try {
        setLoading(true);
        setError(null);
        setQuantityRows([]);

        const data = await loadProductionDetail(id);
        if (cancelled) return;

        setProduction(data);
        setQuantityRows(data?.models ?? []);
      } catch {
        if (cancelled) return;

        setError("生産情報の取得に失敗しました");
        setProduction(null);
        setQuantityRows([]);
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    void loadDetail(targetProductionId);

    return () => {
      cancelled = true;
    };
  }, [productionId]);

  const onSave = React.useCallback(async () => {
    if (!productionId || !production) return;

    if (!canEdit) {
      alert("この生産は編集できません（印刷済みです）。");
      return;
    }

    try {
      const updated = await updateProductionDetail({
        productionId,
        assigneeId: production.assigneeId,
        rows: quantityRows.map(({ modelId, quantity }) => ({
          modelId,
          quantity,
        })),
      });

      if (updated) {
        setProduction(updated);
        setQuantityRows(updated.models);
      }

      setMode("view");
    } catch {
      alert("更新に失敗しました");
    }
  }, [productionId, production, quantityRows, canEdit]);

  const onDelete = React.useCallback(async () => {
    if (!productionId || !production || deleting) return;

    if (production.printed) {
      alert("印刷済みの生産は削除できません。");
      return;
    }

    try {
      setDeleting(true);
      const repository = new ProductionRepositoryHTTP();
      await repository.delete(productionId);
      navigate("/production");
    } catch {
      alert("生産情報の削除に失敗しました。");
    } finally {
      setDeleting(false);
    }
  }, [productionId, production, deleting, navigate]);

  const handleBack = React.useCallback(() => {
    navigate("/production");
  }, [navigate]);

  return {
    isViewMode,
    isEditMode,
    switchToView,
    switchToEdit,
    canEdit,
    adminMode: mode,
    onBack: handleBack,
    onSave,
    onDelete,
    deleting,
    productionId: productionId ?? null,
    production,
    loading,
    error,
    quantityRows,
    setQuantityRows,
  };
}

export default useProductionDetail;