// frontend/console/shell/src/features/production/presentation/hook/useProductionDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useAuthContext } from "../../../../auth/application/AuthContext";

import {
  loadProductionDetail,
  updateProductionDetail,
  type ProductionDetail,
} from "../../application/detail/index";

import { ProductionRepositoryHTTP } from "../../infrastructure/http/productionRepositoryHTTP";
import { getProductBlueprintDetail } from "../../../productBlueprint/application/productBlueprintDetailService";

import type { ProductionQuantityRowVM } from "../viewModels/productionQuantityRowVM";

type Mode = "view" | "edit";

type ProductBlueprintDetailForProduction = Awaited<
  ReturnType<typeof getProductBlueprintDetail>
>;

export function useProductionDetail() {
  const navigate = useNavigate();
  const { productionId } = useParams<{ productionId: string }>();
  const { currentMember } = useAuthContext();

  const creator = currentMember?.displayName ?? "-";

  const [production, setProduction] = React.useState<ProductionDetail | null>(null);
  const [mode, setMode] = React.useState<Mode>("view");

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

  const adminMode: "view" | "edit" = mode;

  const [loading, setLoading] = React.useState(false);
  const [deleting, setDeleting] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const [productBlueprint, setProductBlueprint] =
    React.useState<ProductBlueprintDetailForProduction | null>(null);

  const [pbLoading, setPbLoading] = React.useState(false);
  const [pbError, setPbError] = React.useState<string | null>(null);

  const [quantityRows, setQuantityRows] = React.useState<ProductionQuantityRowVM[]>([]);

  // ======================================================
  // Production詳細取得
  // ======================================================
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
        setProductBlueprint(null);
        setPbError(null);
        setQuantityRows([]);

        const data = await loadProductionDetail(id);

        if (cancelled) {
          return;
        }

        setProduction(data);

        if (!data) {
          setQuantityRows([]);
          return;
        }

        setQuantityRows(
          data.models.map((row) => ({
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
          })),
        );
      } catch {
        if (cancelled) {
          return;
        }

        setError("生産情報の取得に失敗しました");
        setProduction(null);
        setQuantityRows([]);
        setProductBlueprint(null);
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

  // ======================================================
  // ProductBlueprint詳細取得
  // ======================================================
  React.useEffect(() => {
    const productBlueprintId = production?.productBlueprintId;

    if (!productBlueprintId) {
      setProductBlueprint(null);
      setPbError(null);
      return;
    }

    const targetProductBlueprintId = productBlueprintId;
    let cancelled = false;

    async function loadProductBlueprint(id: string) {
      try {
        setPbLoading(true);
        setPbError(null);

        const data = await getProductBlueprintDetail(id);

        if (cancelled) {
          return;
        }

        setProductBlueprint(data);
      } catch {
        if (cancelled) {
          return;
        }

        setPbError("商品設計情報の取得に失敗しました");
        setProductBlueprint(null);
      } finally {
        if (!cancelled) {
          setPbLoading(false);
        }
      }
    }

    void loadProductBlueprint(targetProductBlueprintId);

    return () => {
      cancelled = true;
    };
  }, [production?.productBlueprintId]);

  // ======================================================
  // 保存処理
  // ======================================================
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

        setQuantityRows(
          updated.models.map((row) => ({
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
          })),
        );
      }

      setMode("view");
    } catch {
      alert("更新に失敗しました");
    }
  }, [productionId, production, quantityRows, canEdit]);

  // ======================================================
  // 削除処理
  // ======================================================
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

  // ======================================================
  // 戻る
  // ======================================================
  const handleBack = React.useCallback(() => {
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
    onDelete,
    deleting,
    productionId: productionId ?? null,
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