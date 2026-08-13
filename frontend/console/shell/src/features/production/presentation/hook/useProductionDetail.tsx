// frontend/console/shell/src/features/production/presentation/hook/useProductionDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import {
  loadProductionDetail,
  updateProductionDetail,
  type ProductionDetail,
} from "../../application/detail/index";

import { ProductionRepositoryHTTP } from "../../infrastructure/http/productionRepositoryHTTP";
import type { ProductionQuantityRowVM } from "../viewModels/productionQuantityRowVM";

type Mode = "view" | "edit";

function toQuantityRows(production: ProductionDetail): ProductionQuantityRowVM[] {
  return production.models.map((row) => ({
    modelId: row.modelId,
    kind: row.kind,
    modelNumber: row.modelNumber,
    size: row.size,
    color: row.color,
    rgb: typeof row.rgb === "number" ? row.rgb : undefined,
    volumeValue: row.volumeValue,
    volumeUnit: row.volumeUnit,
    variationLabel: row.variationLabel,
    displayOrder: row.displayOrder,
    quantity: row.quantity,
  }));
}

export function useProductionDetail() {
  const navigate = useNavigate();
  const { productionId } = useParams<{ productionId: string }>();

  const [production, setProduction] = React.useState<ProductionDetail | null>(null);
  const [mode, setMode] = React.useState<Mode>("view");
  const [loading, setLoading] = React.useState(false);
  const [deleting, setDeleting] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [quantityRows, setQuantityRows] = React.useState<ProductionQuantityRowVM[]>([]);

  const isViewMode = mode === "view";
  const isEditMode = mode === "edit";
  const canEdit = production?.printed !== true;

  const switchToView = React.useCallback(() => {
    setMode("view");
  }, []);

  const switchToEdit = React.useCallback(() => {
    if (!canEdit) {
      return;
    }

    setMode("edit");
  }, [canEdit]);

  React.useEffect(() => {
    if (!productionId) {
      return;
    }

    const targetProductionId = productionId;
    let cancelled = false;

    async function loadDetail(id: string) {
      try {
        setLoading(true);
        setError(null);
        setQuantityRows([]);

        const data = await loadProductionDetail(id);

        if (cancelled) {
          return;
        }

        setProduction(data);
        setQuantityRows(data ? toQuantityRows(data) : []);
      } catch {
        if (cancelled) {
          return;
        }

        setError("生産情報の取得に失敗しました");
        setProduction(null);
        setQuantityRows([]);
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void loadDetail(targetProductionId);

    return () => {
      cancelled = true;
    };
  }, [productionId]);

  const onSave = React.useCallback(async () => {
    if (!productionId || !production) {
      return;
    }

    if (!canEdit) {
      alert("この生産は編集できません（印刷済みです）。");
      return;
    }

    try {
      const updated = await updateProductionDetail({
        productionId,
        rows: quantityRows,
        assigneeId: production.assigneeId ?? null,
      });

      if (updated) {
        setProduction(updated);
        setQuantityRows(toQuantityRows(updated));
      }

      setMode("view");
    } catch {
      alert("更新に失敗しました");
    }
  }, [productionId, production, quantityRows, canEdit]);

  const onDelete = React.useCallback(async () => {
    if (!productionId || !production || deleting) {
      return;
    }

    if (production.printed === true) {
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