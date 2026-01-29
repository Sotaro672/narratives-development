// frontend/console/production/src/presentation/hook/useProductionDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

import {
  loadProductionDetail,
  loadModelVariationIndexByProductBlueprintId,
  buildQuantityRowsFromModels,
  updateProductionDetail,
  type ProductionDetail,
  type ModelVariationSummary,
  type ProductionQuantityRow as DetailQuantityRow,
} from "../../application/detail/index";

import {
  loadProductBlueprintDetail,
  type ProductBlueprintDetail,
} from "../../application/productBlueprint/index";

// ★ 印刷用ロジックを分離した hook を利用
import { usePrintCard } from "../../../../product/src/presentation/hook/usePrintCard";

// ★ domain の ProductionStatus 型を import
import type {
  ProductionStatus as DomainProductionStatus,
} from "../../../../production/src/domain/entity/production";

type Mode = "view" | "edit";

// ★ 編集可能なステータス（domain 型に基づく）
//   status が "planned" のときだけ編集可能にする
const EDITABLE_STATUS: DomainProductionStatus = "planned";

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

  // ★ status が planned のときだけ編集可能
  const canEdit = production?.status === EDITABLE_STATUS;

  const switchToView = React.useCallback(() => setMode("view"), []);

  const switchToEdit = React.useCallback(() => {
    if (!canEdit) {
      // eslint-disable-next-line no-console
      console.log(
        "[useProductionDetail] switchToEdit called but production is not editable (status is not 'planned')",
        { status: production?.status },
      );
      return;
    }
    setMode("edit");
  }, [canEdit, production?.status]);

  const toggleMode = React.useCallback(() => {
    if (!canEdit) {
      // eslint-disable-next-line no-console
      console.log(
        "[useProductionDetail] toggleMode called but production is not editable (status is not 'planned')",
        { status: production?.status },
      );
      return;
    }
    setMode((prev) => (prev === "view" ? "edit" : "view"));
  }, [canEdit, production?.status]);

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

  // ✅ rows は detail DTO（dto/detail.go 正）に統一
  const [quantityRows, setQuantityRows] = React.useState<DetailQuantityRow[]>(
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
  // production.models × modelIndex → quantityRows
  //
  // ✅ 重要:
  // - dto/detail.go を正としたいが、現状 backend が PascalCase(ModelID/Quantity) を返している
  // - ここで modelId/quantity に正規化して buildQuantityRowsFromModels に渡す
  // ======================================================
  React.useEffect(() => {
    const raw = (production as any)?.models;

    if (!raw || !Array.isArray(raw)) {
      setQuantityRows([]);
      return;
    }

    // ✅ backend の揺れ吸収: modelId / ModelID / ModelId など
    const normalized = raw.map((m: any, index: number) => {
      const modelIdRaw =
        m?.modelId ?? m?.ModelID ?? m?.ModelId ?? m?.modelID ?? "";
      const quantityRaw = m?.quantity ?? m?.Quantity ?? 0;

      const modelId = String(modelIdRaw ?? "").trim() || String(index);

      const quantity = Number.isFinite(Number(quantityRaw))
        ? Math.max(0, Math.floor(Number(quantityRaw)))
        : 0;

      const modelNumber =
        typeof (m?.modelNumber ?? m?.ModelNumber) === "string"
          ? (m?.modelNumber ?? m?.ModelNumber)
          : undefined;

      const size =
        typeof (m?.size ?? m?.Size) === "string" ? (m?.size ?? m?.Size) : undefined;

      const color =
        typeof (m?.color ?? m?.Color) === "string" ? (m?.color ?? m?.Color) : undefined;

      const rgbCandidate = m?.rgb ?? m?.RGB;
      const rgb = typeof rgbCandidate === "number" ? rgbCandidate : undefined;

      const displayOrderCandidate = m?.displayOrder ?? m?.DisplayOrder;
      const displayOrder =
        typeof displayOrderCandidate === "number"
          ? displayOrderCandidate
          : undefined;

      return {
        modelId,
        quantity,
        modelNumber,
        size,
        color,
        rgb,
        displayOrder,
      };
    });

    const detailRows: DetailQuantityRow[] = buildQuantityRowsFromModels(
      normalized,
      modelIndex,
    );

    setQuantityRows(detailRows);
  }, [production, modelIndex]);

  // ======================================================
  // 保存処理（quantity + assigneeId）
  // ======================================================
  const onSave = React.useCallback(async () => {
    if (!productionId || !production) return;

    // status が planned 以外なら保存も不可
    if (!canEdit) {
      // eslint-disable-next-line no-alert
      alert("この生産は編集できません（ステータスが planned ではありません）。");
      return;
    }

    try {
      // dto/detail.go を正: rows は modelId ベース
      const rowsForUpdate: DetailQuantityRow[] = quantityRows.map((row) => ({
        modelId: row.modelId,
        modelNumber: row.modelNumber,
        size: row.size,
        color: row.color,
        rgb: row.rgb ?? null,
        displayOrder: row.displayOrder,
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
      // eslint-disable-next-line no-alert
      alert("更新に失敗しました");
    }
  }, [productionId, production, quantityRows, canEdit]);

  // ======================================================
  // 印刷時 Product 作成処理は usePrintCard に委譲
  // ======================================================
  const { onPrint } = usePrintCard({
    productionId: productionId ?? null,
    hasProduction: !!production,
    // ✅ usePrintCard が QuantityRowBase(modelVariationId) を要求する場合があるため、
    //   最低限ここでアダプトして渡す（productionDetail.tsx 側と同等）
    rows: (Array.isArray(quantityRows) ? quantityRows : []).map((r) => ({
      modelVariationId: r.modelId,
      modelNumber: r.modelNumber,
      size: r.size,
      color: r.color,
      rgb: r.rgb ?? null,
      quantity: r.quantity ?? 0,
    })) as any,
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

    // ★ 画面側で header の編集ボタン表示可否に使う
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

    quantityRows,
    setQuantityRows,

    creator,
  };
}
